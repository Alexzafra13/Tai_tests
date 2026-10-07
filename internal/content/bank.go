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
	"strings"

	"github.com/alexzafra13/tai_tests/internal/validate"
)

// The question bank (data/bank) ships official questions with the binary,
// one JSON file per source, so a new installation starts with content.
// Loading is idempotent: each question has a stable key within its source,
// and bank_entries remembers which keys were loaded, so questions the
// installation edited, discarded or deleted are never brought back.

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
	Added    int
	Existing int
	Problems []BankProblem
}

// LoadBank adds the bank questions this installation has not loaded yet,
// as drafts to go through Review. Questions that fail validation are
// reported and retried on the next load.
func (s *Store) LoadBank(ctx context.Context, files []BankFile) (BankResult, error) {
	var res BankResult
	for _, f := range files {
		if err := s.inTx(ctx, func(tx *sql.Tx) error { return s.loadBankFile(ctx, tx, f, &res) }); err != nil {
			return res, fmt.Errorf("bank %s: %w", f.Source.Reference, err)
		}
	}
	return res, nil
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
	origin, _ := originFor(f.Source.Kind)
	now := s.timestamp()
	for _, q := range pending {
		correct, _ := CorrectIndex(q.Correct)
		_, err := s.createQuestion(ctx, tx, QuestionInput{
			Stem: q.Stem, Options: q.Options, Correct: correct, Explanation: q.Explanation,
			Origin: origin, Author: AuthorImport, SourceID: sourceID, SourceRef: q.SourceRef,
			Status: StatusDraft, Annulled: q.Annulled, FixedOrder: q.FixedOrder,
		})
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
	}
	return nil
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
