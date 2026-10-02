package db

import (
	"context"
	"testing"
)

func TestConstraintErrors(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	if _, err := d.ExecContext(ctx, `CREATE TABLE p (id INTEGER PRIMARY KEY);
		CREATE TABLE c (u TEXT UNIQUE, p INTEGER REFERENCES p (id))`); err != nil {
		t.Fatal(err)
	}
	_, fk := d.ExecContext(ctx, `INSERT INTO c (p) VALUES (9)`)
	d.ExecContext(ctx, `INSERT INTO c (u) VALUES ('a')`)
	_, uq := d.ExecContext(ctx, `INSERT INTO c (u) VALUES ('a')`)
	if !IsForeignKey(fk) || IsUnique(fk) {
		t.Errorf("foreign key error not detected: %v", fk)
	}
	if !IsUnique(uq) || IsForeignKey(uq) {
		t.Errorf("unique error not detected: %v", uq)
	}
	if IsUnique(nil) || IsForeignKey(nil) {
		t.Error("nil is not a constraint error")
	}
}
