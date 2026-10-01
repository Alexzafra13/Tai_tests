package quiz

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alexzafra13/tai_tests/internal/validate"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// Test is a test as the user sees it. Option indices (Item.Chosen,
// Solution.Correct, AnswerInput.Chosen) are display positions: options may
// be shuffled, and the mapping to the stored order stays inside this package.
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
	// Reported tells whether the current user has an open doubt report on
	// this question, with its note.
	Reported   bool   `json:"reported"`
	ReportNote string `json:"report_note,omitempty"`
	TimeMs     int    `json:"time_ms"`
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

// testRow is a tests row with the stored counts.
type testRow struct {
	Test
	userID                int64
	correct, wrong, blank int
}

// loadTest reads a test owned by userID. Other users' tests are reported
// as not found, so their existence is not revealed.
func (s *Store) loadTest(ctx context.Context, q queryer, userID, id int64) (testRow, error) {
	t := testRow{userID: userID}
	err := q.QueryRowContext(ctx, `SELECT id, mode, status, penalty, time_limit, started_at, deadline, finished_at,
			correct, wrong, blank
		FROM tests WHERE id = ? AND user_id = ?`, id, userID).Scan(&t.ID, &t.Mode, &t.Status, &t.Penalty, &t.TimeLimit,
		&t.StartedAt, &t.Deadline, &t.FinishedAt, &t.correct, &t.wrong, &t.blank)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) expired(t Test) bool {
	if t.Deadline == "" {
		return false
	}
	d, err := time.Parse(timeFormat, t.Deadline)
	return err == nil && !s.now().Before(d)
}

// Get returns the test with its questions. A timed test whose deadline has
// passed is finished first, so a test abandoned mid-exam scores as if time
// ran out.
func (s *Store) Get(ctx context.Context, userID, id int64) (Test, error) {
	row, err := s.loadTest(ctx, s.db, userID, id)
	if err != nil {
		return Test{}, err
	}
	if row.Status == StatusInProgress && s.expired(row.Test) {
		if _, err := s.Finish(ctx, userID, id); err != nil && !errors.Is(err, ErrNotInProgress) {
			return Test{}, err
		}
		if row, err = s.loadTest(ctx, s.db, userID, id); err != nil {
			return Test{}, err
		}
	}

	t := row.Test
	switch {
	case t.Status == StatusFinished:
		scoring, err := s.Scoring(ctx)
		if err != nil {
			return t, err
		}
		r := Score(row.correct, row.wrong, row.blank, t.Penalty, scoring.Scale)
		t.Result = &r
	case t.Status == StatusInProgress && t.Deadline != "":
		if d, err := time.Parse(timeFormat, t.Deadline); err == nil {
			rem := max(0, int(d.Sub(s.now()).Seconds()))
			t.RemainingSec = &rem
		}
	}

	t.Items, err = s.items(ctx, t, userID)
	return t, err
}

