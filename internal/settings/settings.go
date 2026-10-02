// Package settings persists user settings as JSON values by key. Each
// feature owns its keys, types and defaults; this package only stores them.
package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/alexzafra13/tai_tests/internal/db"
)

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

// Get decodes the value stored under key into v. It reports false, leaving
// v untouched, when nothing is stored, so callers can pre-fill defaults.
func (s *Store) Get(ctx context.Context, key string, v any) (bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(raw), v)
}

func (s *Store) Set(ctx context.Context, key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, string(raw), db.Timestamp(s.now()))
	return err
}
