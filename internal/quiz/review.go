package quiz

import (
	"context"
	"database/sql"
	"time"

	"github.com/alexzafra13/tai_tests/internal/srs"
)

// schedule feeds one answer to the spaced-repetition schedule. isCorrect
// is nil for a blank answer.
func (s *Store) schedule(ctx context.Context, userID, questionID int64, isCorrect *bool, timeMs int) error {
	var reported bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM question_reports
		WHERE question_id = ? AND user_id = ? AND resolved_at = '')`, questionID, userID).Scan(&reported); err != nil {
		return err
	}
	rating := srs.RatingFor(isCorrect, time.Duration(timeMs)*time.Millisecond, reported)
	return s.srs.Record(ctx, userID, questionID, rating, s.now())
}

// scheduleExam feeds every question of a finished exam to the schedule.
// Blank answers count as not known: leaving a question blank in an exam
// means it was not known well enough to risk the penalty.
func (s *Store) scheduleExam(ctx context.Context, userID, testID int64) error {
	rows, err := s.db.QueryContext(ctx, `SELECT question_id, is_correct, time_ms FROM attempts WHERE test_id = ?`, testID)
	if err != nil {
		return err
	}
	type answer struct {
		questionID int64
		isCorrect  *bool
		timeMs     int
	}
	var answers []answer
	for rows.Next() {
		var a answer
		var ok sql.NullBool
		if err := rows.Scan(&a.questionID, &ok, &a.timeMs); err != nil {
			rows.Close()
			return err
		}
		if ok.Valid {
			a.isCorrect = &ok.Bool
		}
		answers = append(answers, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, a := range answers {
		if err := s.schedule(ctx, userID, a.questionID, a.isCorrect, a.timeMs); err != nil {
			return err
		}
	}
	return nil
}
