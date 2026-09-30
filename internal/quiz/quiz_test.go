package quiz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/settings"
)

type fixture struct {
	db      *sql.DB
	quiz    *Store
	content *content.Store
	topics  []int64 // B1-T01, B1-T02, B2-T01
	blocks  []int64
	lawID   int64
	examID  int64
	now     time.Time
}

const lawText = "Artículo 1. La Administración está obligada a dictar resolución expresa y a notificarla en todos los procedimientos."

// newFixture creates published questions: 3 law questions in B1-T01, 2
// official ones in B2-T01, plus one draft and one annulled that must never
// appear in tests.
func newFixture(t *testing.T) *fixture {
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
	f := &fixture{db: d, quiz: NewStore(d, settings.NewStore(d)), content: content.NewStore(d), now: time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)}
	f.quiz.now = func() time.Time { return f.now }
	// Keep options in their original order so tests can rely on B being
	// correct; shuffling has its own tests.
	f.quiz.shuffle = func(int, func(i, j int)) {}

	syl, _ := content.ParseSyllabus(strings.NewReader(`{"blocks":[
		{"code":"B1","name":"Uno","topics":[{"code":"B1-T01","title":"a"},{"code":"B1-T02","title":"b"}]},
		{"code":"B2","name":"Dos","topics":[{"code":"B2-T01","title":"c"}]}]}`))
	if _, err := f.content.LoadSyllabus(ctx, syl); err != nil {
		t.Fatal(err)
	}
	blocks, _ := f.content.Syllabus(ctx)
	for _, b := range blocks {
		f.blocks = append(f.blocks, b.ID)
		for _, tp := range b.Topics {
			f.topics = append(f.topics, tp.ID)
		}
	}
	if f.lawID, err = f.content.CreateSource(ctx, content.SourceInput{Kind: content.KindLaw, Title: "Ley", FullText: lawText}); err != nil {
		t.Fatal(err)
	}
	if f.examID, err = f.content.CreateSource(ctx, content.SourceInput{Kind: content.KindINAPExam, Title: "Examen"}); err != nil {
		t.Fatal(err)
	}

	for i := range 3 {
		f.addQuestion(t, content.OriginLaw, f.topics[0], content.StatusPublished, false, fmt.Sprintf("ley %d", i))
	}
	for i := range 2 {
		f.addQuestion(t, content.OriginOfficial, f.topics[2], content.StatusPublished, false, fmt.Sprintf("oficial %d", i))
	}
	f.addQuestion(t, content.OriginLaw, f.topics[0], content.StatusDraft, false, "borrador")
	f.addQuestion(t, content.OriginOfficial, f.topics[2], content.StatusPublished, true, "anulada")
	return f
}

