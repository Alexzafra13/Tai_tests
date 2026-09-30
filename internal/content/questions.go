package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/alexzafra13/tai_tests/internal/textmatch"
)

type Origin string

const (
	OriginOfficial  Origin = "official"
	OriginLaw       Origin = "law"
	OriginTechnical Origin = "technical"
)

// sourceKind is the only source kind a question of this origin may cite.
func (o Origin) sourceKind() (SourceKind, bool) {
	switch o {
	case OriginOfficial:
		return KindINAPExam, true
	case OriginLaw:
		return KindLaw, true
	case OriginTechnical:
		return KindTechnicalDoc, true
	}
	return "", false
}

type Author string

const (
	AuthorManual Author = "manual"
	AuthorAI     Author = "ai"
	AuthorImport Author = "import"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusReviewed  Status = "reviewed"
	StatusPublished Status = "published"
	// StatusDiscarded hides a question for good while keeping its answer
	// history; questions that were never answered can be deleted instead.
	StatusDiscarded Status = "discarded"
)

// MinQuoteLength is the minimum length (in characters, after
// normalization) of a source quote, so that trivial fragments like "la
// Administración" cannot count as justification.
const MinQuoteLength = 20

type Question struct {
	ID          int64     `json:"id"`
	Stem        string    `json:"stem"`
	Options     [4]string `json:"options"`
	Correct     int       `json:"correct"`
	Explanation string    `json:"explanation"`
	Origin      Origin    `json:"origin"`
	Author      Author    `json:"author"`
	SourceID    int64     `json:"source_id"`
	SourceTitle string    `json:"source_title"`
	SourceRef   string    `json:"source_ref"`
	SourceQuote string    `json:"source_quote"`
	Status      Status    `json:"status"`
	Annulled    bool      `json:"annulled"`
	Flagged     bool      `json:"flagged"`
	FlagNote    string    `json:"flag_note"`
	TopicIDs    []int64   `json:"topic_ids"`
	Revision    int       `json:"revision"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

type QuestionInput struct {
	Stem        string    `json:"stem"`
	Options     [4]string `json:"options"`
	Correct     int       `json:"correct"`
	Explanation string    `json:"explanation"`
	Origin      Origin    `json:"origin"`
	Author      Author    `json:"author"`
	SourceID    int64     `json:"source_id"`
	SourceRef   string    `json:"source_ref"`
	SourceQuote string    `json:"source_quote"`
	Status      Status    `json:"status"`
	Annulled    bool      `json:"annulled"`
	Flagged     bool      `json:"flagged"`
	FlagNote    string    `json:"flag_note"`
	TopicIDs    []int64   `json:"topic_ids"`
}

func (in *QuestionInput) normalize() {
	in.Stem = strings.TrimSpace(in.Stem)
	for i := range in.Options {
		in.Options[i] = strings.TrimSpace(in.Options[i])
	}
	in.Explanation = strings.TrimSpace(in.Explanation)
	in.SourceRef = strings.TrimSpace(in.SourceRef)
	in.SourceQuote = strings.TrimSpace(in.SourceQuote)
	in.FlagNote = strings.TrimSpace(in.FlagNote)
	if in.Status == "" {
		in.Status = StatusDraft
	}
	if !in.Flagged {
		in.FlagNote = ""
	}
	seen := map[int64]bool{}
	ids := in.TopicIDs[:0:0]
	for _, id := range in.TopicIDs {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	in.TopicIDs = ids
}

// validate enforces the content rules. It needs the database to check the
// cited source: its kind must match the origin and, when a quote is given,
// the quote must appear literally in the source text.
func (in QuestionInput) validate(ctx context.Context, q queryer) error {
	v := ValidationError{}

	if in.Stem == "" {
		v["stem"] = "El enunciado es obligatorio"
	}
	seen := map[string]bool{}
	for i, opt := range in.Options {
		key := fmt.Sprintf("options.%d", i)
		norm := strings.ToLower(textmatch.Normalize(opt))
		switch {
		case norm == "":
			v[key] = "La opción no puede estar vacía"
		case seen[norm]:
			v[key] = "Opción repetida"
		}
		seen[norm] = true
	}
	if in.Correct < 0 || in.Correct > 3 {
		v["correct"] = "Indica la opción correcta"
	}

	wantKind, ok := in.Origin.sourceKind()
	if !ok {
		v["origin"] = "Origen no válido"
	}
	switch in.Author {
	case AuthorManual, AuthorAI, AuthorImport:
	default:
		v["author"] = "Autor no válido"
	}
	switch in.Status {
	case StatusDraft, StatusReviewed, StatusPublished, StatusDiscarded:
	default:
		v["status"] = "Estado no válido"
	}
	if in.Status == StatusPublished && len(in.TopicIDs) == 0 {
		v["topic_ids"] = "Asigna al menos un tema antes de publicar"
	}

	if in.SourceRef == "" {
		v["source_ref"] = "Indica la referencia (año y nº de pregunta, artículo…)"
	}
	if in.Origin != OriginOfficial && ok {
		if in.SourceQuote == "" {
			v["source_quote"] = "La cita literal de la fuente es obligatoria"
		} else if utf8.RuneCountInString(textmatch.Normalize(in.SourceQuote)) < MinQuoteLength {
			v["source_quote"] = fmt.Sprintf("La cita es demasiado corta (mínimo %d caracteres)", MinQuoteLength)
		}
	}

	if in.SourceID == 0 {
		v["source_id"] = "La fuente es obligatoria"
	} else {
		var kind SourceKind
		var text string
		err := q.QueryRowContext(ctx, `SELECT kind, full_text FROM sources WHERE id = ?`, in.SourceID).Scan(&kind, &text)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			v["source_id"] = "La fuente no existe"
		case err != nil:
			return err
		case ok && kind != wantKind:
			v["source_id"] = "El tipo de fuente no corresponde al origen de la pregunta"
		case in.SourceQuote != "" && v["source_quote"] == "":
			if text == "" {
				v["source_quote"] = "La fuente no tiene texto completo; no se puede verificar la cita"
			} else if !textmatch.Contains(text, in.SourceQuote) {
				v["source_quote"] = "La cita no aparece literalmente en el texto de la fuente"
			}
		}
	}

	if missing, err := topicsMissing(ctx, q, in.TopicIDs); err != nil {
		return err
	} else if len(missing) > 0 {
		v["topic_ids"] = "Algún tema no existe"
	}

	return v.orNil()
}

func (s *Store) CreateQuestion(ctx context.Context, in QuestionInput) (int64, error) {
	in.normalize()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := in.validate(ctx, tx); err != nil {
		return 0, err
	}

	now := s.timestamp()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO questions (stem, option_a, option_b, option_c, option_d, correct,
		explanation, origin, author, source_id, source_ref, source_quote, status, annulled, flagged, flag_note,
		created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		in.Stem, in.Options[0], in.Options[1], in.Options[2], in.Options[3], in.Correct,
		in.Explanation, in.Origin, in.Author, in.SourceID, in.SourceRef, in.SourceQuote, in.Status,
		boolInt(in.Annulled), boolInt(in.Flagged), in.FlagNote, now, now).Scan(&id)
	if err != nil {
		return 0, err
	}
	if err := setTopics(ctx, tx, id, in.TopicIDs); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) UpdateQuestion(ctx context.Context, id int64, in QuestionInput) error {
	in.normalize()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := in.validate(ctx, tx); err != nil {
		return err
	}

	// The revision goes up only when what is asked or answered changes, so
	// attempts on an older wording can be told apart in the stats.
	res, err := tx.ExecContext(ctx, `UPDATE questions SET
		revision = revision + (stem <> ?1 OR option_a <> ?2 OR option_b <> ?3 OR option_c <> ?4
			OR option_d <> ?5 OR correct <> ?6),
		stem = ?1, option_a = ?2, option_b = ?3, option_c = ?4, option_d = ?5, correct = ?6,
		explanation = ?7, origin = ?8, author = ?9, source_id = ?10, source_ref = ?11,
		source_quote = ?12, status = ?13, annulled = ?14, flagged = ?15, flag_note = ?16, updated_at = ?17
		WHERE id = ?18`,
		in.Stem, in.Options[0], in.Options[1], in.Options[2], in.Options[3], in.Correct,
		in.Explanation, in.Origin, in.Author, in.SourceID, in.SourceRef, in.SourceQuote, in.Status,
		boolInt(in.Annulled), boolInt(in.Flagged), in.FlagNote, s.timestamp(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM question_topics WHERE question_id = ?`, id); err != nil {
		return err
	}
	if err := setTopics(ctx, tx, id, in.TopicIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func setTopics(ctx context.Context, tx *sql.Tx, questionID int64, topicIDs []int64) error {
	for _, tid := range topicIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO question_topics (question_id, topic_id) VALUES (?, ?)`, questionID, tid); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteQuestion(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM questions WHERE id = ?`, id)
	if err != nil {
		if isConstraint(err, "FOREIGN KEY") {
			return ErrHasHistory
		}
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

const questionColumns = `q.id, q.stem, q.option_a, q.option_b, q.option_c, q.option_d, q.correct, q.explanation,
	q.origin, q.author, q.source_id, s.title, q.source_ref, q.source_quote, q.status, q.annulled, q.flagged,
	q.flag_note, q.revision, q.created_at, q.updated_at`

func scanQuestion(row interface{ Scan(...any) error }, q *Question) error {
	return row.Scan(&q.ID, &q.Stem, &q.Options[0], &q.Options[1], &q.Options[2], &q.Options[3], &q.Correct,
		&q.Explanation, &q.Origin, &q.Author, &q.SourceID, &q.SourceTitle, &q.SourceRef, &q.SourceQuote,
		&q.Status, &q.Annulled, &q.Flagged, &q.FlagNote, &q.Revision, &q.CreatedAt, &q.UpdatedAt)
}

func (s *Store) Question(ctx context.Context, id int64) (Question, error) {
	var q Question
	err := scanQuestion(s.db.QueryRowContext(ctx, `SELECT `+questionColumns+`
		FROM questions q JOIN sources s ON s.id = q.source_id WHERE q.id = ?`, id), &q)
	if errors.Is(err, sql.ErrNoRows) {
		return q, ErrNotFound
	} else if err != nil {
		return q, err
	}
	topics, err := s.topicIDs(ctx, []int64{id})
	q.TopicIDs = topics[id]
	return q, err
}

// QuestionFilter selects questions for listing. Zero values mean "any".
type QuestionFilter struct {
	Status   Status
	Origin   Origin
	SourceID int64
	TopicID  int64
	BlockID  int64
	Flagged  *bool
	Text     string // substring match on stem and options; FTS5 search comes later
	Limit    int
	Offset   int
}

type QuestionPage struct {
	Items []Question `json:"items"`
	Total int        `json:"total"`
}

func (s *Store) ListQuestions(ctx context.Context, f QuestionFilter) (QuestionPage, error) {
	var where []string
	var args []any
	if f.Status != "" {
		where = append(where, "q.status = ?")
		args = append(args, f.Status)
	} else {
		where = append(where, "q.status <> 'discarded'")
	}
	if f.Origin != "" {
		where = append(where, "q.origin = ?")
		args = append(args, f.Origin)
	}
	if f.SourceID != 0 {
		where = append(where, "q.source_id = ?")
		args = append(args, f.SourceID)
	}
	if f.TopicID != 0 {
		where = append(where, "q.id IN (SELECT question_id FROM question_topics WHERE topic_id = ?)")
		args = append(args, f.TopicID)
	}
	if f.BlockID != 0 {
		where = append(where, `q.id IN (SELECT qt.question_id FROM question_topics qt
			JOIN topics t ON t.id = qt.topic_id WHERE t.block_id = ?)`)
		args = append(args, f.BlockID)
	}
	if f.Flagged != nil {
		where = append(where, "q.flagged = ?")
		args = append(args, boolInt(*f.Flagged))
	}
	if t := strings.TrimSpace(f.Text); t != "" {
		like := "%" + escapeLike(t) + "%"
		where = append(where, `(q.stem LIKE ? ESCAPE '\' OR q.option_a LIKE ? ESCAPE '\' OR q.option_b LIKE ? ESCAPE '\'
			OR q.option_c LIKE ? ESCAPE '\' OR q.option_d LIKE ? ESCAPE '\' OR q.source_ref LIKE ? ESCAPE '\')`)
		for range 6 {
			args = append(args, like)
		}
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}

	page := QuestionPage{Items: []Question{}}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM questions q`+cond, args...).Scan(&page.Total); err != nil {
		return page, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+questionColumns+` FROM questions q
		JOIN sources s ON s.id = q.source_id`+cond+` ORDER BY q.id DESC LIMIT ? OFFSET ?`,
		append(args, limit, max(f.Offset, 0))...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var q Question
		if err := scanQuestion(rows, &q); err != nil {
			return page, err
		}
		page.Items = append(page.Items, q)
		ids = append(ids, q.ID)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}

	topics, err := s.topicIDs(ctx, ids)
	if err != nil {
		return page, err
	}
	for i := range page.Items {
		page.Items[i].TopicIDs = topics[page.Items[i].ID]
	}
	return page, nil
}

// topicIDs returns the topic ids of each question, never nil slices.
func (s *Store) topicIDs(ctx context.Context, questionIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(questionIDs))
	if len(questionIDs) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(questionIDs)), ",")
	args := make([]any, len(questionIDs))
	for i, id := range questionIDs {
		args[i] = id
		out[id] = []int64{}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT qt.question_id, qt.topic_id FROM question_topics qt
		JOIN topics t ON t.id = qt.topic_id JOIN blocks b ON b.id = t.block_id
		WHERE qt.question_id IN (`+placeholders+`) ORDER BY b.position, t.position`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var qid, tid int64
		if err := rows.Scan(&qid, &tid); err != nil {
			return nil, err
		}
		out[qid] = append(out[qid], tid)
	}
	return out, rows.Err()
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
