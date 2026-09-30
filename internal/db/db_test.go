package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	first, err := Migrate(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 {
		t.Fatal("expected migrations to be applied on a fresh database")
	}

	second, err := Migrate(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("expected no migrations on second run, got %v", second)
	}
}

func TestPragmas(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	var fk int
	if err := d.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
	var mode string
	if err := d.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

// Search (phase 7) depends on FTS5 with accent folding; make sure the
// pure-Go driver ships it.
func TestFTS5WithDiacriticFolding(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if _, err := d.Exec(`CREATE VIRTUAL TABLE t USING fts5(body, tokenize = 'unicode61 remove_diacritics 2')`); err != nil {
		t.Fatalf("fts5 unavailable: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO t (body) VALUES ('Ley de Protección de Datos y administración electrónica')`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.QueryRow(`SELECT count(*) FROM t WHERE t MATCH 'proteccion AND administracion'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("accent-insensitive match returned %d rows, want 1", n)
	}
}
