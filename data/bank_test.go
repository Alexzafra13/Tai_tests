package data_test

import (
	"context"
	"testing"

	"github.com/alexzafra13/tai_tests/data"
	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
)

// Every bundled question must pass the same validation as the rest, and
// those marked as checked must arrive published with the bundled syllabus.
func TestBankLoadsCleanly(t *testing.T) {
	ctx := context.Background()
	files, err := content.ReadBank(data.Bank())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("empty bank")
	}
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	cs := content.NewStore(d)
	syl, err := content.ParseSyllabus(data.Syllabus())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.LoadSyllabus(ctx, syl); err != nil {
		t.Fatal(err)
	}
	res, err := cs.LoadBank(ctx, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Problems {
		t.Errorf("%s %s: %s", p.Source, p.Key, p.Reason)
	}
	total, checked := 0, 0
	for _, f := range files {
		total += len(f.Questions)
		for _, q := range f.Questions {
			if q.Status == content.StatusPublished {
				checked++
			}
		}
	}
	if res.Added != total || res.Published != checked {
		t.Errorf("added %d of %d questions, published %d of %d", res.Added, total, res.Published, checked)
	}
}

// The bundled laws load, every topic they name exists in the bundled
// syllabus, and each topic gets its texts.
func TestLawsLoadCleanly(t *testing.T) {
	ctx := context.Background()
	laws, topics, err := content.ReadLaws(data.Laws())
	if err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	cs := content.NewStore(d)
	syl, err := content.ParseSyllabus(data.Syllabus())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.LoadSyllabus(ctx, syl); err != nil {
		t.Fatal(err)
	}
	res, err := cs.LoadLaws(ctx, laws, topics)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != len(laws) || len(res.Problems) > 0 {
		t.Fatalf("result %+v", res)
	}
	blocks, err := cs.Syllabus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	linked := 0
	for _, b := range blocks {
		for _, tp := range b.Topics {
			if _, ok := topics[tp.Code]; ok {
				if tp.Laws == 0 {
					t.Errorf("%s has no laws", tp.Code)
				}
				linked++
			}
		}
	}
	if linked != len(topics) {
		t.Errorf("topics.json names %d topics, %d exist", len(topics), linked)
	}
}

// The articles bank questions point to exist in the bundled laws.
func TestBankArticlesExist(t *testing.T) {
	files, err := content.ReadBank(data.Bank())
	if err != nil {
		t.Fatal(err)
	}
	laws, _, err := content.ReadLaws(data.Laws())
	if err != nil {
		t.Fatal(err)
	}
	sections := map[string]map[string]content.LawSectionKind{}
	for _, l := range laws {
		sections[l.Reference] = map[string]content.LawSectionKind{}
		for _, s := range l.Sections {
			sections[l.Reference][s.ID] = s.Kind
		}
	}
	for _, f := range files {
		for _, q := range f.Questions {
			for _, a := range q.Articles {
				secs, ok := sections[a.Law]
				switch {
				case !ok:
					t.Errorf("%s %s: unknown law %s", f.Source.Reference, q.Key, a.Law)
				case a.Section != "" && secs[a.Section] != content.SectionArticle:
					t.Errorf("%s %s: %s has no article %q", f.Source.Reference, q.Key, a.Law, a.Section)
				}
			}
		}
	}
}
