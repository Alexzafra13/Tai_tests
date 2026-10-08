package content

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/alexzafra13/tai_tests/internal/validate"
)

// The question bank (data/bank) ships official questions with the binary,
// one JSON file per source, so a new installation starts with content.
// Loading is idempotent: each question has a stable key within its source,
// and bank_entries remembers which keys were loaded, so questions the
// installation edited, discarded or deleted are never brought back.
//
// A bank question arrives published only when the file says so (it was
// checked against the exam and its definitive answer key) and its topics
// exist in the installation's syllabus; otherwise it waits in Review.

type BankFile struct {
	Source    BankSource     `json:"source"`
	Questions []BankQuestion `json:"questions"`
}

type BankSource struct {
	Kind      SourceKind `json:"kind"`
	Title     string     `json:"title"`
	Reference string     `json:"reference"`
	URL       string     `json:"url,omitempty"`
}

type BankQuestion struct {
	// Key identifies the question within its source; it must never change.
	Key       string    `json:"key"`
	SourceRef string    `json:"source_ref"`
	Stem      string    `json:"stem"`
	Options   [4]string `json:"options"`
	// Correct is the letter of the right option, as in the answer key.
	Correct     string `json:"correct"`
	Annulled    bool   `json:"annulled,omitempty"`
	FixedOrder  bool   `json:"fixed_order,omitempty"`
	Explanation string `json:"explanation,omitempty"`
	// Topics are syllabus topic codes (B1-T01…).
	Topics []string `json:"topics,omitempty"`
	// Status is "published" for checked questions; empty means draft.
	Status Status `json:"status,omitempty"`
	// Articles are the sections of the study texts (data/laws) that answer
	// the question, checked by hand. They replace the articles the app
	// would find from the stem.
	Articles []BankArticle `json:"articles,omitempty"`
}