func (f *fixture) addQuestion(t *testing.T, origin content.Origin, topic int64, status content.Status, annulled bool, stem string) int64 {
	t.Helper()
	in := content.QuestionInput{
		Stem:     stem,
		Options:  [4]string{"a", "b", "c", "d"},
		Correct:  1,
		Origin:   origin,
		Author:   content.AuthorManual,
		Status:   status,
		Annulled: annulled,
		TopicIDs: []int64{topic},
	}
	if origin == content.OriginOfficial {
		in.SourceID, in.SourceRef = f.examID, "2024 · 1"
	} else {
		in.SourceID, in.SourceRef, in.SourceQuote = f.lawID, "art. 1", "obligada a dictar resolución expresa"
	}
	id, err := f.content.CreateQuestion(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func intp(i int) *int { return &i }

func TestOnlyPublishedNotAnnulledQuestions(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	n, err := f.quiz.Available(ctx, Filters{})
	if err != nil || n != 5 {
		t.Fatalf("Available = %d, %v; want 5", n, err)
	}
	id, err := f.quiz.Create(ctx, CreateInput{Mode: ModePractice, Count: 50, Penalty: 1.0 / 3})
	if err != nil {
		t.Fatal(err)
	}
	test, _ := f.quiz.Get(ctx, id)
	if len(test.Items) != 5 {
		t.Fatalf("items = %d, want 5", len(test.Items))
	}
	for _, it := range test.Items {
		if it.Stem == "borrador" || it.Stem == "anulada" {
			t.Errorf("ineligible question %q in test", it.Stem)
		}
	}
}

func TestFilters(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	tests := []struct {
		name    string
		filters Filters
		want    int
	}{
		{"topic", Filters{TopicIDs: []int64{f.topics[0]}}, 3},
		{"empty topic", Filters{TopicIDs: []int64{f.topics[1]}}, 0},
		{"block", Filters{BlockIDs: []int64{f.blocks[1]}}, 2},
		{"topic OR block", Filters{TopicIDs: []int64{f.topics[0]}, BlockIDs: []int64{f.blocks[1]}}, 5},
		{"origin", Filters{Origins: []content.Origin{content.OriginOfficial}}, 2},
		{"source", Filters{SourceIDs: []int64{f.lawID}}, 3},
		{"question ids", Filters{QuestionIDs: []int64{1, 2, 6}}, 2}, // 6 is a draft
		{"block AND origin", Filters{BlockIDs: []int64{f.blocks[0]}, Origins: []content.Origin{content.OriginOfficial}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if n, err := f.quiz.Available(ctx, tt.filters); err != nil || n != tt.want {
				t.Errorf("Available = %d, %v; want %d", n, err, tt.want)
			}
		})
	}

	_, err := f.quiz.Create(ctx, CreateInput{Mode: ModeExam, Count: 10, Filters: Filters{TopicIDs: []int64{f.topics[1]}}})
	if !errors.Is(err, ErrNoQuestions) {
		t.Errorf("empty selection: err = %v, want ErrNoQuestions", err)
	}
}

func TestPracticeRevealsAfterEachAnswer(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, err := f.quiz.Create(ctx, CreateInput{Mode: ModePractice, Count: 3, Penalty: 0})
	if err != nil {
		t.Fatal(err)
	}
	test, _ := f.quiz.Get(ctx, id)
	for _, it := range test.Items {
		if it.Solution != nil {
			t.Fatal("solution shown before answering")
		}
	}

	sol, err := f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(1), TimeMs: 4200})
	if err != nil {
		t.Fatal(err)
	}
	if sol == nil || sol.Correct != 1 || sol.IsCorrect == nil || !*sol.IsCorrect || sol.SourceTitle == "" {
		t.Fatalf("practice feedback = %+v", sol)
	}
	if _, err := f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(2)}); !errors.Is(err, ErrAlreadyAnswered) {
		t.Errorf("re-answer: err = %v, want ErrAlreadyAnswered", err)
	}

	test, _ = f.quiz.Get(ctx, id)
	if test.Items[0].Solution == nil || test.Items[1].Solution != nil {
		t.Error("solution should be shown only for the answered question")
	}
	if test.Items[0].TimeMs != 4200 {
		t.Errorf("time_ms = %d, want 4200", test.Items[0].TimeMs)
	}
}

