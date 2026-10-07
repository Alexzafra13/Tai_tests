package content

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestExamsReplaceUnavailableWithReserves(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	topic := loadTestSyllabus(t, s)[0].Topics[0].ID

	// First part: 3 questions and 2 reserves; nº 2 is annulled, so reserve
	// nº 1 takes its place. Case I has nº 1 in review and its only reserve
	// annulled. Case II is complete.
	var qs []string
	add := func(key, ref, extra string) {
		qs = append(qs, fmt.Sprintf(`{"key": %q, "source_ref": "OEP 2024 · %s", "stem": "¿%s?",
			"options": ["Uno", "Dos", "Tres", "Cuatro"], "correct": "a", "topics": ["B1-T01"]%s}`, key, ref, key, extra))
	}
	pub := `, "status": "published"`
	add("P-1", "nº 1", pub)
	add("P-2", "nº 2", pub+`, "annulled": true`)
	add("P-3", "nº 3", pub)
	add("P-R1", "reserva nº 1", pub)
	add("P-R2", "reserva nº 2", pub)
	add("SI-1", "Supuesto I · nº 1", "")
	add("SI-2", "Supuesto I · nº 2", pub)
	add("SI-R1", "Supuesto I · reserva nº 1", pub+`, "annulled": true`)
	add("SII-1", "Supuesto II · nº 1", pub)
	add("SII-R1", "Supuesto II · reserva nº 1", pub)
	f, err := ParseBankFile([]byte(`{"source": {"kind": "inap_exam", "title": "OEP 2024", "reference": "OEP2024"},
		"questions": [` + strings.Join(qs, ",") + `]}`))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := s.LoadBank(ctx, []BankFile{f}); err != nil || res.Added != len(qs) {
		t.Fatalf("load: %+v, %v", res, err)
	}
	// Not from the exam paper: left out.
	exams, _ := s.Exams(ctx)
	if _, err := s.CreateQuestion(ctx, QuestionInput{Stem: "¿Otra?", Options: [4]string{"a", "b", "c", "d"},
		Origin: OriginOfficial, Author: AuthorManual, SourceID: exams[0].ID, SourceRef: "añadida",
		Status: StatusPublished, TopicIDs: []int64{topic}}); err != nil {
		t.Fatal(err)
	}

	exams, err = s.Exams(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(exams) != 1 {
		t.Fatalf("exams = %+v", exams)
	}
	stems := map[int64]string{}
	page, _ := s.ListQuestions(ctx, QuestionFilter{})
	for _, q := range page.Items {
		stems[q.ID] = strings.Trim(q.Stem, "¿?")
	}
	var got []string
	for _, p := range exams[0].Parts {
		var keys []string
		for _, id := range p.QuestionIDs {
			keys = append(keys, stems[id])
		}
		got = append(got, fmt.Sprintf("%s case=%v %v replaced=%d missing=%d annulled=%d unpublished=%d",
			p.Name, p.Case, keys, p.Replaced, p.Missing, p.Annulled, p.Unpublished))
	}
	want := []string{
		"Primera parte case=false [P-1 P-3 P-R1] replaced=1 missing=0 annulled=1 unpublished=0",
		"Supuesto I case=true [SI-2] replaced=0 missing=1 annulled=0 unpublished=1",
		"Supuesto II case=true [SII-1] replaced=0 missing=0 annulled=0 unpublished=0",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("parts:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
