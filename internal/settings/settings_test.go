package settings

import (
	"context"
	"testing"

	"github.com/alexzafra13/tai_tests/internal/db"
)

func TestGetSet(t *testing.T) {
	ctx := context.Background()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	s := NewStore(d)

	type conf struct{ Max float64 }
	v := conf{Max: 10}
	if found, err := s.Get(ctx, "k", &v); err != nil || found || v.Max != 10 {
		t.Fatalf("missing key: found=%v err=%v v=%+v", found, err, v)
	}
	if err := s.Set(ctx, "k", conf{Max: 100}); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(ctx, "k", conf{Max: 50}); err != nil {
		t.Fatal(err)
	}
	if found, err := s.Get(ctx, "k", &v); err != nil || !found || v.Max != 50 {
		t.Fatalf("after set: found=%v err=%v v=%+v", found, err, v)
	}
}
