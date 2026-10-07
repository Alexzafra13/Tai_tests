package data_test

import (
	"context"
	"testing"

	"github.com/alexzafra13/tai_tests/data"
	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
)

// Every bundled question must pass the same validation as the rest.
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
	res, err := content.NewStore(d).LoadBank(ctx, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Problems {
		t.Errorf("%s %s: %s", p.Source, p.Key, p.Reason)
	}
	total := 0
	for _, f := range files {
		total += len(f.Questions)
	}
	if res.Added != total {
		t.Errorf("added %d of %d questions", res.Added, total)
	}
}
