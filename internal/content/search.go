package content

import (
	"context"
	"strings"
	"unicode"
)

// Full-text search uses the questions_fts table (migration 0006), which
// ignores case and accents.

// matchesText is the SQL condition for questions matching an FTS query.
const matchesText = `q.id IN (SELECT rowid FROM questions_fts WHERE questions_fts MATCH ?)`

// ftsQuery turns what the user typed into an FTS5 query: every word must
// appear, as a prefix ("protec" finds "protección"). FTS syntax characters
// are dropped, so any input is safe. It returns "" when there is nothing
// to search for.
func ftsQuery(text string) string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, w := range words {
		words[i] = `"` + w + `"*`
	}
	return strings.Join(words, " ")
}

// SearchHit is a published question found by text, with its answer and
// source so it can be studied directly.
type SearchHit struct {
	ID          int64     `json:"id"`
	Stem        string    `json:"stem"`
	Options     [4]string `json:"options"`
	Correct     int       `json:"correct"`
	Explanation string    `json:"explanation"`
	Origin      Origin    `json:"origin"`
	SourceTitle string    `json:"source_title"`
	SourceRef   string    `json:"source_ref"`
	SourceQuote string    `json:"source_quote"`
}

// Search finds published, non-annulled questions by text, best matches
// first. Studying users use it; drafts never show up.
func (s *Store) Search(ctx context.Context, text string, limit int) ([]SearchHit, error) {
	hits := []SearchHit{}
	match := ftsQuery(text)
	if match == "" {
		return hits, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.db.QueryContext(ctx, `SELECT q.id, q.stem, q.option_a, q.option_b, q.option_c, q.option_d, q.correct,
			q.explanation, q.origin, s.title, q.source_ref, q.source_quote
		FROM questions_fts f
		JOIN questions q ON q.id = f.rowid
		JOIN sources s ON s.id = q.source_id
		WHERE questions_fts MATCH ? AND q.status = 'published' AND q.annulled = 0
		ORDER BY bm25(questions_fts) LIMIT ?`, match, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.ID, &h.Stem, &h.Options[0], &h.Options[1], &h.Options[2], &h.Options[3], &h.Correct,
			&h.Explanation, &h.Origin, &h.SourceTitle, &h.SourceRef, &h.SourceQuote); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
