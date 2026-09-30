package content

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alexzafra13/tai_tests/internal/textmatch"
)

// The review queue holds what is not ready for tests yet: drafts (imported
// or generated questions) and questions flagged as doubtful during a test.
// Flagged ones come first, since they were raised while studying.

type ReviewKind string

const (
	ReviewAll     ReviewKind = ""
	ReviewFlagged ReviewKind = "flagged"
	ReviewDrafts  ReviewKind = "drafts"
)

func (k ReviewKind) where() string {
	switch k {
	case ReviewFlagged:
		return "q.flagged = 1 AND q.status <> 'discarded'"
	case ReviewDrafts:
		return "q.status IN ('draft', 'reviewed')"
	default:
		return "(q.status IN ('draft', 'reviewed') OR q.flagged = 1) AND q.status <> 'discarded'"
	}
}

// ExcerptWindow is how much source text is shown around a quote.
const ExcerptWindow = 300

type ReviewItem struct {
	Question
	// Excerpt shows the quote inside its source text; nil when there is no
	// quote or it cannot be found (which validation normally prevents).
	Excerpt *textmatch.Excerpt `json:"excerpt"`
}

type ReviewPage struct {
	Items []ReviewItem `json:"items"`
	Total int          `json:"total"`
}

type ReviewCounts struct {
	Flagged int `json:"flagged"`
	Drafts  int `json:"drafts"`
	Total   int `json:"total"`
}

func (s *Store) ReviewCounts(ctx context.Context) (ReviewCounts, error) {
	var c ReviewCounts
	err := s.db.QueryRowContext(ctx, `SELECT
			count(CASE WHEN `+ReviewFlagged.where()+` THEN 1 END),
			count(CASE WHEN `+ReviewDrafts.where()+` THEN 1 END),
			count(CASE WHEN `+ReviewAll.where()+` THEN 1 END)
		FROM questions q`).Scan(&c.Flagged, &c.Drafts, &c.Total)
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
		WHERE `+cond+` ORDER BY q.flagged DESC, q.id LIMIT ? OFFSET ?`, append(args, limit, max(offset, 0))...)
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
	for i := range page.Items {
		page.Items[i].TopicIDs = topics[page.Items[i].ID]
	}
	return page, nil
}

// ReviewState is what a review decision changes, returned so the decision
// can be undone.
type ReviewState struct {
	Status   Status `json:"status"`
	Flagged  bool   `json:"flagged"`
	FlagNote string `json:"flag_note"`
}

func (s *Store) reviewState(ctx context.Context, id int64) (ReviewState, error) {
	var st ReviewState
	err := s.db.QueryRowContext(ctx, `SELECT status, flagged, flag_note FROM questions WHERE id = ?`, id).
		Scan(&st.Status, &st.Flagged, &st.FlagNote)
	if errors.Is(err, sql.ErrNoRows) {
		return st, ErrNotFound
	}
	return st, err
}

// Accept publishes a question and clears its doubt flag. It goes through
// the normal validation, so a question without topics or with a quote that
// is not in its source cannot be published. topicIDs, when given, replace
// the question's topics first.
func (s *Store) Accept(ctx context.Context, id int64, topicIDs []int64) (ReviewState, error) {
	q, err := s.Question(ctx, id)
	if err != nil {
		return ReviewState{}, err
	}
	prev := ReviewState{Status: q.Status, Flagged: q.Flagged, FlagNote: q.FlagNote}
	in := q.Input()
	if len(topicIDs) > 0 {
		in.TopicIDs = topicIDs
	}
	in.Status, in.Flagged, in.FlagNote = StatusPublished, false, ""
	return prev, s.UpdateQuestion(ctx, id, in)
}

// Discard takes a question out of tests and of the queue for good, keeping
// its history. Discarded questions also mark content a re-import should not
// bring back.
func (s *Store) Discard(ctx context.Context, id int64) (ReviewState, error) {
	prev, err := s.reviewState(ctx, id)
	if err != nil {
		return prev, err
	}
	return prev, s.RestoreReview(ctx, id, ReviewState{Status: StatusDiscarded})
}

// RestoreReview sets a question's review state back, to undo a decision.
func (s *Store) RestoreReview(ctx context.Context, id int64, st ReviewState) error {
	switch st.Status {
	case StatusDraft, StatusReviewed, StatusPublished, StatusDiscarded:
	default:
		return ValidationError{"status": "Estado no válido"}
	}
	if !st.Flagged {
		st.FlagNote = ""
	}
	res, err := s.db.ExecContext(ctx, `UPDATE questions SET status = ?, flagged = ?, flag_note = ?, updated_at = ?
		WHERE id = ?`, st.Status, boolInt(st.Flagged), st.FlagNote, s.timestamp(), id)
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
	Errors ValidationError `json:"errors,omitempty"`
}

// AcceptBatch accepts each question independently: the ones that pass
// validation are published and the rest are reported with their reasons.
func (s *Store) AcceptBatch(ctx context.Context, ids []int64) ([]BatchResult, error) {
	out := make([]BatchResult, 0, len(ids))
	for _, id := range ids {
		_, err := s.Accept(ctx, id, nil)
		var v ValidationError
		switch {
		case err == nil:
			out = append(out, BatchResult{ID: id, OK: true})
		case errors.As(err, &v):
			out = append(out, BatchResult{ID: id, Errors: v})
		case errors.Is(err, ErrNotFound):
			out = append(out, BatchResult{ID: id, Errors: ValidationError{"id": "No existe"}})
		default:
			return out, err
		}
	}
	return out, nil
}
