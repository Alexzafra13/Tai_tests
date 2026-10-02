// Package stats summarizes a user's results: overall accuracy, accuracy per
// block and topic, progress over time, and which topics the official exams
// ask about most. It only reads; answers are recorded by package quiz.
package stats

import (
	"context"
	"database/sql"
	"time"
)

// Store reads statistics. Days are calendar days in the caller's time zone
// (the browser's), so an answer at 23:30 in Madrid counts for that day, not
// for the next one in UTC.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

const dateFormat = "2006-01-02"

// answered is the set of a user's answered attempts (blank answers in
// finished exams included: they count as not known).
const answered = `attempts a JOIN tests t ON t.id = a.test_id
	WHERE t.user_id = ? AND (a.answered_at <> '' OR (t.status = 'finished' AND t.mode = 'exam'))`

// answeredHour is the UTC hour ("YYYY-MM-DDTHH") an answer counts for: when
// it was given or, for blanks in an exam, when the exam was finished.
// Grouping by hour in SQL and converting to local days in Go is exact for
// time zones with whole-hour offsets (all of Spain's).
const answeredHour = `substr(CASE WHEN a.answered_at <> '' THEN a.answered_at ELSE t.finished_at END, 1, 13)`

type Overview struct {
	Answered      int `json:"answered"`
	Correct       int `json:"correct"`
	Wrong         int `json:"wrong"`
	Blank         int `json:"blank"`
	TestsFinished int `json:"tests_finished"`
	// StudyDays counts distinct days with at least one answer.
	StudyDays int `json:"study_days"`
}

func (s *Store) Overview(ctx context.Context, userID int64, loc *time.Location) (Overview, error) {
	var o Overview
	err := s.db.QueryRowContext(ctx, `SELECT count(*),
			count(CASE WHEN a.is_correct = 1 THEN 1 END),
			count(CASE WHEN a.is_correct = 0 THEN 1 END),
			count(CASE WHEN a.chosen IS NULL THEN 1 END)
		FROM `+answered, userID).Scan(&o.Answered, &o.Correct, &o.Wrong, &o.Blank)
	if err != nil {
		return o, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM tests WHERE user_id = ? AND status = 'finished'`, userID).
		Scan(&o.TestsFinished)
	if err != nil {
		return o, err
	}
	days, err := s.days(ctx, userID, time.Time{}, loc)
	o.StudyDays = len(days)
	return o, err
}

// TopicStats is one syllabus topic: the user's results on it, how much
// there is to practise, and how often official exams asked about it.
type TopicStats struct {
	BlockID   int64  `json:"block_id"`
	BlockName string `json:"block_name"`
	TopicID   int64  `json:"topic_id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Answered  int    `json:"answered"`
	Correct   int    `json:"correct"`
	// Available is the number of published questions on the topic.
	Available int `json:"available"`
	// Official is the number of official exam questions on the topic.
	Official int `json:"official"`
}

// Topics returns every active topic in syllabus order.
func (s *Store) Topics(ctx context.Context, userID int64) ([]TopicStats, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH mine AS (
			SELECT a.question_id, a.is_correct FROM `+answered+`
		)
		SELECT b.id, b.name, tp.id, tp.number, tp.title,
			(SELECT count(*) FROM mine m JOIN question_topics qt ON qt.question_id = m.question_id WHERE qt.topic_id = tp.id),
			(SELECT count(*) FROM mine m JOIN question_topics qt ON qt.question_id = m.question_id
				WHERE qt.topic_id = tp.id AND m.is_correct = 1),
			(SELECT count(*) FROM question_topics qt JOIN questions q ON q.id = qt.question_id
				WHERE qt.topic_id = tp.id AND q.status = 'published' AND q.annulled = 0),
			(SELECT count(*) FROM question_topics qt JOIN questions q ON q.id = qt.question_id
				WHERE qt.topic_id = tp.id AND q.origin = 'official' AND q.status <> 'discarded')
		FROM topics tp JOIN blocks b ON b.id = tp.block_id
		WHERE tp.active = 1 AND b.active = 1
		ORDER BY b.position, tp.position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TopicStats{}
	for rows.Next() {
		var t TopicStats
		if err := rows.Scan(&t.BlockID, &t.BlockName, &t.TopicID, &t.Number, &t.Title,
			&t.Answered, &t.Correct, &t.Available, &t.Official); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Day is the user's activity on one calendar day.
type Day struct {
	Date     string `json:"date"` // YYYY-MM-DD
	Answered int    `json:"answered"`
	Correct  int    `json:"correct"`
}

// Timeline returns one entry per day for the last days days, oldest first,
// including days without activity so charts have a continuous axis.
func (s *Store) Timeline(ctx context.Context, userID int64, days int, loc *time.Location) ([]Day, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	now := s.now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	from := today.AddDate(0, 0, -(days - 1))

	byDay, err := s.days(ctx, userID, from, loc)
	if err != nil {
		return nil, err
	}
	out := make([]Day, 0, days)
	for d := from; !d.After(today); d = d.AddDate(0, 0, 1) {
		day := byDay[d.Format(dateFormat)]
		day.Date = d.Format(dateFormat)
		out = append(out, day)
	}
	return out, nil
}

// days returns the user's activity per day in loc since from (the zero
// time for all of it), keyed by date.
func (s *Store) days(ctx context.Context, userID int64, from time.Time, loc *time.Location) (map[string]Day, error) {
	fromHour := ""
	if !from.IsZero() {
		fromHour = from.UTC().Format("2006-01-02T15")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+answeredHour+` AS hour,
			count(*), count(CASE WHEN a.is_correct = 1 THEN 1 END)
		FROM `+answered+` AND hour >= ?
		GROUP BY hour`, userID, fromHour)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Day{}
	for rows.Next() {
		var hour string
		var answered, correct int
		if err := rows.Scan(&hour, &answered, &correct); err != nil {
			return nil, err
		}
		t, err := time.ParseInLocation("2006-01-02T15", hour, time.UTC)
		if err != nil {
			return nil, err
		}
		key := t.In(loc).Format(dateFormat)
		d := out[key]
		d.Answered += answered
		d.Correct += correct
		out[key] = d
	}
	return out, rows.Err()
}
