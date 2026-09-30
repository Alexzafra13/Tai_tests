// Package quiz runs test sessions: it picks questions, records every answer
// as an attempt and scores the result.
package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
)

type Mode string

const (
	// ModePractice reveals the answer, explanation and source right after
	// each question. Answers are final.
	ModePractice Mode = "practice"
	// ModeExam reveals nothing until the end; answers can be changed or
	// left blank, and an optional time limit applies.
	ModeExam Mode = "exam"
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusFinished   Status = "finished"
	StatusAbandoned  Status = "abandoned"
)

var (
	ErrNotFound        = errors.New("test not found")
	ErrNotInProgress   = errors.New("test is not in progress")
	ErrExpired         = errors.New("time is up")
	ErrAlreadyAnswered = errors.New("question already answered")
	ErrNoQuestions     = errors.New("no questions match the filters")
)

const (
	MaxQuestions    = 200
	MaxTimeLimitMin = 300
)

// Filters narrows the questions a test draws from. Empty lists mean "any".
// Topics and blocks combine with OR (a question in any selected topic or
// block); the other filters narrow further.
type Filters struct {
	TopicIDs  []int64          `json:"topic_ids"`
	BlockIDs  []int64          `json:"block_ids"`
	SourceIDs []int64          `json:"source_ids"`
	Origins   []content.Origin `json:"origins"`
	// QuestionIDs restricts the test to specific questions, e.g. to retry
	// the ones failed in a previous test.
	QuestionIDs []int64 `json:"question_ids"`
}

type CreateInput struct {
	Mode    Mode    `json:"mode"`
	Filters Filters `json:"filters"`
	Count   int     `json:"count"`
	// Penalty is the fraction of a correct answer subtracted per wrong one.
	Penalty      float64 `json:"penalty"`
	TimeLimitMin int     `json:"time_limit_min"`
}

func (in CreateInput) validate() error {
	v := content.ValidationError{}
	if in.Mode != ModePractice && in.Mode != ModeExam {
		v["mode"] = "Modo no válido"
	}
	if in.Count < 1 || in.Count > MaxQuestions {
		v["count"] = fmt.Sprintf("Entre 1 y %d preguntas", MaxQuestions)
	}
	if in.Penalty < 0 || in.Penalty > 1 {
		v["penalty"] = "La penalización debe estar entre 0 y 1"
	}
	if in.TimeLimitMin < 0 || in.TimeLimitMin > MaxTimeLimitMin {
		v["time_limit_min"] = fmt.Sprintf("Entre 0 y %d minutos", MaxTimeLimitMin)
	}
	if len(v) == 0 {
		return nil
	}
	return v
}

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

const timeFormat = "2006-01-02T15:04:05.000Z"

// eligibleWhere builds the condition for questions that may appear in a
// test: published, not annulled, and matching the filters.
func eligibleWhere(f Filters) (string, []any) {
	where := []string{"q.status = 'published'", "q.annulled = 0"}
	var args []any

	var scope []string
	if len(f.TopicIDs) > 0 {
		scope = append(scope, "q.id IN (SELECT question_id FROM question_topics WHERE topic_id IN ("+placeholders(len(f.TopicIDs))+"))")
		args = appendIDs(args, f.TopicIDs)
	}
	if len(f.BlockIDs) > 0 {
		scope = append(scope, `q.id IN (SELECT qt.question_id FROM question_topics qt JOIN topics t ON t.id = qt.topic_id
			WHERE t.block_id IN (`+placeholders(len(f.BlockIDs))+"))")
		args = appendIDs(args, f.BlockIDs)
	}
	if len(scope) > 0 {
		where = append(where, "("+strings.Join(scope, " OR ")+")")
	}
	if len(f.SourceIDs) > 0 {
		where = append(where, "q.source_id IN ("+placeholders(len(f.SourceIDs))+")")
		args = appendIDs(args, f.SourceIDs)
	}
	if len(f.QuestionIDs) > 0 {
		where = append(where, "q.id IN ("+placeholders(len(f.QuestionIDs))+")")
		args = appendIDs(args, f.QuestionIDs)
	}
	if len(f.Origins) > 0 {
		where = append(where, "q.origin IN ("+placeholders(len(f.Origins))+")")
		for _, o := range f.Origins {
			args = append(args, o)
		}
	}
	return strings.Join(where, " AND "), args
}

