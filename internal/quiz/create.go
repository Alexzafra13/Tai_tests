package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/validate"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/srs"
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
	// Due limits the test to the user's questions due for review, most
	// overdue first.
	Due bool `json:"due"`
	// Failed limits the test to questions whose last answer by the user was
	// wrong.
	Failed bool `json:"failed"`
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
	v := validate.Errors{}
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

// failedCondition matches questions whose most recent answer by the user
// was wrong.
const failedCondition = `q.id IN (SELECT a.question_id FROM attempts a JOIN tests t ON t.id = a.test_id
	WHERE t.user_id = ? AND a.answered_at <> '' GROUP BY a.question_id
	HAVING max(a.answered_at) = max(CASE WHEN a.is_correct = 0 THEN a.answered_at END))`

// eligibleWhere builds the condition for questions that may appear in a
// test for the user: published, not annulled, and matching the filters.
func (s *Store) eligibleWhere(userID int64, f Filters) (string, []any) {
	where := []string{"q.status = 'published'", "q.annulled = 0"}
	var args []any

	if f.Due {
		cond, a := srs.DueCondition(userID, s.now())
		where = append(where, cond)
		args = append(args, a...)
	}
	if f.Failed {
		where = append(where, failedCondition)
		args = append(args, userID)
	}

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
func (s *Store) Available(ctx context.Context, userID int64, f Filters) (int, error) {
	cond, args := s.eligibleWhere(userID, f)
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM questions q WHERE `+cond, args...).Scan(&n)
	return n, err
}

type pick struct {
	id         int64
	revision   int64
	options    [4]string
	fixedOrder bool
}

// Create draws up to in.Count random eligible questions and fixes both
// their order and the order of their options, so the test resumes exactly
// as it was.
func (s *Store) Create(ctx context.Context, userID int64, in CreateInput) (int64, error) {
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

	cond, args := s.eligibleWhere(userID, in.Filters)
	order := "random()"
	if in.Filters.Due {
		var orderArgs []any
		order, orderArgs = srs.DueOrder(userID)
		args = append(args, orderArgs...)
	}
	rows, err := tx.QueryContext(ctx, `SELECT q.id, q.revision, q.option_a, q.option_b, q.option_c, q.option_d, q.fixed_order
		FROM questions q WHERE `+cond+` ORDER BY `+order+` LIMIT ?`, append(args, in.Count)...)
	if err != nil {
		return 0, err
	}
	var picks []pick
	for rows.Next() {
		var p pick
		if err := rows.Scan(&p.id, &p.revision, &p.options[0], &p.options[1], &p.options[2], &p.options[3], &p.fixedOrder); err != nil {
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
	err = tx.QueryRowContext(ctx, `INSERT INTO tests (user_id, mode, config, penalty, time_limit, started_at, deadline)
		VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		userID, in.Mode, string(config), in.Penalty, in.TimeLimitMin*60, now.Format(timeFormat), deadline).Scan(&id)
	if err != nil {
		return 0, err
	}
	for i, p := range picks {
		order := newOptionOrder(p.options, p.fixedOrder, s.shuffle)
		if _, err := tx.ExecContext(ctx, `INSERT INTO attempts (test_id, position, question_id, revision, option_order)
			VALUES (?, ?, ?, ?, ?)`, id, i, p.id, p.revision, order.String()); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