func (s *Store) items(ctx context.Context, t Test, userID int64) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.position, a.option_order, q.id, q.stem,
			q.option_a, q.option_b, q.option_c, q.option_d,
			a.chosen, a.is_correct, a.time_ms, r.id IS NOT NULL, COALESCE(r.note, ''),
			q.correct, q.explanation, q.origin, s.title, q.source_ref, q.source_quote
		FROM attempts a JOIN questions q ON q.id = a.question_id JOIN sources s ON s.id = q.source_id
		LEFT JOIN question_reports r ON r.question_id = q.id AND r.user_id = ? AND r.resolved_at = ''
		WHERE a.test_id = ? ORDER BY a.position`, userID, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		var orderStr string
		var options [4]string
		var chosen, isCorrect sql.NullInt64
		var sol Solution
		if err := rows.Scan(&it.Position, &orderStr, &it.QuestionID, &it.Stem,
			&options[0], &options[1], &options[2], &options[3],
			&chosen, &isCorrect, &it.TimeMs, &it.Reported, &it.ReportNote,
			&sol.Correct, &sol.Explanation, &sol.Origin, &sol.SourceTitle, &sol.SourceRef, &sol.SourceQuote); err != nil {
			return nil, err
		}
		order := parseOptionOrder(orderStr)
		it.Options = order.display(options)
		if chosen.Valid {
			c := order.toDisplay(int(chosen.Int64))
			it.Chosen = &c
		}
		sol.Correct = order.toDisplay(sol.Correct)
		if isCorrect.Valid {
			b := isCorrect.Int64 == 1
			sol.IsCorrect = &b
		}
		if t.Status != StatusInProgress || (t.Mode == ModePractice && it.Chosen != nil) {
			it.Solution = &sol
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	for i := range items {
		if items[i].Solution == nil {
			continue
		}
		if items[i].Solution.TopicIDs, err = s.topicIDs(ctx, items[i].QuestionID); err != nil {
			return nil, err
		}
	}
	return items, nil
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
	Chosen   *int `json:"chosen"` // display position; nil clears the answer (exam mode only)
	TimeMs   int  `json:"time_ms"`
}

// Answer records the answer to one question. In practice mode it returns
// the solution; in exam mode it returns nil so nothing leaks before the end.
func (s *Store) Answer(ctx context.Context, userID, testID int64, in AnswerInput) (*Solution, error) {
	if in.Chosen != nil && (*in.Chosen < 0 || *in.Chosen > 3) {
		return nil, validate.Errors{"chosen": "Opción no válida"}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	t, err := s.loadTest(ctx, tx, userID, testID)
	if err != nil {
		return nil, err
	}
	if t.Status != StatusInProgress {
		return nil, ErrNotInProgress
	}
	if s.expired(t.Test) {
		// Finishing happens outside this transaction, on the next Get.
		return nil, ErrExpired
	}
	if t.Mode == ModePractice && in.Chosen == nil {
		return nil, validate.Errors{"chosen": "Elige una opción"}
	}

	var prev sql.NullInt64
	var sol Solution
	var orderStr string
	var revision int
	var questionID int64
	err = tx.QueryRowContext(ctx, `SELECT q.id, a.chosen, a.option_order, q.correct, q.revision, q.explanation, q.origin,
			s.title, q.source_ref, q.source_quote
		FROM attempts a JOIN questions q ON q.id = a.question_id JOIN sources s ON s.id = q.source_id
		WHERE a.test_id = ? AND a.position = ?`, testID, in.Position).Scan(
		&questionID, &prev, &orderStr, &sol.Correct, &revision, &sol.Explanation, &sol.Origin,
		&sol.SourceTitle, &sol.SourceRef, &sol.SourceQuote)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, validate.Errors{"position": "La pregunta no pertenece al test"}
	} else if err != nil {
		return nil, err
	}
	if t.Mode == ModePractice && prev.Valid {
		return nil, ErrAlreadyAnswered
	}

	order := parseOptionOrder(orderStr)
	var chosenOrig, isCorrect any
	answeredAt := ""
	if in.Chosen != nil {
		orig := order.toOriginal(*in.Chosen)
		ok := orig == sol.Correct
		sol.IsCorrect = &ok
		chosenOrig, isCorrect = orig, boolInt(ok)
		answeredAt = s.now().UTC().Format(timeFormat)
	}
	sol.Correct = order.toDisplay(sol.Correct)

	// Record the revision actually answered: the question may have been
	// edited since the test was created.
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET chosen = ?, is_correct = ?, answered_at = ?, revision = ?,
			time_ms = time_ms + ? WHERE test_id = ? AND position = ?`,
		chosenOrig, isCorrect, answeredAt, revision, max(0, min(in.TimeMs, 3_600_000)), testID, in.Position); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if t.Mode == ModeExam {
		// Exam answers can still change; they are scheduled on finish.
		return nil, nil
	}
	if err := s.schedule(ctx, userID, questionID, sol.IsCorrect, in.TimeMs); err != nil {
		return nil, err
	}
	sol.TopicIDs, err = s.topicIDs(ctx, questionID)
	return &sol, err
}

// Finish closes the test and stores its net ratio. Unanswered questions
// count as blank.
func (s *Store) Finish(ctx context.Context, userID, testID int64) (Result, error) {
	scoring, err := s.Scoring(ctx)
	if err != nil {
		return Result{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()

	t, err := s.loadTest(ctx, tx, userID, testID)
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
	r := Score(correct, wrong, blank, t.Penalty, scoring.Scale)

	finishedAt := s.now().UTC()
	if s.expired(t.Test) {
		// Time ran out while away: the test ended at the deadline.
		finishedAt, _ = time.Parse(timeFormat, t.Deadline)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tests SET status = 'finished', finished_at = ?, correct = ?, wrong = ?,
			blank = ?, score = ? WHERE id = ?`,
		finishedAt.Format(timeFormat), correct, wrong, blank, r.Ratio, testID); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(); err != nil {
		return Result{}, err
	}
	if t.Mode == ModeExam {
		if err := s.scheduleExam(ctx, userID, testID); err != nil {
			return Result{}, err
		}
	}
	return r, nil
}

// Abandon closes a test without scoring it. Answers already given are kept
// as attempts.
func (s *Store) Abandon(ctx context.Context, userID, testID int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tests SET status = 'abandoned', finished_at = ?
		WHERE id = ? AND user_id = ? AND status = 'in_progress'`, s.now().UTC().Format(timeFormat), testID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.loadTest(ctx, s.db, userID, testID); err != nil {
			return err
		}
		return ErrNotInProgress
	}
	return nil
}

// QuestionAt returns the question at a position of the user's test, e.g.
// to report a doubt about it.
func (s *Store) QuestionAt(ctx context.Context, userID, testID int64, position int) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT a.question_id FROM attempts a JOIN tests t ON t.id = a.test_id
		WHERE a.test_id = ? AND t.user_id = ? AND a.position = ?`, testID, userID, position).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}
