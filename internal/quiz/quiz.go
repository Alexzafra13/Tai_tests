// Package quiz runs test sessions: it picks questions, records every answer
// as an attempt and scores the result.
//
// Files: create.go builds tests from filters, session.go answers and
// finishes them, history.go lists them, score.go and scoring.go compute and
// configure marks, shuffle.go orders the options.
package quiz

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/settings"
)

type Mode string

const (
	// ModePractice reveals the answer, explanation and source right after
	// each question. Answers are final.
	ModePractice Mode = "practice"
	// ModeExam reveals nothing until the end; answers can be changed or
	// left blank, and an optional time limit applies.
	ModeExam Mode = "exam"
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusFinished   Status = "finished"
	StatusAbandoned  Status = "abandoned"
)

var (
	ErrNotFound        = errors.New("test not found")
	ErrNotInProgress   = errors.New("test is not in progress")
	ErrExpired         = errors.New("time is up")
	ErrAlreadyAnswered = errors.New("question already answered")
	ErrNoQuestions     = errors.New("no questions match the filters")
)

type Store struct {
	db       *sql.DB
	settings *settings.Store
	now      func() time.Time
	// shuffle orders options; replaced in tests for determinism.
	shuffle func(n int, swap func(i, j int))
}

func NewStore(db *sql.DB, st *settings.Store) *Store {
	return &Store{db: db, settings: st, now: time.Now, shuffle: randomShuffle}
}

const timeFormat = "2006-01-02T15:04:05.000Z"

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func appendIDs(args []any, ids []int64) []any {
	for _, id := range ids {
		args = append(args, id)
	}
	return args
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