// Available counts the questions a test with these filters can draw from.
func (s *Store) Available(ctx context.Context, f Filters) (int, error) {
	cond, args := eligibleWhere(f)
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM questions q WHERE `+cond, args...).Scan(&n)
	return n, err
}

// Create draws up to in.Count random eligible questions and fixes their
// order, so the test can be resumed exactly as it was.
func (s *Store) Create(ctx context.Context, in CreateInput) (int64, error) {
	if in.Mode == ModePractice {
		in.TimeLimitMin = 0
	}
	if err := in.validate(); err != nil {
		return 0, err
	}
	config, err := json.Marshal(struct {
		Filters Filters `json:"filters"`
		Count   int     `json:"count"`
	}{in.Filters, in.Count})
	if err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	cond, args := eligibleWhere(in.Filters)
	rows, err := tx.QueryContext(ctx, `SELECT q.id, q.revision FROM questions q WHERE `+cond+` ORDER BY random() LIMIT ?`,
		append(args, in.Count)...)
	if err != nil {
		return 0, err
	}
	type pick struct{ id, revision int64 }
	var picks []pick
	for rows.Next() {
		var p pick
		if err := rows.Scan(&p.id, &p.revision); err != nil {
			rows.Close()
			return 0, err
		}
		picks = append(picks, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(picks) == 0 {
		return 0, ErrNoQuestions
	}

	now := s.now().UTC()
	deadline := ""
	if in.TimeLimitMin > 0 {
		deadline = now.Add(time.Duration(in.TimeLimitMin) * time.Minute).Format(timeFormat)
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO tests (mode, config, penalty, time_limit, started_at, deadline)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		in.Mode, string(config), in.Penalty, in.TimeLimitMin*60, now.Format(timeFormat), deadline).Scan(&id)
	if err != nil {
		return 0, err
	}
	for i, p := range picks {
		if _, err := tx.ExecContext(ctx, `INSERT INTO attempts (test_id, position, question_id, revision) VALUES (?, ?, ?, ?)`,
			id, i, p.id, p.revision); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

type Test struct {
	ID         int64   `json:"id"`
	Mode       Mode    `json:"mode"`
	Status     Status  `json:"status"`
	Penalty    float64 `json:"penalty"`
	TimeLimit  int     `json:"time_limit"` // seconds
	StartedAt  string  `json:"started_at"`
	Deadline   string  `json:"deadline,omitempty"`
	FinishedAt string  `json:"finished_at,omitempty"`
	// RemainingSec is set for timed tests in progress.
	RemainingSec *int    `json:"remaining_sec,omitempty"`
	Result       *Result `json:"result,omitempty"`
	Items        []Item  `json:"items"`
}

type Item struct {
	Position   int       `json:"position"`
	QuestionID int64     `json:"question_id"`
	Stem       string    `json:"stem"`
	Options    [4]string `json:"options"`
	Chosen     *int      `json:"chosen"`
	Flagged    bool      `json:"flagged"`
	FlagNote   string    `json:"flag_note,omitempty"`
	TimeMs     int       `json:"time_ms"`
	// Solution is only included once it may be shown: in practice mode
	// after answering, and in any mode once the test is over.
	Solution *Solution `json:"solution,omitempty"`
}

type Solution struct {
	Correct     int            `json:"correct"`
	IsCorrect   *bool          `json:"is_correct"` // nil when left blank
	Explanation string         `json:"explanation"`
	Origin      content.Origin `json:"origin"`
	SourceTitle string         `json:"source_title"`
	SourceRef   string         `json:"source_ref"`
	SourceQuote string         `json:"source_quote"`
	TopicIDs    []int64        `json:"topic_ids"`
}

// Get returns the test with its questions. A timed test whose deadline has
// passed is finished first, so a test abandoned mid-exam scores as if time
// ran out.
func (s *Store) Get(ctx context.Context, id int64) (Test, error) {
	t, err := s.header(ctx, s.db, id)
	if err != nil {
		return t, err
	}
	if t.Status == StatusInProgress && s.expired(t) {
		if _, err := s.Finish(ctx, id); err != nil && !errors.Is(err, ErrNotInProgress) {
			return t, err
		}
		if t, err = s.header(ctx, s.db, id); err != nil {
			return t, err
		}
	}

	rows, err := s.db.QueryContext(ctx, `SELECT a.position, q.id, q.stem, q.option_a, q.option_b, q.option_c, q.option_d,
			a.chosen, a.is_correct, a.time_ms, q.flagged, q.flag_note,
			q.correct, q.explanation, q.origin, s.title, q.source_ref, q.source_quote
		FROM attempts a JOIN questions q ON q.id = a.question_id JOIN sources s ON s.id = q.source_id
		WHERE a.test_id = ? ORDER BY a.position`, id)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	t.Items = []Item{}
	var withSolution []int
	for rows.Next() {
		var it Item
		var chosen, isCorrect sql.NullInt64
		var sol Solution
		if err := rows.Scan(&it.Position, &it.QuestionID, &it.Stem, &it.Options[0], &it.Options[1], &it.Options[2], &it.Options[3],
			&chosen, &isCorrect, &it.TimeMs, &it.Flagged, &it.FlagNote,
			&sol.Correct, &sol.Explanation, &sol.Origin, &sol.SourceTitle, &sol.SourceRef, &sol.SourceQuote); err != nil {
			return t, err
		}
		if chosen.Valid {
			c := int(chosen.Int64)
			it.Chosen = &c
		}
		if isCorrect.Valid {
			b := isCorrect.Int64 == 1
			sol.IsCorrect = &b
		}
		if t.Status != StatusInProgress || (t.Mode == ModePractice && it.Chosen != nil) {
			it.Solution = &sol
			withSolution = append(withSolution, len(t.Items))
		}
		t.Items = append(t.Items, it)
	}
	if err := rows.Err(); err != nil {
		return t, err
	}
	rows.Close()

	for _, i := range withSolution {
		if t.Items[i].Solution.TopicIDs, err = s.topicIDs(ctx, t.Items[i].QuestionID); err != nil {
			return t, err
		}
	}
	return t, nil
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) header(ctx context.Context, q queryer, id int64) (Test, error) {
	var t Test
	var score sql.NullFloat64
	var correct, wrong, blank int
	err := q.QueryRowContext(ctx, `SELECT id, mode, status, penalty, time_limit, started_at, deadline, finished_at,
			correct, wrong, blank, score
		FROM tests WHERE id = ?`, id).Scan(&t.ID, &t.Mode, &t.Status, &t.Penalty, &t.TimeLimit, &t.StartedAt,
		&t.Deadline, &t.FinishedAt, &correct, &wrong, &blank, &score)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	} else if err != nil {
		return t, err
	}
	if t.Status == StatusFinished {
		r := Score(correct, wrong, blank, t.Penalty)
		t.Result = &r
	}
	if t.Status == StatusInProgress && t.Deadline != "" {
		if d, err := time.Parse(timeFormat, t.Deadline); err == nil {
			rem := max(0, int(d.Sub(s.now()).Seconds()))
			t.RemainingSec = &rem
		}
	}
	return t, nil
}

func (s *Store) expired(t Test) bool {
	if t.Deadline == "" {
		return false
	}
	d, err := time.Parse(timeFormat, t.Deadline)
	return err == nil && !s.now().Before(d)
}

func (s *Store) topicIDs(ctx context.Context, questionID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT topic_id FROM question_topics WHERE question_id = ?`, questionID)
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

