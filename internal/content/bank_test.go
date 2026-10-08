package content

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

const sampleBank = `{
  "source": {"kind": "inap_exam", "title": "TAI ingreso libre · OEP 2024", "reference": "INAP-TAI-L-OEP2024",
    "url": "https://sede.inap.gob.es/"},
  "questions": [
    {"key": "P-1", "source_ref": "OEP 2024 · nº 1", "stem": "¿Primera?",
      "options": ["Uno", "Dos", "Tres", "Cuatro"], "correct": "b"},
    {"key": "P-2", "source_ref": "OEP 2024 · nº 2", "stem": "¿Segunda?",
      "options": ["Uno", "Dos", "Tres", "Cuatro"], "correct": "d", "annulled": true,
      "explanation": "Pregunta anulada en la plantilla definitiva."},
    {"key": "P-3", "source_ref": "OEP 2024 · nº 3", "stem": "¿Tercera?",
      "options": ["Uno", "Uno", "Tres", "Cuatro"], "correct": "a"}
  ]
}`

func readSampleBank(t *testing.T) []BankFile {
	t.Helper()
	files, err := ReadBank(fstest.MapFS{"oep2024.json": {Data: []byte(sampleBank)}})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestLoadBankIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	files := readSampleBank(t)

	res, err := s.LoadBank(ctx, files)
	if err != nil {
		t.Fatal(err)
	}
	// P-3 repeats an option: reported, not loaded.
	if res.Added != 2 || res.Existing != 0 || len(res.Problems) != 1 || res.Problems[0].Key != "P-3" {
		t.Fatalf("first load: %+v", res)
	}

	page, err := s.ListQuestions(ctx, QuestionFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("got %d questions", page.Total)
	}
	for _, q := range page.Items {
		if q.Status != StatusDraft || q.Author != AuthorImport || q.Origin != OriginOfficial ||
			q.SourceTitle != "TAI ingreso libre · OEP 2024" {
			t.Errorf("question %+v", q)
		}
		if q.SourceRef == "OEP 2024 · nº 2" && (!q.Annulled || q.Correct != 3) {
			t.Errorf("annulled question %+v", q)
		}
	}

	// Discard one and delete the other: reloading brings neither back.
	byRef := map[string]int64{}
	for _, q := range page.Items {
		byRef[q.SourceRef] = q.ID
	}
	if _, err := s.Discard(ctx, byRef["OEP 2024 · nº 1"]); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteQuestion(ctx, byRef["OEP 2024 · nº 2"]); err != nil {
		t.Fatal(err)
	}
	res, err = s.LoadBank(ctx, files)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || res.Existing != 2 || len(res.Problems) != 1 {
		t.Fatalf("second load: %+v", res)
	}
	if page, _ := s.ListQuestions(ctx, QuestionFilter{}); page.Total != 0 {
		t.Errorf("reload brought back %d questions", page.Total)
	}
	sources, _ := s.ListSources(ctx, KindINAPExam)
	if len(sources) != 1 {
		t.Errorf("got %d sources, want the one created on the first load", len(sources))
	}
}

