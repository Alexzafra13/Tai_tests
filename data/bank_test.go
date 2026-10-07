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
