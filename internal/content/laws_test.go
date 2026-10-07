package content

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func testLaw(version string) LawFile {
	return LawFile{
		Reference: "BOE-A-1978-31229", Title: "Constitución Española", URL: "https://www.boe.es/",
		VersionDate: version, Aliases: []string{"Constitución Española", "CE"},
		Sections: []LawSection{
			{ID: "preambulo", Kind: SectionText, Title: "Preámbulo", Body: "La Nación española…"},
			{ID: "ti", Kind: SectionHeading, Level: 1, Title: "TÍTULO I. De los derechos"},
			{ID: "c1", Kind: SectionHeading, Level: 2, Title: "CAPÍTULO I. De los españoles"},
			{ID: "a13", Kind: SectionArticle, Title: "Artículo 13", Body: "1. Los extranjeros gozarán en España de las libertades públicas."},
			{ID: "a13-2", Kind: SectionArticle, Title: "Artículo 13 bis", Body: "Texto anterior.",
				Upcoming: &LawChange{Date: "2026-10-23", Title: "Artículo 13 bis. Nuevo", Body: "Texto nuevo."}},
			{ID: "tii", Kind: SectionHeading, Level: 1, Title: "TÍTULO II. De la Corona"},
			{ID: "a62", Kind: SectionArticle, Title: "Artículo 62", Body: "Corresponde al Rey: sancionar y promulgar las leyes."},
		},
	}
}

func lawTopicsFor(t *testing.T, s *Store) (int64, int64) {
	t.Helper()
	blocks := loadTestSyllabus(t, s)
	return blocks[0].Topics[0].ID, blocks[0].Topics[1].ID
}

func TestLoadLawsAndReadByTopic(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	t1, t2 := lawTopicsFor(t, s)
	topics := LawTopics{
		"B1-T01": {{Law: "BOE-A-1978-31229", Parts: []string{"ti"}}},
		"B1-T02": {{Law: "BOE-A-1978-31229"}},
	}
	res, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, topics)
	if err != nil || res.Added != 1 {
		t.Fatalf("%+v %v", res, err)
	}

	st, err := s.StudyTopic(ctx, t1, time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Laws) != 1 || strings.Join(st.Laws[0].Parts, "|") != "TÍTULO I. De los derechos" {
		t.Fatalf("study topic %+v", st)
	}
	lawID := st.Laws[0].SourceID

	// Topic 1 covers title I only: its headings and articles, nothing else.
	text, err := s.LawText(ctx, lawID, t1, time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, sec := range text.Sections {
		ids = append(ids, sec.ID)
	}
	if strings.Join(ids, ",") != "ti,c1,a13,a13-2" {
		t.Errorf("topic part %v", ids)
	}
	if up := text.Sections[3].Upcoming; up == nil || text.Sections[3].Body != "Texto anterior." {
		t.Errorf("before the change %+v", text.Sections[3])
	}

	// From the date of the change, the new wording is the one in force.
	text, _ = s.LawText(ctx, lawID, t2, time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC))
	if len(text.Sections) != 7 {
		t.Fatalf("whole law: %d sections", len(text.Sections))
	}
	if sec := text.Sections[4]; sec.Title != "Artículo 13 bis. Nuevo" || sec.Body != "Texto nuevo." || sec.Upcoming != nil {
		t.Errorf("after the change %+v", sec)
	}

	// The full text backs quotes, and reloading the same version is a no-op.
	src, _ := s.Source(ctx, lawID)
	if src.Kind != KindLaw || !strings.Contains(src.FullText, "Corresponde al Rey") {
		t.Errorf("source %+v", src)
	}
	res, _ = s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, topics)
	if res.Added != 0 || res.Updated != 0 {
		t.Errorf("reload %+v", res)
	}
}