type AnswerInput struct {
	Position int  `json:"position"`
	Chosen   *int `json:"chosen"` // nil clears the answer (exam mode only)
	TimeMs   int  `json:"time_ms"`
}

// Answer records the answer to one question. In practice mode it returns
// the solution; in exam mode it returns nil so nothing leaks before the end.
func (s *Store) Answer(ctx context.Context, testID int64, in AnswerInput) (*Solution, error) {
	if in.Chosen != nil && (*in.Chosen < 0 || *in.Chosen > 3) {
		return nil, content.ValidationError{"chosen": "Opción no válida"}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	t, err := s.header(ctx, tx, testID)
	if err != nil {
		return nil, err
	}
	if t.Status != StatusInProgress {
		return nil, ErrNotInProgress
	}
	if s.expired(t) {
		// Finishing happens outside this transaction, on the next Get.
		return nil, ErrExpired
	}
	if t.Mode == ModePractice && in.Chosen == nil {
		return nil, content.ValidationError{"chosen": "Elige una opción"}
	}

	var prev sql.NullInt64
	var sol Solution
	var revision int
	var questionID int64
	err = tx.QueryRowContext(ctx, `SELECT q.id, a.chosen, q.correct, q.revision, q.explanation, q.origin, s.title, q.source_ref, q.source_quote
		FROM attempts a JOIN questions q ON q.id = a.question_id JOIN sources s ON s.id = q.source_id
		WHERE a.test_id = ? AND a.position = ?`, testID, in.Position).Scan(
		&questionID, &prev, &sol.Correct, &revision, &sol.Explanation, &sol.Origin, &sol.SourceTitle, &sol.SourceRef, &sol.SourceQuote)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, content.ValidationError{"position": "La pregunta no pertenece al test"}
	} else if err != nil {
		return nil, err
	}
	if t.Mode == ModePractice && prev.Valid {
		return nil, ErrAlreadyAnswered
	}

	var isCorrect any
	answeredAt := ""
	if in.Chosen != nil {
		ok := *in.Chosen == sol.Correct
		sol.IsCorrect = &ok
		isCorrect = boolInt(ok)
		answeredAt = s.now().UTC().Format(timeFormat)
	}
	// Record the revision actually answered: the question may have been
	// edited since the test was created.
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET chosen = ?, is_correct = ?, answered_at = ?, revision = ?,
			time_ms = time_ms + ? WHERE test_id = ? AND position = ?`,
		in.Chosen, isCorrect, answeredAt, revision, max(0, min(in.TimeMs, 3_600_000)), testID, in.Position); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if t.Mode == ModeExam {
		return nil, nil
	}
	sol.TopicIDs, err = s.topicIDs(ctx, questionID)
	return &sol, err
}

// Finish closes the test and stores its score. Unanswered questions count
// as blank.
func (s *Store) Finish(ctx context.Context, testID int64) (Result, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()

	t, err := s.header(ctx, tx, testID)
	if err != nil {
		return Result{}, err
	}
	if t.Status != StatusInProgress {
		return Result{}, ErrNotInProgress
	}

	var correct, wrong, blank int
	if err := tx.QueryRowContext(ctx, `SELECT
			count(CASE WHEN is_correct = 1 THEN 1 END),
			count(CASE WHEN is_correct = 0 THEN 1 END),
			count(CASE WHEN chosen IS NULL THEN 1 END)
		FROM attempts WHERE test_id = ?`, testID).Scan(&correct, &wrong, &blank); err != nil {
		return Result{}, err
	}
	r := Score(correct, wrong, blank, t.Penalty)

	finishedAt := s.now().UTC()
	if s.expired(t) {
		// Time ran out while away: the test ended at the deadline.
		finishedAt, _ = time.Parse(timeFormat, t.Deadline)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tests SET status = 'finished', finished_at = ?, correct = ?, wrong = ?,
			blank = ?, score = ? WHERE id = ?`,
		finishedAt.Format(timeFormat), correct, wrong, blank, r.Score, testID); err != nil {
		return Result{}, err
	}
	return r, tx.Commit()
}

