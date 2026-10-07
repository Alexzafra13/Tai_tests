package content

import (
	"context"
	"database/sql"
	"errors"
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
	ID          int64      `json:"id"`
	Stem        string     `json:"stem"`
	Options     [4]string  `json:"options"`
	Correct     int        `json:"correct"`
	Explanation string     `json:"explanation"`
	Origin      Origin     `json:"origin"`
	SourceTitle string     `json:"source_title"`
	SourceRef   string     `json:"source_ref"`
	SourceQuote string     `json:"source_quote"`
	SourceKind  SourceKind `json:"source_kind"`
	SourceURL   string     `json:"source_url"`
	// Articles are the law articles the question cites.
	Articles []ArticleLink `json:"articles"`
}

// hitColumns and scanHit read a SearchHit from questions q joined with
// sources s.
const hitColumns = `q.id, q.stem, q.option_a, q.option_b, q.option_c, q.option_d, q.correct,
	q.explanation, q.origin, s.title, q.source_ref, q.source_quote, s.kind, s.url`

func scanHit(row interface{ Scan(...any) error }, h *SearchHit) error {
	return row.Scan(&h.ID, &h.Stem, &h.Options[0], &h.Options[1], &h.Options[2], &h.Options[3], &h.Correct,
		&h.Explanation, &h.Origin, &h.SourceTitle, &h.SourceRef, &h.SourceQuote, &h.SourceKind, &h.SourceURL)
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
	rows, err := s.db.QueryContext(ctx, `SELECT `+hitColumns+`
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
		if err := scanHit(rows, &h); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	ids := make([]int64, len(hits))
	for i, h := range hits {
		ids[i] = h.ID
	}
	links, err := s.QuestionArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].Articles = append([]ArticleLink{}, links[hits[i].ID]...)
	}
	return hits, nil
}

// TopicRef names a topic a question belongs to.
type TopicRef struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// StudyQuestion is a published question as a studying user sees it on its
// own page.
type StudyQuestion struct {
	SearchHit
	Topics []TopicRef `json:"topics"`
}

// PublishedQuestion returns a published, non-annulled question with its
// topics. Anything else is ErrNotFound, so drafts stay hidden.
func (s *Store) PublishedQuestion(ctx context.Context, id int64) (StudyQuestion, error) {
	q := StudyQuestion{Topics: []TopicRef{}}
	err := scanHit(s.db.QueryRowContext(ctx, `SELECT `+hitColumns+`
		FROM questions q JOIN sources s ON s.id = q.source_id
		WHERE q.id = ? AND q.status = 'published' AND q.annulled = 0`, id), &q.SearchHit)
	if errors.Is(err, sql.ErrNoRows) {
		return q, ErrNotFound
	} else if err != nil {
		return q, err
	}
	links, err := s.QuestionArticles(ctx, []int64{id})
	if err != nil {
		return q, err
	}
	q.Articles = append([]ArticleLink{}, links[id]...)
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.number, t.title FROM question_topics qt
		JOIN topics t ON t.id = qt.topic_id JOIN blocks b ON b.id = t.block_id
		WHERE qt.question_id = ? ORDER BY b.position, t.position`, id)
	if err != nil {
		return q, err
	}
	defer rows.Close()
	for rows.Next() {
		var t TopicRef
		if err := rows.Scan(&t.ID, &t.Number, &t.Title); err != nil {
			return q, err
		}
		q.Topics = append(q.Topics, t)
	}
	return q, rows.Err()
}
