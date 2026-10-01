// Package srs schedules spaced repetition with FSRS. Each user has one card
// per question they have answered; every answer reschedules it. The rating
// FSRS needs is derived from the answer itself, so studying never asks
// "how well did you know it?".
package srs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// Rating is how well a question was known, in FSRS terms.
type Rating = fsrs.Rating

const (
	Again = fsrs.Again
	Hard  = fsrs.Hard
	Good  = fsrs.Good
	Easy  = fsrs.Easy
)

// QuickAnswer is the time under which a correct answer counts as easy.
const QuickAnswer = 5 * time.Second

// RatingFor maps an answer to a rating: wrong or blank is Again, correct
// but reported as doubtful is Hard, correct and quick is Easy, any other
// correct answer is Good.
func RatingFor(correct *bool, spent time.Duration, doubtful bool) Rating {
	switch {
	case correct == nil || !*correct:
		return Again
	case doubtful:
		return Hard
	case spent > 0 && spent < QuickAnswer:
		return Easy
	default:
		return Good
	}
}

const timeFormat = "2006-01-02T15:04:05.000Z"

type Store struct {
	db        *sql.DB
	scheduler *fsrs.FSRS
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, scheduler: fsrs.NewFSRS(Parameters())}
}

// Parameters are the FSRS defaults with intervals in whole days: the
// short-term learning steps (minutes) suit flashcards, not test questions,
// where a failed question can be retried at once from the results. Fuzz
// spreads due dates so reviews do not pile up on the same day.
func Parameters() fsrs.Parameters {
	p := fsrs.DefaultParam()
	p.EnableShortTerm = false
	p.EnableFuzz = true
	return p
}

// Record reschedules the user's card for a question after an answer given
// at time at.
func (s *Store) Record(ctx context.Context, userID, questionID int64, rating Rating, at time.Time) error {
	card := fsrs.NewCard()
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT card FROM review_cards WHERE user_id = ? AND question_id = ?`,
		userID, questionID).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return err
	default:
		if err := json.Unmarshal([]byte(raw), &card); err != nil {
			return err
		}
	}

	next := s.scheduler.Next(card, at.UTC(), rating).Card
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO review_cards (user_id, question_id, due, card, reps, lapses, last_rating, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, question_id) DO UPDATE SET due = excluded.due, card = excluded.card, reps = excluded.reps,
			lapses = excluded.lapses, last_rating = excluded.last_rating, updated_at = excluded.updated_at`,
		userID, questionID, next.Due.UTC().Format(timeFormat), string(b), next.Reps, next.Lapses, int(rating),
		at.UTC().Format(timeFormat))
	return err
}

// DueCondition is the SQL condition (on questions aliased q) for questions
// whose card is due for the user at the given time, with its arguments.
func DueCondition(userID int64, now time.Time) (string, []any) {
	return `q.id IN (SELECT question_id FROM review_cards WHERE user_id = ? AND due <= ?)`,
		[]any{userID, now.UTC().Format(timeFormat)}
}

// DueOrder orders questions by how overdue their card is (most first).
func DueOrder(userID int64) (string, []any) {
	return `(SELECT due FROM review_cards WHERE user_id = ? AND question_id = q.id)`, []any{userID}
}

// Summary describes a user's review workload.
type Summary struct {
	// Due is how many published questions are due now.
	Due int `json:"due"`
	// Tracked is how many questions have a card (answered at least once).
	Tracked int `json:"tracked"`
	// Mastered counts cards scheduled three weeks or more ahead.
	Mastered int `json:"mastered"`
}

func (s *Store) Summary(ctx context.Context, userID int64, now time.Time) (Summary, error) {
	var sm Summary
	at := now.UTC()
	err := s.db.QueryRowContext(ctx, `SELECT
			count(CASE WHEN c.due <= ? THEN 1 END),
			count(*),
			count(CASE WHEN c.due >= ? THEN 1 END)
		FROM review_cards c JOIN questions q ON q.id = c.question_id
		WHERE c.user_id = ? AND q.status = 'published' AND q.annulled = 0`,
		at.Format(timeFormat), at.Add(21*24*time.Hour).Format(timeFormat), userID).
		Scan(&sm.Due, &sm.Tracked, &sm.Mastered)
	return sm, err
}