// Abandon closes a test without scoring it. Answers already given are kept
// as attempts.
func (s *Store) Abandon(ctx context.Context, testID int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tests SET status = 'abandoned', finished_at = ?
		WHERE id = ? AND status = 'in_progress'`, s.now().UTC().Format(timeFormat), testID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.header(ctx, s.db, testID); err != nil {
			return err
		}
		return ErrNotInProgress
	}
	return nil
}

type FlagInput struct {
	Position int    `json:"position"`
	Flagged  bool   `json:"flagged"`
	Note     string `json:"note"`
}

// Flag marks (or unmarks) the question at a test position as doubtful,
// which sends it to the review queue.
func (s *Store) Flag(ctx context.Context, testID int64, in FlagInput) error {
	note := strings.TrimSpace(in.Note)
	if !in.Flagged {
		note = ""
	}
	res, err := s.db.ExecContext(ctx, `UPDATE questions SET flagged = ?, flag_note = ?, updated_at = ?
		WHERE id = (SELECT question_id FROM attempts WHERE test_id = ? AND position = ?)`,
		boolInt(in.Flagged), note, s.now().UTC().Format(timeFormat), testID, in.Position)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

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
	Score      *float64 `json:"score"`
}

// List returns recent tests, newest first, optionally filtered by status.
func (s *Store) List(ctx context.Context, status Status, limit int) ([]Summary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.mode, t.status, t.started_at, t.finished_at, t.deadline, t.score,
			count(a.position), count(a.chosen), count(CASE WHEN a.is_correct = 1 THEN 1 END),
			count(CASE WHEN a.is_correct = 0 THEN 1 END)
		FROM tests t JOIN attempts a ON a.test_id = t.id
		WHERE ? = '' OR t.status = ?
		GROUP BY t.id ORDER BY t.id DESC LIMIT ?`, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var sm Summary
		var score sql.NullFloat64
		if err := rows.Scan(&sm.ID, &sm.Mode, &sm.Status, &sm.StartedAt, &sm.FinishedAt, &sm.Deadline, &score,
			&sm.Total, &sm.Answered, &sm.Correct, &sm.Wrong); err != nil {
			return nil, err
		}
		if score.Valid {
			sm.Score = &score.Float64
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func appendIDs(args []any, ids []int64) []any {
	for _, id := range ids {
		args = append(args, id)
	}
	return args
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