func TestExamHidesUntilFinishAndScores(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, err := f.quiz.Create(ctx, CreateInput{Mode: ModeExam, Count: 5, Penalty: 1.0 / 3, TimeLimitMin: 10})
	if err != nil {
		t.Fatal(err)
	}

	// Correct answer is always B (1). Two right, one wrong, one changed
	// from wrong to right, one left blank after answering.
	answers := []struct {
		pos    int
		chosen *int
	}{{0, intp(1)}, {1, intp(1)}, {2, intp(0)}, {3, intp(3)}, {3, intp(1)}, {4, intp(2)}, {4, nil}}
	for _, a := range answers {
		sol, err := f.quiz.Answer(ctx, id, AnswerInput{Position: a.pos, Chosen: a.chosen})
		if err != nil {
			t.Fatal(err)
		}
		if sol != nil {
			t.Fatal("exam answer leaked the solution")
		}
	}
	test, _ := f.quiz.Get(ctx, id)
	for _, it := range test.Items {
		if it.Solution != nil {
			t.Fatal("exam shows solutions before finishing")
		}
	}
	if test.RemainingSec == nil || *test.RemainingSec != 600 {
		t.Errorf("remaining = %v, want 600", test.RemainingSec)
	}

	r, err := f.quiz.Finish(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Correct != 3 || r.Wrong != 1 || r.Blank != 1 {
		t.Fatalf("result = %+v, want 3/1/1", r)
	}
	// (3 - 1/3) / 5 on the default scale of 100 = 53.33, above the pass mark.
	if r.Score != 53.33 || !r.Passed {
		t.Errorf("score = %v passed = %v, want 53.33 passed", r.Score, r.Passed)
	}

	test, _ = f.quiz.Get(ctx, id)
	if test.Status != StatusFinished || test.Result == nil || test.Items[0].Solution == nil {
		t.Fatalf("finished test: %+v", test)
	}
	if test.Items[4].Solution.IsCorrect != nil {
		t.Error("blank answer should have nil is_correct")
	}
	if _, err := f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(0)}); !errors.Is(err, ErrNotInProgress) {
		t.Errorf("answer after finish: err = %v", err)
	}
}

func TestDeadlineFinishesAtDeadline(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, err := f.quiz.Create(ctx, CreateInput{Mode: ModeExam, Count: 2, Penalty: 0.25, TimeLimitMin: 30})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(1)}); err != nil {
		t.Fatal(err)
	}

	// The phone was locked and the user came back two hours later.
	f.now = f.now.Add(2 * time.Hour)
	if _, err := f.quiz.Answer(ctx, id, AnswerInput{Position: 1, Chosen: intp(1)}); !errors.Is(err, ErrExpired) {
		t.Fatalf("late answer: err = %v, want ErrExpired", err)
	}
	test, err := f.quiz.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if test.Status != StatusFinished || test.Result.Correct != 1 || test.Result.Blank != 1 {
		t.Fatalf("expired test: status %s result %+v", test.Status, test.Result)
	}
	if want := "2026-05-23T10:30:00.000Z"; test.FinishedAt != want {
		t.Errorf("finished_at = %s, want deadline %s", test.FinishedAt, want)
	}
}

func TestResumeKeepsOrderAndAnswers(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, _ := f.quiz.Create(ctx, CreateInput{Mode: ModeExam, Count: 5, Penalty: 0})
	first, _ := f.quiz.Get(ctx, id)
	f.quiz.Answer(ctx, id, AnswerInput{Position: 2, Chosen: intp(3)})

	again, _ := f.quiz.Get(ctx, id)
	for i := range first.Items {
		if first.Items[i].QuestionID != again.Items[i].QuestionID {
			t.Fatal("question order changed on reload")
		}
	}
	if again.Items[2].Chosen == nil || *again.Items[2].Chosen != 3 {
		t.Error("answer lost on reload")
	}

	list, _ := f.quiz.List(ctx, StatusInProgress, 10)
	if len(list) != 1 || list[0].Answered != 1 || list[0].Total != 5 {
		t.Errorf("in-progress list = %+v", list)
	}
}

