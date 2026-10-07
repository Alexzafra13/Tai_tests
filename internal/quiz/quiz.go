// Package quiz builds tests, records each answer as an attempt and scores
// the result. Every operation is scoped to the user id it is given; answers
// also feed the spaced-repetition schedule (package srs).
package quiz

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/settings"
	"github.com/alexzafra13/tai_tests/internal/srs"
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
	srs      *srs.Store
	// content finds the law articles questions cite, for the solutions.
	content *content.Store
	now     func() time.Time
	// shuffle orders options; replaced in tests for determinism.
	shuffle func(n int, swap func(i, j int))
}

func NewStore(db *sql.DB, st *settings.Store, sr *srs.Store) *Store {
	return &Store{db: db, settings: st, srs: sr, content: content.NewStore(db), now: time.Now, shuffle: randomShuffle}
}

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
