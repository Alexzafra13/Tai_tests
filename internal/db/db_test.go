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

// Question search depends on FTS5 with accent folding; make sure the pure-Go
// driver ships it.
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

// Migration 0003 rebuilds the questions table; child rows in
// question_topics must survive (a DROP TABLE with foreign keys on would
// cascade-delete them).
func TestRebuildKeepsChildRows(t *testing.T) {
	ctx := context.Background()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if _, err := migrateTo(ctx, d, 2); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`INSERT INTO blocks (id, code, name, position) VALUES (1, 'B1', 'b', 0)`,
		`INSERT INTO topics (id, block_id, code, number, title, position) VALUES (1, 1, 'T1', 1, 't', 0)`,
		`INSERT INTO sources (id, kind, title, created_at, updated_at) VALUES (1, 'inap_exam', 'e', '', '')`,
		`INSERT INTO questions (id, stem, option_a, option_b, option_c, option_d, correct, origin, author,
			source_id, source_ref, created_at, updated_at) VALUES (7, 's', 'a', 'b', 'c', 'd', 0, 'official', 'import', 1, '2024 · 1', '', '')`,
		`INSERT INTO question_topics (question_id, topic_id) VALUES (7, 1)`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}

	if _, err := Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	var links, revision int
	if err := d.QueryRow(`SELECT count(*) FROM question_topics WHERE question_id = 7`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Fatalf("question_topics rows after rebuild = %d, want 1", links)
	}
	if err := d.QueryRow(`SELECT revision FROM questions WHERE id = 7`).Scan(&revision); err != nil || revision != 1 {
		t.Fatalf("revision = %d, %v", revision, err)
	}
	var fk int
	d.QueryRow(`PRAGMA foreign_keys`).Scan(&fk)
	if fk != 1 {
		t.Errorf("foreign_keys left off after rebuild")
	}
	if _, err := d.Exec(`UPDATE questions SET status = 'discarded' WHERE id = 7`); err != nil {
		t.Errorf("discarded status rejected: %v", err)
	}
}

// Migration 0005 introduces users: existing tests and doubt flags must end
// up owned by the first administrator, with attempts intact.
func TestUsersMigrationKeepsData(t *testing.T) {
	ctx := context.Background()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if _, err := migrateTo(ctx, d, 4); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`INSERT INTO sources (id, kind, title, created_at, updated_at) VALUES (1, 'inap_exam', 'e', '', '')`,
		`INSERT INTO questions (id, stem, option_a, option_b, option_c, option_d, correct, origin, author, source_id,
			source_ref, status, flagged, flag_note, created_at, updated_at)
			VALUES (7, 's', 'a', 'b', 'c', 'd', 0, 'official', 'import', 1, '2024 · 1', 'published', 1, 'revisar', '', 'T')`,
		`INSERT INTO tests (id, mode, penalty, started_at) VALUES (3, 'exam', 0.25, 'S')`,
		`INSERT INTO attempts (test_id, position, question_id, revision, chosen, is_correct) VALUES (3, 0, 7, 1, 0, 1)`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}

	if _, err := Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	var owner, attempts int
	d.QueryRow(`SELECT user_id FROM tests WHERE id = 3`).Scan(&owner)
	d.QueryRow(`SELECT count(*) FROM attempts WHERE test_id = 3`).Scan(&attempts)
	if owner != 1 || attempts != 1 {
		t.Errorf("test owner = %d, attempts = %d; want 1, 1", owner, attempts)
	}
	var note string
	var reporter int
	if err := d.QueryRow(`SELECT user_id, note FROM question_reports WHERE question_id = 7 AND resolved_at = ''`).Scan(&reporter, &note); err != nil || reporter != 1 || note != "revisar" {
		t.Errorf("migrated report: user %d note %q err %v", reporter, note, err)
	}
	if _, err := d.Exec(`SELECT flagged FROM questions`); err == nil {
		t.Error("questions.flagged still exists")
	}
	var role, hash string
	d.QueryRow(`SELECT role, password_hash FROM users WHERE id = 1`).Scan(&role, &hash)
	if role != "admin" || hash != "" {
		t.Errorf("placeholder admin: role %q hash %q", role, hash)
	}
}
