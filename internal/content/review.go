package content

import (
	"context"
	"errors"

	"github.com/alexzafra13/tai_tests/internal/textmatch"
	"github.com/alexzafra13/tai_tests/internal/validate"
)

// The review queue holds what is not ready for tests yet: drafts (imported
// or generated questions) and questions users reported as doubtful during
// a test. Reported ones come first, since they were raised while studying.

type ReviewKind string

const (
	ReviewAll      ReviewKind = ""
	ReviewReported ReviewKind = "reported"
	ReviewDrafts   ReviewKind = "drafts"
)

func (k ReviewKind) where() string {
	switch k {
	case ReviewReported:
		return hasOpenReports + " AND q.status <> 'discarded'"
	case ReviewDrafts:
		return "q.status IN ('draft', 'reviewed')"
	default:
		return "(q.status IN ('draft', 'reviewed') OR " + hasOpenReports + ") AND q.status <> 'discarded'"
	}
}

// ExcerptWindow is how much source text is shown around a quote.
const ExcerptWindow = 300

type ReviewItem struct {
	Question
	Reports []Report `json:"reports"`
	// Excerpt shows the quote inside its source text; nil when there is no
	// quote or it cannot be found (which validation normally prevents).
	Excerpt *textmatch.Excerpt `json:"excerpt"`
}

type ReviewPage struct {
	Items []ReviewItem `json:"items"`
	Total int          `json:"total"`
}

type ReviewCounts struct {
	Reported int `json:"reported"`
	Drafts   int `json:"drafts"`
	Total    int `json:"total"`
}

func (s *Store) ReviewCounts(ctx context.Context) (ReviewCounts, error) {
	var c ReviewCounts
	err := s.db.QueryRowContext(ctx, `SELECT
			count(CASE WHEN `+ReviewReported.where()+` THEN 1 END),
			count(CASE WHEN `+ReviewDrafts.where()+` THEN 1 END),
			count(CASE WHEN `+ReviewAll.where()+` THEN 1 END)
		FROM questions q`).Scan(&c.Reported, &c.Drafts, &c.Total)
	return c, err
}

// ReviewQueue returns a page of the queue, optionally limited to one source
// (e.g. the exam just imported).
func (s *Store) ReviewQueue(ctx context.Context, kind ReviewKind, sourceID int64, offset, limit int) (ReviewPage, error) {
	cond := kind.where()
	var args []any
	if sourceID != 0 {
		cond += " AND q.source_id = ?"
		args = append(args, sourceID)
	}
	page := ReviewPage{Items: []ReviewItem{}}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM questions q WHERE `+cond, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `SELECT `+questionColumns+`, s.full_text
		FROM questions q JOIN sources s ON s.id = q.source_id
		WHERE `+cond+` ORDER BY `+hasOpenReports+` DESC, q.id LIMIT ? OFFSET ?`, append(args, limit, max(offset, 0))...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var it ReviewItem
		var sourceText string
		if err := rows.Scan(append(questionFields(&it.Question), &sourceText)...); err != nil {
			return page, err
		}
		if ex, ok := textmatch.Locate(sourceText, it.SourceQuote, ExcerptWindow); ok {
			it.Excerpt = &ex
		}
		page.Items = append(page.Items, it)
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	rows.Close()

	topics, err := s.topicIDs(ctx, ids)
	if err != nil {
		return page, err
	}
	reports, err := s.openReports(ctx, ids)
	if err != nil {
		return page, err
	}
	for i := range page.Items {
		page.Items[i].TopicIDs = topics[page.Items[i].ID]
		page.Items[i].Reports = reports[page.Items[i].ID]
	}
	return page, nil
}

// ReviewState is what a review decision changed, returned so the decision
// can be undone.
type ReviewState struct {
	Status Status `json:"status"`
	// ResolvedReports are the reports the decision closed.
	ResolvedReports []int64 `json:"resolved_reports"`
}

// Accept publishes a question and resolves its reports. It goes through the
// normal validation, so a question without topics or with a quote that is
// not in its source cannot be published. topicIDs, when given, replace the
// question's topics first.
func (s *Store) Accept(ctx context.Context, id int64, topicIDs []int64) (ReviewState, error) {
	q, err := s.Question(ctx, id)
	if err != nil {
		return ReviewState{}, err
	}
	in := q.Input()
	if len(topicIDs) > 0 {
		in.TopicIDs = topicIDs
	}
	in.Status = StatusPublished
	if err := s.UpdateQuestion(ctx, id, in); err != nil {
		return ReviewState{}, err
	}
	resolved, err := s.resolveReports(ctx, id)
	return ReviewState{Status: q.Status, ResolvedReports: resolved}, err
}

// Discard takes a question out of tests and of the queue for good, keeping
// its history, and resolves its reports. Discarded questions also mark
// content a re-import should not bring back.
func (s *Store) Discard(ctx context.Context, id int64) (ReviewState, error) {
	q, err := s.Question(ctx, id)
	if err != nil {
		return ReviewState{}, err
	}
	if err := s.setStatus(ctx, id, StatusDiscarded); err != nil {
		return ReviewState{}, err
	}
	resolved, err := s.resolveReports(ctx, id)
	return ReviewState{Status: q.Status, ResolvedReports: resolved}, err
}

// RestoreReview undoes a decision: it sets the previous status back and
// reopens the reports the decision resolved.
func (s *Store) RestoreReview(ctx context.Context, id int64, st ReviewState) error {
	if err := s.setStatus(ctx, id, st.Status); err != nil {
		return err
	}
	for _, rid := range st.ResolvedReports {
		if _, err := s.db.ExecContext(ctx, `UPDATE question_reports SET resolved_at = '' WHERE id = ? AND question_id = ?`,
			rid, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) setStatus(ctx context.Context, id int64, st Status) error {
	switch st {
	case StatusDraft, StatusReviewed, StatusPublished, StatusDiscarded:
	default:
		return validate.Errors{"status": "Estado no válido"}
	}
	res, err := s.db.ExecContext(ctx, `UPDATE questions SET status = ?, updated_at = ? WHERE id = ?`, st, s.timestamp(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

type BatchResult struct {
	ID     int64           `json:"id"`
	OK     bool            `json:"ok"`
	Errors validate.Errors `json:"errors,omitempty"`
}

// AcceptBatch accepts each question independently: the ones that pass
// validation are published and the rest are reported with their reasons.
func (s *Store) AcceptBatch(ctx context.Context, ids []int64) ([]BatchResult, error) {
	out := make([]BatchResult, 0, len(ids))
	for _, id := range ids {
		_, err := s.Accept(ctx, id, nil)
		var v validate.Errors
		switch {
		case err == nil:
			out = append(out, BatchResult{ID: id, OK: true})
		case errors.As(err, &v):
			out = append(out, BatchResult{ID: id, Errors: v})
		case errors.Is(err, ErrNotFound):
			out = append(out, BatchResult{ID: id, Errors: validate.Errors{"id": "No existe"}})
		default:
			return out, err
		}
	}
	return out, nil
}
