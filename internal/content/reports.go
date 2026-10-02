package content

import (
	"context"
	"database/sql"
	"strings"

	"github.com/alexzafra13/tai_tests/internal/db"
)

// A report is a user's doubt about a question ("I think the answer is
// wrong"), raised during a test. Each user has at most one open report per
// question; reviewing the question resolves all of them.

// hasOpenReports is the SQL condition for a question with open reports.
const hasOpenReports = `EXISTS (SELECT 1 FROM question_reports r WHERE r.question_id = q.id AND r.resolved_at = '')`

type Report struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

// SetReport opens (or updates the note of) a user's report on a question,
// or withdraws it when open is false.
func (s *Store) SetReport(ctx context.Context, questionID, userID int64, open bool, note string) error {
	if !open {
		_, err := s.db.ExecContext(ctx, `DELETE FROM question_reports
			WHERE question_id = ? AND user_id = ? AND resolved_at = ''`, questionID, userID)
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO question_reports (question_id, user_id, note, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (question_id, user_id) WHERE resolved_at = '' DO UPDATE SET note = excluded.note`,
		questionID, userID, strings.TrimSpace(note), s.timestamp())
	if db.IsForeignKey(err) {
		return ErrNotFound
	}
	return err
}

// openReports returns the open reports of each question, never nil slices.
func (s *Store) openReports(ctx context.Context, questionIDs []int64) (map[int64][]Report, error) {
	out := make(map[int64][]Report, len(questionIDs))
	if len(questionIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(questionIDs))
	for i, id := range questionIDs {
		args[i] = id
		out[id] = []Report{}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.question_id, r.id, r.user_id, u.username, r.note, r.created_at
		FROM question_reports r JOIN users u ON u.id = r.user_id
		WHERE r.resolved_at = '' AND r.question_id IN (`+placeholders(len(questionIDs))+`)
		ORDER BY r.created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var qid int64
		var r Report
		if err := rows.Scan(&qid, &r.ID, &r.UserID, &r.Username, &r.Note, &r.CreatedAt); err != nil {
			return nil, err
		}
		out[qid] = append(out[qid], r)
	}
	return out, rows.Err()
}

// resolveReports closes the open reports of a question and returns their
// ids, so the decision can be undone.
func (s *Store) resolveReports(ctx context.Context, tx *sql.Tx, questionID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `UPDATE question_reports SET resolved_at = ?
		WHERE question_id = ? AND resolved_at = '' RETURNING id`, s.timestamp(), questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
