package quiz

import (
	"context"
	"database/sql"
)

type Summary struct {
	ID         int64    `json:"id"`
	Mode       Mode     `json:"mode"`
	Status     Status   `json:"status"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at,omitempty"`
	Deadline   string   `json:"deadline,omitempty"`
	Total      int      `json:"total"`
	Answered   int      `json:"answered"`
	Correct    int      `json:"correct"`
	Wrong      int      `json:"wrong"`
	Score      *float64 `json:"score"` // on the configured scale; nil unless finished
	Passed     bool     `json:"passed"`
}

// List returns the user's recent tests, newest first, optionally filtered by
// status.
func (s *Store) List(ctx context.Context, userID int64, status Status, limit int) ([]Summary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	scoring, err := s.Scoring(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.mode, t.status, t.started_at, t.finished_at, t.deadline, t.score,
			count(a.position), count(a.chosen), count(CASE WHEN a.is_correct = 1 THEN 1 END),
			count(CASE WHEN a.is_correct = 0 THEN 1 END)
		FROM tests t JOIN attempts a ON a.test_id = t.id
		WHERE t.user_id = ? AND (? = '' OR t.status = ?)
		GROUP BY t.id ORDER BY t.id DESC LIMIT ?`, userID, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var sm Summary
		var ratio sql.NullFloat64
		if err := rows.Scan(&sm.ID, &sm.Mode, &sm.Status, &sm.StartedAt, &sm.FinishedAt, &sm.Deadline, &ratio,
			&sm.Total, &sm.Answered, &sm.Correct, &sm.Wrong); err != nil {
			return nil, err
		}
		if ratio.Valid {
			r := Result{Ratio: ratio.Float64}
			r.applyScale(scoring.Scale)
			sm.Score, sm.Passed = &r.Score, r.Passed
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}
