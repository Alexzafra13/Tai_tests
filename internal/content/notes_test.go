package content

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const testNote = `{"topic": "B1-T01", "sections": [{"title": "La Corona", "points": [
  {"text": "Corresponde al **Rey** sancionar las leyes.",
   "refs": [{"law": "BOE-A-1978-31229", "section": "a62", "quote": "Corresponde al Rey: sancionar"},
     {"law": "BOE-A-1978-31229", "section": "a62", "quote": "sancionar y promulgar las leyes"}],
   "items": [{"text": "Y promulgarlas.", "refs": [{"law": "BOE-A-1978-31229", "section": "a62", "quote": "promulgar las leyes."}]}]},
  {"text": "Fuente oficial.", "refs": [{"title": "Constitución · BOE", "url": "https://www.boe.es/x", "quote": "Publicado en el BOE núm. 311"}]}
]}]}`

func TestNotesLoadAndRead(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	t1, _ := lawTopicsFor(t, s)
	if _, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, nil); err != nil {
		t.Fatal(err)
	}
	notes, err := ReadNotes(fstest.MapFS{"B1-T01.json": {Data: []byte(testNote)}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.LoadNotes(ctx, notes)
	if err != nil || res.Loaded != 1 || len(res.Problems) > 0 {
		t.Fatalf("load %+v %v", res, err)
	}
	st, err := s.StudyTopic(ctx, t1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	n := st.Note
	if n == nil || len(n.Sections) != 1 || len(n.Sections[0].Points) != 2 {
		t.Fatalf("note %+v", n)
	}
	p := n.Sections[0].Points[0]
	if p.Path != "1.1" || p.Refs[0].Label != "Artículo 62 · Constitución Española" || p.Refs[0].SourceID == 0 ||
		len(p.Refs) != 1 || len(p.Refs[0].Quotes) != 2 || p.Items[0].Path != "1.1.1" {
		t.Errorf("point %+v", p)
	}
	if r := n.Sections[0].Points[1].Refs[0]; r.URL != "https://www.boe.es/x" || r.Label != "Constitución · BOE" {
		t.Errorf("page ref %+v", r)
	}
	blocks, _ := s.Syllabus(ctx)
	if !blocks[0].Topics[0].Notes || blocks[0].Topics[1].Notes {
		t.Errorf("syllabus notes flags")
	}

	// A quote that is not in the law keeps the note out, and says why.
	bad := strings.Replace(testNote, "promulgar las leyes.\"", "vetar las leyes.\"", 1)
	notes, _ = ReadNotes(fstest.MapFS{"B1-T01.json": {Data: []byte(bad)}})
	res, err = s.LoadNotes(ctx, notes)
	if err != nil || res.Loaded != 0 || len(res.Problems) != 1 || !strings.Contains(res.Problems[0], "1.1.1") {
		t.Fatalf("bad quote %+v %v", res, err)
	}
	if st, _ := s.StudyTopic(ctx, t1, time.Now()); st.Note != nil {
		t.Error("a note that fails stays loaded")
	}
}

func TestReadNotesRejects(t *testing.T) {
	for name, body := range map[string]string{
		"no refs":     `{"topic": "B1-T01", "sections": [{"title": "x", "points": [{"text": "Sin fuente.", "refs": []}]}]}`,
		"short quote": `{"topic": "B1-T01", "sections": [{"title": "x", "points": [{"text": "x", "refs": [{"law": "L", "section": "a1", "quote": "corta"}]}]}]}`,
		"http page":   `{"topic": "B1-T01", "sections": [{"title": "x", "points": [{"text": "x", "refs": [{"title": "t", "url": "http://x", "quote": "una cita bastante larga"}]}]}]}`,
		"deep items": `{"topic": "B1-T01", "sections": [{"title": "x", "points": [{"text": "x", "refs": [{"law": "L", "section": "a1", "quote": "una cita bastante larga"}],
			"items": [{"text": "y", "refs": [{"law": "L", "section": "a1", "quote": "una cita bastante larga"}],
			"items": [{"text": "z", "refs": [{"law": "L", "section": "a1", "quote": "una cita bastante larga"}]}]}]}]}]}`,
		"unknown field": `{"topic": "B1-T01", "extra": 1, "sections": []}`,
	} {
		if _, err := ReadNotes(fstest.MapFS{"x.json": {Data: []byte(body)}}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestNoteReports(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	t1, t2 := lawTopicsFor(t, s)
	if _, err := s.LoadLaws(ctx, []LawFile{testLaw("2026-05-20")}, nil); err != nil {
		t.Fatal(err)
	}
	notes, _ := ReadNotes(fstest.MapFS{"B1-T01.json": {Data: []byte(testNote)}})
	if _, err := s.LoadNotes(ctx, notes); err != nil {
		t.Fatal(err)
	}
	const uid = 1 // the admin every database starts with
	if err := s.ReportNote(ctx, uid, t1, "1.1", "Corresponde al Rey", "  "); err == nil {
		t.Error("empty note accepted")
	}
	if err := s.ReportNote(ctx, uid, t2, "1.1", "", "Falta algo"); err != ErrNotFound {
		t.Errorf("topic without note: %v", err)
	}
	if err := s.ReportNote(ctx, uid, t1, "1.1", "Corresponde al Rey", "Falta el refrendo"); err != nil {
		t.Fatal(err)
	}
	open, err := s.OpenNoteReports(ctx)
	if err != nil || len(open) != 1 || open[0].Point != "1.1" || open[0].TopicCode != "B1-T01" || open[0].Note != "Falta el refrendo" {
		t.Fatalf("open %+v %v", open, err)
	}
	if err := s.ResolveNoteReport(ctx, open[0].ID); err != nil {
		t.Fatal(err)
	}
	if open, _ := s.OpenNoteReports(ctx); len(open) != 0 {
		t.Errorf("still open %+v", open)
	}
	if err := s.ResolveNoteReport(ctx, open[0].ID); err != ErrNotFound {
		t.Errorf("resolve twice: %v", err)
	}
}