func TestLoadBankUsesExistingSource(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	id, err := s.CreateSource(ctx, SourceInput{Kind: KindINAPExam, Title: "Mi título", Reference: "INAP-TAI-L-OEP2024"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadBank(ctx, readSampleBank(t)); err != nil {
		t.Fatal(err)
	}
	if src, _ := s.Source(ctx, id); src.Questions != 2 || src.Title != "Mi título" {
		t.Errorf("source %+v", src)
	}
}

func TestParseBankFileRejects(t *testing.T) {
	for name, body := range map[string]string{
		"no reference": `{"source": {"kind": "inap_exam", "title": "x"}, "questions": []}`,
		"law source":   `{"source": {"kind": "law", "title": "x", "reference": "r"}, "questions": []}`,
		"repeated key": `{"source": {"kind": "inap_exam", "title": "x", "reference": "r"}, "questions": [
			{"key": "a", "correct": "a"}, {"key": "a", "correct": "a"}]}`,
		"bad letter": `{"source": {"kind": "inap_exam", "title": "x", "reference": "r"}, "questions": [
			{"key": "a", "correct": "e"}]}`,
		"unknown field": `{"source": {"kind": "inap_exam", "title": "x", "reference": "r"}, "questions": [], "extra": 1}`,
		"article without law": `{"source": {"kind": "inap_exam", "title": "x", "reference": "r"}, "questions": [
			{"key": "a", "correct": "a", "articles": [{"section": "a1"}]}]}`,
		"page without https": `{"source": {"kind": "inap_exam", "title": "x", "reference": "r"}, "questions": [
			{"key": "a", "correct": "a", "articles": [{"title": "Ficha", "url": "http://x"}]}]}`,
	} {
		if _, err := ParseBankFile([]byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		} else if !strings.HasPrefix(err.Error(), "bank:") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestLoadBankPublishesCheckedQuestions(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	loadTestSyllabus(t, s)
	files, err := ReadBank(fstest.MapFS{"x.json": {Data: []byte(`{
  "source": {"kind": "inap_exam", "title": "Examen", "reference": "REF"},
  "questions": [
    {"key": "1", "source_ref": "nº 1", "stem": "¿Una?", "options": ["A", "B", "C", "D"], "correct": "a",
      "topics": ["B1-T01"], "status": "published"},
    {"key": "2", "source_ref": "nº 2", "stem": "¿Dos?", "options": ["A", "B", "C", "D"], "correct": "a",
      "topics": ["B9-T99"], "status": "published"},
    {"key": "3", "source_ref": "nº 3", "stem": "¿Tres?", "options": ["A", "B", "C", "D"], "correct": "a",
      "topics": ["B1-T02"]}
  ]}`)}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.LoadBank(ctx, files)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 3 || res.Published != 1 {
		t.Fatalf("result %+v", res)
	}
	page, _ := s.ListQuestions(ctx, QuestionFilter{})
	got := map[string]Question{}
	for _, q := range page.Items {
		got[q.SourceRef] = q
	}
	// Unknown topic codes or no published status: the question waits in Review.
	if q := got["nº 1"]; q.Status != StatusPublished || len(q.TopicIDs) != 1 {
		t.Errorf("checked question %+v", q)
	}
	if q := got["nº 2"]; q.Status != StatusDraft || len(q.TopicIDs) != 0 {
		t.Errorf("unknown topic %+v", q)
	}
	if q := got["nº 3"]; q.Status != StatusDraft || len(q.TopicIDs) != 1 {
		t.Errorf("unchecked question %+v", q)
	}
}

func TestLoadBankUpdatesUneditedQuestions(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	loadTestSyllabus(t, s)
	file := func(topic, explanation string) BankFile {
		q := func(key string) BankQuestion {
			return BankQuestion{Key: key, SourceRef: "nº " + key, Stem: "¿Pregunta " + key + "?",
				Options: [4]string{"A", "B", "C", "D"}, Correct: "a", Topics: []string{topic}, Status: StatusPublished,
				Explanation: explanation}
		}
		return BankFile{Source: BankSource{Kind: KindINAPExam, Title: "Examen", Reference: "REF"},
			Questions: []BankQuestion{q("1"), q("2")}}
	}
	if _, err := s.LoadBank(ctx, []BankFile{file("B1-T01", "")}); err != nil {
		t.Fatal(err)
	}
	page, _ := s.ListQuestions(ctx, QuestionFilter{})
	byRef := map[string]Question{}
	for _, q := range page.Items {
		byRef[q.SourceRef] = q
	}
	// Someone edits question 2 here: the bank leaves it alone from then on.
	edited := byRef["nº 2"]
	in := QuestionInput{Stem: edited.Stem, Options: edited.Options, Correct: edited.Correct, Explanation: "Mía.",
		Origin: edited.Origin, Author: edited.Author, SourceID: edited.SourceID, SourceRef: edited.SourceRef,
		Status: edited.Status, TopicIDs: edited.TopicIDs}
	if err := s.UpdateQuestion(ctx, edited.ID, in); err != nil {
		t.Fatal(err)
	}

	res, err := s.LoadBank(ctx, []BankFile{file("B1-T02", "Nota del banco.")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || res.Updated != 1 {
		t.Errorf("result %+v", res)
	}
	blocks, _ := s.Syllabus(ctx)
	t2 := blocks[0].Topics[1].ID
	q1, _ := s.Question(ctx, byRef["nº 1"].ID)
	if q1.Explanation != "Nota del banco." || len(q1.TopicIDs) != 1 || q1.TopicIDs[0] != t2 {
		t.Errorf("unedited question %+v", q1)
	}
	q2, _ := s.Question(ctx, edited.ID)
	if q2.Explanation != "Mía." || q2.TopicIDs[0] == t2 {
		t.Errorf("edited question %+v", q2)
	}
	// Nothing left to change: a third start updates nothing.
	if res, _ := s.LoadBank(ctx, []BankFile{file("B1-T02", "Nota del banco.")}); res.Updated != 0 {
		t.Errorf("third load %+v", res)
	}
}