// BankArticle points to a section of a bundled law (an empty Section
// points to the law as a whole) or, for questions no law answers, to the
// official page that does (Title and URL), with the literal sentence of
// the page that backs the answer (Quote).
type BankArticle struct {
	Law     string `json:"law,omitempty"`
	Section string `json:"section,omitempty"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Quote   string `json:"quote,omitempty"`
}

// CorrectIndex converts a letter (a-d) to an option index.
func CorrectIndex(letter string) (int, bool) {
	if len(letter) != 1 || letter[0] < 'a' || letter[0] > 'd' {
		return 0, false
	}
	return int(letter[0] - 'a'), true
}

// ReadBank reads every *.json file at the root of fsys.
func ReadBank(fsys fs.FS) ([]BankFile, error) {
	names, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, err
	}
	var files []BankFile
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		f, err := ParseBankFile(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path.Base(name), err)
		}
		files = append(files, f)
	}
	return files, nil
}

func ParseBankFile(b []byte) (BankFile, error) {
	var f BankFile
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return f, fmt.Errorf("bank: %w", err)
	}
	if f.Source.Reference == "" {
		return f, errors.New("bank: the source needs a reference, it identifies the file's questions")
	}
	if _, ok := originFor(f.Source.Kind); !ok {
		return f, fmt.Errorf("bank: source kind %q not supported", f.Source.Kind)
	}
	keys := map[string]bool{}
	for _, q := range f.Questions {
		if strings.TrimSpace(q.Key) == "" || keys[q.Key] {
			return f, fmt.Errorf("bank: missing or repeated key %q", q.Key)
		}
		keys[q.Key] = true
		if _, ok := CorrectIndex(q.Correct); !ok {
			return f, fmt.Errorf("bank: question %s: correct must be a letter a-d", q.Key)
		}
		if q.Status != "" && q.Status != StatusPublished {
			return f, fmt.Errorf("bank: question %s: status must be empty or published", q.Key)
		}
		for _, a := range q.Articles {
			law := a.Law != "" && a.Title == "" && a.URL == "" && a.Quote == ""
			page := a.Law == "" && a.Section == "" && a.Title != "" && strings.HasPrefix(a.URL, "https://")
			if !law && !page {
				return f, fmt.Errorf("bank: question %s: an article is a law and section, or a title and https url", q.Key)
			}
		}
	}
	return f, nil
}

// originFor is the question origin of a bank source. Only official exams
// are supported: law and technical questions need a quote checked against
// the source's full text, which bank files do not carry.
func originFor(k SourceKind) (Origin, bool) {
	if k == KindINAPExam {
		return OriginOfficial, true
	}
	return "", false
}

type BankProblem struct {
	Source string `json:"source"`
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

type BankResult struct {
	Added     int
	Published int // of those added
	Existing  int
	Updated   int // existing questions brought up to date with the bank
	Problems  []BankProblem
}

// LoadBank adds the bank questions this installation has not loaded yet.
// Questions that fail validation are reported and retried on the next
// load.
func (s *Store) LoadBank(ctx context.Context, files []BankFile) (BankResult, error) {
	var res BankResult
	for _, f := range files {
		err := s.inTx(ctx, func(tx *sql.Tx) error {
			if err := s.loadBankFile(ctx, tx, f, &res); err != nil {
				return err
			}
			return syncBankQuestions(ctx, tx, f, &res)
		})
		if err != nil {
			return res, fmt.Errorf("bank %s: %w", f.Source.Reference, err)
		}
	}
	return res, nil
}

// syncBankQuestions brings the bank's checks to the questions of the file
// this installation already has, found by their source reference: their
// articles always and, while nobody has edited the question here and its
// text and answer are still the bank's, its explanation, topics, status
// and annulment.
func syncBankQuestions(ctx context.Context, tx *sql.Tx, f BankFile, res *BankResult) error {
	topics, err := topicsByCode(ctx, tx)
	if err != nil {
		return err
	}
	for _, q := range f.Questions {
		var id int64
		var cur Question
		var created, updated string
		err := tx.QueryRowContext(ctx, `SELECT q.id, q.stem, q.option_a, q.option_b, q.option_c, q.option_d, q.correct,
			q.explanation, q.status, q.annulled, q.created_at, q.updated_at
			FROM questions q JOIN sources s ON s.id = q.source_id
			WHERE s.kind = ? AND s.reference = ? AND q.source_ref = ?`, f.Source.Kind, f.Source.Reference, q.SourceRef).Scan(
			&id, &cur.Stem, &cur.Options[0], &cur.Options[1], &cur.Options[2], &cur.Options[3], &cur.Correct,
			&cur.Explanation, &cur.Status, &cur.Annulled, &created, &updated)
		if errors.Is(err, sql.ErrNoRows) {
			continue // deleted in this installation
		} else if err != nil {
			return err
		}
		correct, _ := CorrectIndex(q.Correct)
		if created == updated && cur.Stem == q.Stem && cur.Options == q.Options && cur.Correct == correct {
			if err := syncBankQuestion(ctx, tx, id, cur, q, topics, res); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM question_sections WHERE question_id = ?`, id); err != nil {
			return err
		}
		for i, a := range q.Articles {
			if _, err := tx.ExecContext(ctx, `INSERT INTO question_sections (question_id, position, law_reference, block_id,
				title, url, quote) VALUES (?, ?, ?, ?, ?, ?, ?)`, id, i, a.Law, a.Section, a.Title, a.URL, a.Quote); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) loadBankFile(ctx context.Context, tx *sql.Tx, f BankFile, res *BankResult) error {
	loaded := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT question_key FROM bank_entries WHERE source_kind = ? AND source_reference = ?`,
		f.Source.Kind, f.Source.Reference)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		loaded[k] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var pending []BankQuestion
	for _, q := range f.Questions {
		if loaded[q.Key] {
			res.Existing++
		} else {
			pending = append(pending, q)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	sourceID, err := s.bankSource(ctx, tx, f.Source)
	if err != nil {
		return err
	}
	topics, err := topicsByCode(ctx, tx)
	if err != nil {
		return err
	}
	origin, _ := originFor(f.Source.Kind)
	now := s.timestamp()
	for _, q := range pending {
		correct, _ := CorrectIndex(q.Correct)
		in := QuestionInput{
			Stem: q.Stem, Options: q.Options, Correct: correct, Explanation: q.Explanation,
			Origin: origin, Author: AuthorImport, SourceID: sourceID, SourceRef: q.SourceRef,
			Status: StatusDraft, Annulled: q.Annulled, FixedOrder: q.FixedOrder,
		}
		complete := len(q.Topics) > 0
		for _, code := range q.Topics {
			if id, ok := topics[code]; ok {
				in.TopicIDs = append(in.TopicIDs, id)
			} else {
				complete = false
			}
		}
		if q.Status == StatusPublished && complete {
			in.Status = StatusPublished
		}
		_, err := s.createQuestion(ctx, tx, in)
		var verr validate.Errors
		if errors.As(err, &verr) {
			res.Problems = append(res.Problems, BankProblem{f.Source.Reference, q.Key, verr.Error()})
			continue
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO bank_entries (source_kind, source_reference, question_key, loaded_at)
			VALUES (?, ?, ?, ?)`, f.Source.Kind, f.Source.Reference, q.Key, now); err != nil {
			return err
		}
		res.Added++
		if in.Status == StatusPublished {
			res.Published++
		}
	}
	return nil
}

