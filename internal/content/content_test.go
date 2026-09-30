package content

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alexzafra13/tai_tests/internal/db"
)

const lawText = `Artículo 21. Obligación de resolver.
1. La Administración está obligada a dictar resolución expresa y a notificarla en todos los procedimientos cualquiera que sea su forma de iniciación.
2. El plazo máximo en el que debe notificarse la resolución expresa será el fijado por la norma reguladora del correspondiente procedimiento. Este plazo no podrá exceder de seis meses salvo que una norma con rango de Ley establezca uno mayor.`

func newStore(t *testing.T) *Store {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return NewStore(d)
}

var testSyllabus = `{
  "name": "Test", "reference": "BOE-TEST",
  "blocks": [
    {"code": "B1", "name": "Bloque uno", "topics": [
      {"code": "B1-T01", "number": 1, "title": "Tema uno"},
      {"code": "B1-T02", "number": 2, "title": "Tema dos"}
    ]},
    {"code": "B2", "name": "Bloque dos", "topics": [
      {"code": "B2-T01", "number": 1, "title": "Tema tres"}
    ]}
  ]
}`

func loadTestSyllabus(t *testing.T, s *Store) []Block {
	t.Helper()
	f, err := ParseSyllabus(strings.NewReader(testSyllabus))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadSyllabus(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	blocks, err := s.Syllabus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return blocks
}

func TestSyllabusReloadKeepsIDsAndDeactivates(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	before := loadTestSyllabus(t, s)
	if len(before) != 2 || len(before[0].Topics) != 2 {
		t.Fatalf("unexpected syllabus: %+v", before)
	}
	firstID := before[0].Topics[0].ID

	// New call: topic B1-T02 disappears, B1-T01 is renamed.
	f, err := ParseSyllabus(strings.NewReader(`{"name":"v2","reference":"","blocks":[
		{"code":"B1","name":"Bloque uno","topics":[{"code":"B1-T01","number":1,"title":"Tema uno (nuevo)"}]},
		{"code":"B2","name":"Bloque dos","topics":[{"code":"B2-T01","number":1,"title":"Tema tres"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.LoadSyllabus(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deactivated != 1 {
		t.Errorf("Deactivated = %d, want 1", res.Deactivated)
	}
	after, _ := s.Syllabus(ctx)
	if got := after[0].Topics; len(got) != 1 || got[0].ID != firstID || got[0].Title != "Tema uno (nuevo)" {
		t.Errorf("after reload: %+v", got)
	}
}

func TestParseSyllabusRejectsDuplicates(t *testing.T) {
	_, err := ParseSyllabus(strings.NewReader(`{"blocks":[{"code":"B1","name":"x","topics":[
		{"code":"T1","title":"a"},{"code":"T1","title":"b"}]}]}`))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err = %v, want duplicate code error", err)
	}
}

type fixture struct {
	s        *Store
	lawID    int64
	examID   int64
	techID   int64
	topicIDs []int64
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	blocks := loadTestSyllabus(t, s)
	f := fixture{s: s}
	for _, b := range blocks {
		for _, tp := range b.Topics {
			f.topicIDs = append(f.topicIDs, tp.ID)
		}
	}
	var err error
	if f.lawID, err = s.CreateSource(ctx, SourceInput{Kind: KindLaw, Title: "Ley 39/2015", Reference: "BOE-A-2015-10565", FullText: lawText}); err != nil {
		t.Fatal(err)
	}
	if f.examID, err = s.CreateSource(ctx, SourceInput{Kind: KindINAPExam, Title: "TAI libre 2024", Reference: "TAI-L-2024"}); err != nil {
		t.Fatal(err)
	}
	if f.techID, err = s.CreateSource(ctx, SourceInput{Kind: KindTechnicalDoc, Title: "RFC sin texto"}); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f fixture) lawQuestion() QuestionInput {
	return QuestionInput{
		Stem:        "Según la Ley 39/2015, el plazo máximo para notificar la resolución expresa no podrá exceder, con carácter general, de:",
		Options:     [4]string{"Tres meses", "Seis meses", "Un año", "Dos meses"},
		Correct:     1,
		Origin:      OriginLaw,
		Author:      AuthorManual,
		SourceID:    f.lawID,
		SourceRef:   "art. 21.2",
		SourceQuote: "Este plazo no podrá exceder de seis meses",
		Status:      StatusPublished,
		TopicIDs:    []int64{f.topicIDs[0]},
	}
}

func fieldErr(t *testing.T, err error, field string) string {
	t.Helper()
	var v ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	msg, ok := v[field]
	if !ok {
		t.Fatalf("no error for %q in %v", field, v)
	}
	return msg
}

func TestCreateAndGetQuestion(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, err := f.s.CreateQuestion(ctx, f.lawQuestion())
	if err != nil {
		t.Fatal(err)
	}
	q, err := f.s.Question(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if q.Options[1] != "Seis meses" || q.SourceTitle != "Ley 39/2015" || len(q.TopicIDs) != 1 {
		t.Errorf("unexpected question: %+v", q)
	}

	page, err := f.s.ListQuestions(ctx, QuestionFilter{TopicID: f.topicIDs[0], Text: "seis"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != id {
		t.Errorf("list = %+v", page)
	}
	blocks, _ := f.s.Syllabus(ctx)
	if blocks[0].Topics[0].Published != 1 {
		t.Errorf("published count = %d, want 1", blocks[0].Topics[0].Published)
	}
}

func TestQuestionValidation(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	tests := []struct {
		name   string
		mutate func(*QuestionInput)
		field  string
	}{
		{"quote not in source", func(q *QuestionInput) { q.SourceQuote = "Este plazo no podrá exceder de tres meses" }, "source_quote"},
		{"quote missing for law", func(q *QuestionInput) { q.SourceQuote = "" }, "source_quote"},
		{"quote too short", func(q *QuestionInput) { q.SourceQuote = "seis meses" }, "source_quote"},
		{"no source", func(q *QuestionInput) { q.SourceID = 0 }, "source_id"},
		{"source kind mismatch", func(q *QuestionInput) { q.SourceID = f.examID }, "source_id"},
		{"source without text", func(q *QuestionInput) { q.Origin = OriginTechnical; q.SourceID = f.techID }, "source_quote"},
		{"no source ref", func(q *QuestionInput) { q.SourceRef = "  " }, "source_ref"},
		{"empty option", func(q *QuestionInput) { q.Options[2] = "" }, "options.2"},
		{"duplicate option", func(q *QuestionInput) { q.Options[3] = " seis  meses " }, "options.3"},
		{"published without topic", func(q *QuestionInput) { q.TopicIDs = nil }, "topic_ids"},
		{"unknown topic", func(q *QuestionInput) { q.TopicIDs = []int64{9999} }, "topic_ids"},
		{"bad correct", func(q *QuestionInput) { q.Correct = 4 }, "correct"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := f.lawQuestion()
			tt.mutate(&in)
			_, err := f.s.CreateQuestion(ctx, in)
			fieldErr(t, err, tt.field)
		})
	}
}

func TestOfficialQuestionNeedsNoQuote(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	in := f.lawQuestion()
	in.Origin, in.Author, in.SourceID, in.SourceRef, in.SourceQuote = OriginOfficial, AuthorImport, f.examID, "2024 · nº 17", ""
	in.Status = StatusDraft
	in.TopicIDs = nil
	if _, err := f.s.CreateQuestion(ctx, in); err != nil {
		t.Fatalf("official draft without quote/topics: %v", err)
	}
}

func TestUpdateSourceProtectsQuotes(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	if _, err := f.s.CreateQuestion(ctx, f.lawQuestion()); err != nil {
		t.Fatal(err)
	}

	in := SourceInput{Kind: KindLaw, Title: "Ley 39/2015", Reference: "BOE-A-2015-10565",
		FullText: strings.Replace(lawText, "seis meses", "tres meses", 1)}
	fieldErr(t, f.s.UpdateSource(ctx, f.lawID, in), "full_text")

	in.FullText = lawText + "\n3. Nuevo apartado."
	if err := f.s.UpdateSource(ctx, f.lawID, in); err != nil {
		t.Fatalf("compatible update refused: %v", err)
	}

	if err := f.s.DeleteSource(ctx, f.lawID); !errors.Is(err, ErrInUse) {
		t.Fatalf("delete used source: err = %v, want ErrInUse", err)
	}
}

func TestDuplicateSourceReference(t *testing.T) {
	f := newFixture(t)
	_, err := f.s.CreateSource(context.Background(), SourceInput{Kind: KindLaw, Title: "Otra", Reference: "BOE-A-2015-10565"})
	fieldErr(t, err, "reference")
}