func TestLoadLawsKeepsTextThatBacksQuotes(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	topicID, _ := lawTopicsFor(t, s)
	if _, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, nil); err != nil {
		t.Fatal(err)
	}
	srcs, _ := s.ListSources(ctx, KindLaw)
	_, err := s.CreateQuestion(ctx, QuestionInput{
		Stem: "¿Quién sanciona las leyes?", Options: [4]string{"El Rey", "Las Cortes", "El Gobierno", "El Senado"},
		Origin: OriginLaw, Author: AuthorManual, SourceID: srcs[0].ID, SourceRef: "art. 62",
		SourceQuote: "Corresponde al Rey: sancionar y promulgar las leyes", TopicIDs: []int64{topicID},
	})
	if err != nil {
		t.Fatal(err)
	}

	newer := testLaw("2026-09-01")
	newer.Sections[6].Body = "Corresponde al Rey otra cosa."
	res, err := s.LoadLaws(ctx, []LawFile{newer}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 0 || len(res.Problems) != 1 {
		t.Fatalf("result %+v", res)
	}
	if src, _ := s.Source(ctx, srcs[0].ID); src.VersionDate != "2026-05-20" {
		t.Errorf("text replaced: %s", src.VersionDate)
	}
}

func TestLawQuestionsLinkCitedArticles(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	topicID, _ := lawTopicsFor(t, s)
	if _, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, nil); err != nil {
		t.Fatal(err)
	}
	exam, err := s.CreateSource(ctx, SourceInput{Kind: KindINAPExam, Title: "Examen", Reference: "EX"})
	if err != nil {
		t.Fatal(err)
	}
	add := func(ref, stem string, options [4]string, correct int) {
		t.Helper()
		_, err := s.CreateQuestion(ctx, QuestionInput{Stem: stem, Options: options, Correct: correct, Origin: OriginOfficial,
			Author: AuthorImport, SourceID: exam, SourceRef: ref, Status: StatusPublished, TopicIDs: []int64{topicID}})
		if err != nil {
			t.Fatal(err)
		}
	}
	opts := [4]string{"Uno", "Dos", "Tres", "Cuatro"}
	add("1", "Según el artículo 62 de la Constitución Española, corresponde al Rey:", opts, 0)
	add("2", "De acuerdo con el art. 13 de la CE, los extranjeros:", opts, 0)
	// Ranges and articles in wrong options cite no single article.
	add("3", "¿Qué derecho de los artículos 15 a 29 de la Constitución Española NO es fundamental?", opts, 0)
	add("4", "La suspensión de derechos se regula en la CE en:", [4]string{"El artículo 55", "El artículo 52", "El artículo 58", "El artículo 99"}, 0)
	// Another law's article is not this one's.
	add("5", "Según el artículo 13 de la Ley 39/2015, los interesados:", opts, 0)

	srcs, _ := s.ListSources(ctx, KindLaw)
	text, err := s.LawText(ctx, srcs[0].ID, 0, time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, sec := range text.Sections {
		for _, q := range sec.Questions {
			got[sec.ID] = append(got[sec.ID], q.SourceRef)
		}
	}
	if strings.Join(got["a62"], ",") != "1" || strings.Join(got["a13"], ",") != "2" || len(got) != 2 {
		t.Errorf("by article %v", got)
	}
	var general []string
	for _, q := range text.Questions {
		general = append(general, q.SourceRef)
	}
	if strings.Join(general, ",") != "3,4" {
		t.Errorf("general %v", general)
	}
}