// syncBankQuestion applies the bank's explanation, topics, status and
// annulment to a question nobody has edited, keeping updated_at so later
// versions of the bank can still update it.
func syncBankQuestion(ctx context.Context, tx *sql.Tx, id int64, cur Question, q BankQuestion,
	topics map[string]int64, res *BankResult) error {
	var ids []int64
	complete := len(q.Topics) > 0
	for _, code := range q.Topics {
		if tid, ok := topics[code]; ok {
			ids = append(ids, tid)
		} else {
			complete = false
		}
	}
	status := StatusDraft
	if q.Status == StatusPublished && complete {
		status = StatusPublished
	}
	rows, err := tx.QueryContext(ctx, `SELECT topic_id FROM question_topics WHERE question_id = ? ORDER BY topic_id`, id)
	if err != nil {
		return err
	}
	var have []int64
	for rows.Next() {
		var tid int64
		if err := rows.Scan(&tid); err != nil {
			rows.Close()
			return err
		}
		have = append(have, tid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	slices.Sort(ids)
	if cur.Explanation == q.Explanation && cur.Status == status && cur.Annulled == q.Annulled && slices.Equal(have, ids) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE questions SET explanation = ?, status = ?, annulled = ? WHERE id = ?`,
		q.Explanation, status, q.Annulled, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM question_topics WHERE question_id = ?`, id); err != nil {
		return err
	}
	for _, tid := range ids {
		if _, err := tx.ExecContext(ctx, `INSERT INTO question_topics (question_id, topic_id) VALUES (?, ?)`, id, tid); err != nil {
			return err
		}
	}
	res.Updated++
	return nil
}

func topicsByCode(ctx context.Context, tx *sql.Tx) (map[string]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT code, id FROM topics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var code string
		var id int64
		if err := rows.Scan(&code, &id); err != nil {
			return nil, err
		}
		out[code] = id
	}
	return out, rows.Err()
}

// bankSource returns the source with the bank file's kind and reference,
// creating it if the installation does not have it.
func (s *Store) bankSource(ctx context.Context, tx *sql.Tx, src BankSource) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM sources WHERE kind = ? AND reference = ?`, src.Kind, src.Reference).Scan(&id)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}
	in := SourceInput{Kind: src.Kind, Title: src.Title, Reference: src.Reference, URL: src.URL}
	in.normalize()
	if err := in.validate(); err != nil {
		return 0, err
	}
	now := s.timestamp()
	err = tx.QueryRowContext(ctx, `INSERT INTO sources (kind, title, reference, url, version_date, full_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, '', '', ?, ?) RETURNING id`, in.Kind, in.Title, in.Reference, in.URL, now, now).Scan(&id)
	return id, mapSourceErr(err)
}
