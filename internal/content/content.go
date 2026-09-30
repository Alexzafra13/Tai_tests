// Package content stores the study material: syllabus, sources and
// questions. It owns the validation rules that keep every question traceable
// to a real source.
package content

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// ErrInUse is returned when deleting a record that others still reference.
var ErrInUse = errors.New("in use")

// ErrHasHistory is returned when deleting a question that has been
// answered; it should be discarded instead so stats keep its attempts.
var ErrHasHistory = errors.New("question has answer history")

// ValidationError maps field names to human-readable (Spanish) messages
// shown next to the form fields.
type ValidationError map[string]string

func (v ValidationError) Error() string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + v[k]
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

func (v ValidationError) orNil() error {
	if len(v) == 0 {
		return nil
	}
	return v
}

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

const timeFormat = "2006-01-02T15:04:05.000Z"

func (s *Store) timestamp() string {
	return s.now().UTC().Format(timeFormat)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