func TestQuestionArticlesLinkBack(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	t1, t2 := lawTopicsFor(t, s)
	topics := LawTopics{
		"B1-T01": {{Law: "BOE-A-1978-31229", Parts: []string{"ti"}}},
		"B1-T02": {{Law: "BOE-A-1978-31229"}},
	}
	if _, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, topics); err != nil {
		t.Fatal(err)
	}
	exam, err := s.CreateSource(ctx, SourceInput{Kind: KindINAPExam, Title: "Examen", Reference: "EX"})
	if err != nil {
		t.Fatal(err)
	}
	add := func(stem string, topic int64) int64 {
		t.Helper()
		id, err := s.CreateQuestion(ctx, QuestionInput{Stem: stem, Options: [4]string{"Uno", "Dos", "Tres", "Cuatro"},
			Origin: OriginOfficial, Author: AuthorImport, SourceID: exam, SourceRef: "1", Status: StatusPublished,
			TopicIDs: []int64{topic}})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	both := add("Caso práctico: el artículo 99 de la CE no cuenta.\n\nSegún el artículo 62 y el artículo 13 bis de la CE:", t1)
	title1 := add("Según el artículo 13 de la Constitución Española:", t2)
	none := add("Según el artículo 13 de la Ley 39/2015:", t1)

	links, err := s.QuestionArticles(ctx, []int64{both, title1, none})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, l := range links[both] {
		got = append(got, fmt.Sprintf("%s %s %d", l.Law, l.BlockID, l.TopicID))
	}
	// Article 62 is outside topic 1's part, so it opens in topic 2; the law
	// order wins over the order of the stem.
	if want := fmt.Sprintf("Constitución Española a13-2 %d|Constitución Española a62 %d", t1, t2); strings.Join(got, "|") != want {
		t.Errorf("practical case %q, want %q", strings.Join(got, "|"), want)
	}
	if l := links[title1]; len(l) != 1 || l[0].BlockID != "a13" || l[0].TopicID != t2 || l[0].Title != "Artículo 13" {
		t.Errorf("question's own topic %+v", l)
	}
	if len(links[none]) != 0 {
		t.Errorf("another law %+v", links[none])
	}
}

func TestBankArticlesReplaceCitations(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	t1, _ := lawTopicsFor(t, s)
	topics := LawTopics{"B1-T01": {{Law: "BOE-A-1978-31229"}}}
	if _, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, topics); err != nil {
		t.Fatal(err)
	}
	file := func(articles []BankArticle) BankFile {
		return BankFile{Source: BankSource{Kind: KindINAPExam, Title: "Examen", Reference: "EX"}, Questions: []BankQuestion{
			// The stem cites article 13, but the bank says article 62 answers it.
			{Key: "1", SourceRef: "nº 1", Stem: "Según el artículo 13 de la CE, ¿a quién corresponde sancionar las leyes?",
				Options: [4]string{"Al Rey", "B", "C", "D"}, Correct: "a", Topics: []string{"B1-T01"}, Status: StatusPublished,
				Articles: append(articles, BankArticle{Title: "Ficha del CTT", URL: "https://administracionelectronica.gob.es/ctt/x"})},
			{Key: "2", SourceRef: "nº 2", Stem: "¿Qué dice la Constitución Española?", Options: [4]string{"A", "B", "C", "D"},
				Correct: "a", Topics: []string{"B1-T01"}, Status: StatusPublished,
				Articles: []BankArticle{{Law: "BOE-A-1978-31229"}}},
		}}
	}
	if _, err := s.LoadBank(ctx, []BankFile{file([]BankArticle{{Law: "BOE-A-1978-31229", Section: "a13"}})}); err != nil {
		t.Fatal(err)
	}
	// A later bank changes the article of a question already loaded.
	if _, err := s.LoadBank(ctx, []BankFile{file([]BankArticle{{Law: "BOE-A-1978-31229", Section: "a62"}})}); err != nil {
		t.Fatal(err)
	}

	srcs, _ := s.ListSources(ctx, KindLaw)
	text, err := s.LawText(ctx, srcs[0].ID, 0, time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, sec := range text.Sections {
		for _, q := range sec.Questions {
			got[sec.ID] = append(got[sec.ID], q.SourceRef)
		}
	}
	if len(got) != 1 || strings.Join(got["a62"], ",") != "nº 1" {
		t.Errorf("by article %v", got)
	}
	if len(text.Questions) != 1 || text.Questions[0].SourceRef != "nº 2" {
		t.Errorf("general %+v", text.Questions)
	}

	page, _ := s.ListQuestions(ctx, QuestionFilter{SourceID: 0, Limit: 10})
	var ids []int64
	for _, q := range page.Items {
		ids = append(ids, q.ID)
	}
	links, err := s.QuestionArticles(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for _, ls := range links {
		for _, l := range ls {
			if l.URL != "" && l.Law != "Ficha del CTT" {
				t.Errorf("page %+v", l)
			}
			all = append(all, fmt.Sprintf("%s %d", l.BlockID, l.TopicID))
		}
	}
	if strings.Join(all, "|") != fmt.Sprintf("a62 %d| 0", t1) {
		t.Errorf("links %v", all)
	}
}
