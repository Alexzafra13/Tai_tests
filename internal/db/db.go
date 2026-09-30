// Package db opens the SQLite database and applies schema migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens (or creates) the database at path with the pragmas the app
// relies on: WAL for concurrent reads, foreign keys, and a busy timeout.
// Use ":memory:" for tests.
func Open(path string) (*sql.DB, error) {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_txlock", "immediate")

	memory := path == ":memory:"
	if !memory {
		q.Add("_pragma", "journal_mode(WAL)")
		q.Add("_pragma", "synchronous(NORMAL)")
	}

	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	if memory {
		// Each connection would get its own empty in-memory database.
		db.SetMaxOpenConns(1)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return db, nil
}

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	var ms []migration
	seen := map[int]string{}
	for _, e := range entries {
		name := e.Name()
		prefix, _, ok := strings.Cut(name, "_")
		if !ok || !strings.HasSuffix(name, ".sql") {
			return nil, fmt.Errorf("migration %q: expected NNNN_name.sql", name)
		}
		v, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("migration %q: bad version: %w", name, err)
		}
		if other, dup := seen[v]; dup {
			return nil, fmt.Errorf("migrations %q and %q share version %d", other, name, v)
		}
		seen[v] = name
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		ms = append(ms, migration{version: v, name: name, sql: string(body)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	return ms, nil
}

// Migrate applies pending migrations in order, each in its own transaction,
// and returns the names of the ones applied.
func Migrate(ctx context.Context, db *sql.DB) ([]string, error) {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`); err != nil {
		return nil, err
	}

	ms, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return nil, err
	}

	var applied []string
	for _, m := range ms {
		if m.version <= current {
			continue
		}
		if err := apply(ctx, db, m); err != nil {
			return applied, fmt.Errorf("migration %s: %w", m.name, err)
		}
		applied = append(applied, m.name)
	}
	return applied, nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
		return err
	}
	return tx.Commit()
}
