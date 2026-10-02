package stats

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/quiz"
	"github.com/alexzafra13/tai_tests/internal/settings"
	"github.com/alexzafra13/tai_tests/internal/srs"
)

// The fixture has two topics: T1 with three official questions and T2 with
// one, all with B (1) as the correct answer and options in fixed order.
func setup(t *testing.T) (*sql.DB, *quiz.Store, *Store, []int64) {
	t.Helper()
	ctx := context.Background()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	cs := content.NewStore(d)
	syl, _ := content.ParseSyllabus(strings.NewReader(`{"blocks":[{"code":"B1","name":"Bloque","topics":[
		{"code":"T1","title":"Uno"},{"code":"T2","title":"Dos"}]}]}`))
	if _, err := cs.LoadSyllabus(ctx, syl); err != nil {
		t.Fatal(err)
	}
	blocks, _ := cs.Syllabus(ctx)
	var topics []int64
	for _, tp := range blocks[0].Topics {
		topics = append(topics, tp.ID)
	}
	src, _ := cs.CreateSource(ctx, content.SourceInput{Kind: content.KindINAPExam, Title: "e"})
	for i, topic := range []int64{topics[0], topics[0], topics[0], topics[1]} {
		if _, err := cs.CreateQuestion(ctx, content.QuestionInput{Stem: string(rune('a' + i)), Options: [4]string{"1", "2", "3", "4"},
			Correct: 1, FixedOrder: true, Origin: content.OriginOfficial, Author: content.AuthorImport, SourceID: src, SourceRef: "r",
			Status: content.StatusPublished, TopicIDs: []int64{topic}}); err != nil {
			t.Fatal(err)
		}
	}
	return d, quiz.NewStore(d, settings.NewStore(d), srs.NewStore(d)), NewStore(d), topics
}

func TestOverviewAndTopics(t *testing.T) {
	ctx := context.Background()
	_, q, s, topics := setup(t)
	const user = 1

	// Practice on T1: two right, one wrong.
	id, _ := q.Create(ctx, user, quiz.CreateInput{Mode: quiz.ModePractice, Count: 3, Filters: quiz.Filters{TopicIDs: []int64{topics[0]}}})
	for pos, chosen := range []int{1, 1, 0} {
		c := chosen
		if _, err := q.Answer(ctx, user, id, quiz.AnswerInput{Position: pos, Chosen: &c}); err != nil {
			t.Fatal(err)
		}
	}
	q.Finish(ctx, user, id)

	// Exam on T2 left blank: counts as answered, not correct.
	eid, _ := q.Create(ctx, user, quiz.CreateInput{Mode: quiz.ModeExam, Count: 1, Filters: quiz.Filters{TopicIDs: []int64{topics[1]}}})
	q.Finish(ctx, user, eid)

	o, err := s.Overview(ctx, user, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if o != (Overview{Answered: 4, Correct: 2, Wrong: 1, Blank: 1, TestsFinished: 2, StudyDays: 1}) {
		t.Errorf("overview = %+v", o)
	}

	ts, err := s.Topics(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 {
		t.Fatalf("topics = %+v", ts)
	}
	if ts[0].Answered != 3 || ts[0].Correct != 2 || ts[0].Available != 3 || ts[0].Official != 3 {
		t.Errorf("T1 = %+v", ts[0])
	}
	if ts[1].Answered != 1 || ts[1].Correct != 0 || ts[1].Official != 1 {
		t.Errorf("T2 = %+v", ts[1])
	}

	// Another user has no results.
	if o, _ := s.Overview(ctx, 2, time.UTC); o.Answered != 0 {
		t.Errorf("other user's overview = %+v", o)
	}
}

func TestTimelineHasEveryDay(t *testing.T) {
	ctx := context.Background()
	_, q, s, _ := setup(t)
	s.now = func() time.Time { return time.Now() }

	id, _ := q.Create(ctx, 1, quiz.CreateInput{Mode: quiz.ModePractice, Count: 1})
	one := 1
	q.Answer(ctx, 1, id, quiz.AnswerInput{Position: 0, Chosen: &one})

	days, err := s.Timeline(ctx, 1, 7, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 7 {
		t.Fatalf("days = %d, want 7", len(days))
	}
	last := days[6]
	if last.Date != time.Now().UTC().Format("2006-01-02") || last.Answered != 1 || last.Correct != 1 {
		t.Errorf("today = %+v", last)
	}
	if days[0].Answered != 0 {
		t.Errorf("a week ago = %+v, want empty", days[0])
	}
}

func TestDaysUseLocalTimeZone(t *testing.T) {
	ctx := context.Background()
	d, q, _, _ := setup(t)
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	s := NewStore(d)
	// 2 October, 00:30 in Madrid (CEST, UTC+2) is still 1 October in UTC.
	s.now = func() time.Time { return time.Date(2026, 10, 2, 9, 0, 0, 0, madrid) }

	id, _ := q.Create(ctx, 1, quiz.CreateInput{Mode: quiz.ModePractice, Count: 2})
	one := 1
	q.Answer(ctx, 1, id, quiz.AnswerInput{Position: 0, Chosen: &one})
	q.Answer(ctx, 1, id, quiz.AnswerInput{Position: 1, Chosen: &one})
	if _, err := d.Exec(`UPDATE attempts SET answered_at = CASE position
		WHEN 0 THEN '2026-10-01T22:30:00.000Z' ELSE '2026-10-01T21:30:00.000Z' END`); err != nil {
		t.Fatal(err)
	}

	days, err := s.Timeline(ctx, 1, 2, madrid)
	if err != nil {
		t.Fatal(err)
	}
	want := []Day{{Date: "2026-10-01", Answered: 1, Correct: 1}, {Date: "2026-10-02", Answered: 1, Correct: 1}}
	if len(days) != 2 || days[0] != want[0] || days[1] != want[1] {
		t.Errorf("timeline = %+v, want %+v", days, want)
	}
	if o, _ := s.Overview(ctx, 1, madrid); o.StudyDays != 2 {
		t.Errorf("study days = %d, want 2", o.StudyDays)
	}
}