func TestFlagAndRevisionAndHistoryProtection(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, _ := f.quiz.Create(ctx, CreateInput{Mode: ModePractice, Count: 1, Filters: Filters{Origins: []content.Origin{content.OriginLaw}}})
	test, _ := f.quiz.Get(ctx, id)
	qid := test.Items[0].QuestionID

	if err := f.quiz.Flag(ctx, id, FlagInput{Position: 0, Flagged: true, Note: "¿no era el art. 2?"}); err != nil {
		t.Fatal(err)
	}
	q, _ := f.content.Question(ctx, qid)
	if !q.Flagged || q.FlagNote != "¿no era el art. 2?" {
		t.Errorf("flag not stored: %+v", q)
	}

	// Edit the question after the test was created: the answer records the
	// revision actually seen.
	in := content.QuestionInput{Stem: q.Stem + " (corregida)", Options: q.Options, Correct: q.Correct, Origin: q.Origin,
		Author: q.Author, SourceID: q.SourceID, SourceRef: q.SourceRef, SourceQuote: q.SourceQuote, Status: q.Status,
		TopicIDs: q.TopicIDs}
	if err := f.content.UpdateQuestion(ctx, qid, in); err != nil {
		t.Fatal(err)
	}
	f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(1)})
	var revision int
	f.db.QueryRow(`SELECT revision FROM attempts WHERE test_id = ? AND position = 0`, id).Scan(&revision)
	if revision != 2 {
		t.Errorf("attempt revision = %d, want 2", revision)
	}

	if err := f.content.DeleteQuestion(ctx, qid); !errors.Is(err, content.ErrHasHistory) {
		t.Errorf("deleting an answered question: err = %v, want ErrHasHistory", err)
	}
}

func TestCreateValidation(t *testing.T) {
	f := newFixture(t)
	_, err := f.quiz.Create(context.Background(), CreateInput{Mode: "x", Count: 0, Penalty: 2})
	var v content.ValidationError
	if !errors.As(err, &v) || v["mode"] == "" || v["count"] == "" || v["penalty"] == "" {
		t.Fatalf("err = %v", err)
	}
}

func TestShuffledOptionsMapToOriginal(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	f.quiz.shuffle = reverse // options "a b c d" are shown as "d c b a"
	id, err := f.quiz.Create(ctx, CreateInput{Mode: ModePractice, Count: 1, Filters: Filters{Origins: []content.Origin{content.OriginLaw}}})
	if err != nil {
		t.Fatal(err)
	}
	test, _ := f.quiz.Get(ctx, id)
	if test.Items[0].Options != [4]string{"d", "c", "b", "a"} {
		t.Fatalf("shown options = %v", test.Items[0].Options)
	}

	// The correct answer is original B, now shown third (index 2).
	sol, err := f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(2)})
	if err != nil {
		t.Fatal(err)
	}
	if sol.Correct != 2 || !*sol.IsCorrect {
		t.Fatalf("solution = %+v, want correct shown at 2", sol)
	}
	var stored int
	f.db.QueryRow(`SELECT chosen FROM attempts WHERE test_id = ?`, id).Scan(&stored)
	if stored != 1 {
		t.Errorf("stored chosen = %d, want original index 1", stored)
	}
	test, _ = f.quiz.Get(ctx, id)
	if *test.Items[0].Chosen != 2 || test.Items[0].Solution.Correct != 2 {
		t.Errorf("reloaded item = %+v", test.Items[0])
	}
}

func TestScoringSettings(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	sc, err := f.quiz.Scoring(ctx)
	if err != nil || sc != DefaultScoring {
		t.Fatalf("default scoring = %+v, %v", sc, err)
	}
	if err := f.quiz.SetScoring(ctx, ScoringSettings{Scale: Scale{Max: 10, PassMark: 20}}); err == nil {
		t.Error("pass mark above max accepted")
	}

	id, _ := f.quiz.Create(ctx, CreateInput{Mode: ModeExam, Count: 2, Penalty: 0})
	f.quiz.Answer(ctx, id, AnswerInput{Position: 0, Chosen: intp(1)})
	f.quiz.Finish(ctx, id)

	// Changing the scale re-expresses past results.
	if err := f.quiz.SetScoring(ctx, ScoringSettings{Scale: Scale{Max: 10, PassMark: 6}, DefaultPenalty: 0.25}); err != nil {
		t.Fatal(err)
	}
	test, _ := f.quiz.Get(ctx, id)
	if test.Result.Score != 5 || test.Result.Max != 10 || test.Result.Passed {
		t.Errorf("result on new scale = %+v", test.Result)
	}
	list, _ := f.quiz.List(ctx, StatusFinished, 5)
	if *list[0].Score != 5 || list[0].Passed {
		t.Errorf("summary on new scale = %+v", list[0])
	}
}
