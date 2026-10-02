// Package content stores the study material: syllabus, sources and
// questions. It owns the validation rules that keep every question traceable
// to a real source.
package content

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alexzafra13/tai_tests/internal/db"
)

var ErrNotFound = errors.New("not found")

// ErrInUse is returned when deleting a record that others still reference.
var ErrInUse = errors.New("in use")

// ErrHasHistory is returned when deleting a question that has been
// answered; it should be discarded instead so stats keep its attempts.
var ErrHasHistory = errors.New("question has answer history")

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

func (s *Store) timestamp() string { return db.Timestamp(s.now()) }

// inTx runs fn in a transaction that is committed only if fn succeeds.
func (s *Store) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
