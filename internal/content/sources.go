package content

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/validate"

	"github.com/alexzafra13/tai_tests/internal/textmatch"
)

type SourceKind string

const (
	KindINAPExam     SourceKind = "inap_exam"
	KindLaw          SourceKind = "law"
	KindTechnicalDoc SourceKind = "technical_doc"
)

func (k SourceKind) valid() bool {
	return k == KindINAPExam || k == KindLaw || k == KindTechnicalDoc
}

type Source struct {
	ID          int64      `json:"id"`
	Kind        SourceKind `json:"kind"`
	Title       string     `json:"title"`
	Reference   string     `json:"reference"`
	URL         string     `json:"url"`
	VersionDate string     `json:"version_date"`
	FullText    string     `json:"full_text,omitempty"`
	HasText     bool       `json:"has_text"`
	Questions   int        `json:"questions"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

type SourceInput struct {
	Kind        SourceKind `json:"kind"`
	Title       string     `json:"title"`
	Reference   string     `json:"reference"`
	URL         string     `json:"url"`
	VersionDate string     `json:"version_date"`
	FullText    string     `json:"full_text"`
}

func (in *SourceInput) normalize() {
	in.Title = strings.TrimSpace(in.Title)
	in.Reference = strings.TrimSpace(in.Reference)
	in.URL = strings.TrimSpace(in.URL)
	in.VersionDate = strings.TrimSpace(in.VersionDate)
	in.FullText = strings.TrimSpace(in.FullText)
}

func (in SourceInput) validate() error {
	v := validate.Errors{}
	if !in.Kind.valid() {
		v["kind"] = "Tipo de fuente no válido"
	}
	if in.Title == "" {
		v["title"] = "El título es obligatorio"
	}
	if in.VersionDate != "" {
		if _, err := time.Parse("2006-01-02", in.VersionDate); err != nil {
			v["version_date"] = "Fecha no válida (AAAA-MM-DD)"
		}
	}
	if in.URL != "" && !strings.HasPrefix(in.URL, "https://") && !strings.HasPrefix(in.URL, "http://") {
		v["url"] = "La URL debe empezar por http:// o https://"
	}
	return v.Err()
}

const sourceColumns = `s.id, s.kind, s.title, s.reference, s.url, s.version_date, s.full_text <> '',
	(SELECT count(*) FROM questions q WHERE q.source_id = s.id), s.created_at, s.updated_at`

func scanSource(row interface{ Scan(...any) error }, src *Source) error {
	return row.Scan(&src.ID, &src.Kind, &src.Title, &src.Reference, &src.URL, &src.VersionDate,
		&src.HasText, &src.Questions, &src.CreatedAt, &src.UpdatedAt)
}

// ListSources returns sources without their full text, optionally filtered
// by kind.
func (s *Store) ListSources(ctx context.Context, kind SourceKind) ([]Source, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+sourceColumns+` FROM sources s
		WHERE ? = '' OR s.kind = ? ORDER BY s.kind, s.title`, kind, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		var src Source
		if err := scanSource(rows, &src); err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	return out, rows.Err()
}

// Source returns one source including its full text.
func (s *Store) Source(ctx context.Context, id int64) (Source, error) {
	var src Source
	err := scanSource(s.db.QueryRowContext(ctx, `SELECT `+sourceColumns+` FROM sources s WHERE s.id = ?`, id), &src)
	if errors.Is(err, sql.ErrNoRows) {
		return src, ErrNotFound
	} else if err != nil {
		return src, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT full_text FROM sources WHERE id = ?`, id).Scan(&src.FullText)
	return src, err
}

func (s *Store) CreateSource(ctx context.Context, in SourceInput) (int64, error) {
	in.normalize()
	if err := in.validate(); err != nil {
		return 0, err
	}
	now := s.timestamp()
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO sources (kind, title, reference, url, version_date, full_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		in.Kind, in.Title, in.Reference, in.URL, in.VersionDate, in.FullText, now, now).Scan(&id)
	return id, mapSourceErr(err)
}

// UpdateSource replaces a source's fields. Changing the text of a source
// that already backs questions is refused if any of their quotes would no
// longer be found in it.
func (s *Store) UpdateSource(ctx context.Context, id int64, in SourceInput) error {
	in.normalize()
	if err := in.validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldKind SourceKind
	if err := tx.QueryRowContext(ctx, `SELECT kind FROM sources WHERE id = ?`, id).Scan(&oldKind); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, `SELECT id, source_quote FROM questions WHERE source_id = ?`, id)
	if err != nil {
		return err
	}
	var broken, total int
	for rows.Next() {
		var qid int64
		var quote string
		if err := rows.Scan(&qid, &quote); err != nil {
			rows.Close()
			return err
		}
		total++
		if quote != "" && !textmatch.Contains(in.FullText, quote) {
			broken++
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	v := validate.Errors{}
	if total > 0 && oldKind != in.Kind {
		v["kind"] = "No se puede cambiar el tipo de una fuente con preguntas"
	}
	if broken > 0 {
		v["full_text"] = pluralize(broken, "pregunta cita", "preguntas citan") + " texto que ya no aparece en esta versión"
	}
	if err := v.Err(); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE sources SET kind = ?, title = ?, reference = ?, url = ?, version_date = ?,
		full_text = ?, updated_at = ? WHERE id = ?`,
		in.Kind, in.Title, in.Reference, in.URL, in.VersionDate, in.FullText, s.timestamp(), id)
	if err := mapSourceErr(err); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteSource(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, id)
	if err != nil {
		if isConstraint(err, "FOREIGN KEY") {
			return ErrInUse
		}
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CheckQuote reports whether quote appears literally in the source text.
func (s *Store) CheckQuote(ctx context.Context, sourceID int64, quote string) (bool, error) {
	var text string
	err := s.db.QueryRowContext(ctx, `SELECT full_text FROM sources WHERE id = ?`, sourceID).Scan(&text)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	} else if err != nil {
		return false, err
	}
	return textmatch.Contains(text, quote), nil
}

func mapSourceErr(err error) error {
	if err != nil && isConstraint(err, "UNIQUE") {
		return validate.Errors{"reference": "Ya existe una fuente de este tipo con esa referencia"}
	}
	return err
}

func isConstraint(err error, kind string) bool {
	return err != nil && strings.Contains(err.Error(), kind+" constraint failed")
}

func pluralize(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
