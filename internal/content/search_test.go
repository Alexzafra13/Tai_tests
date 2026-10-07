package content

import (
	"context"
	"errors"
	"testing"
)

func TestFTSQuery(t *testing.T) {
	tests := map[string]string{
		"":                      "",
		"  ":                    "",
		"protección datos":      `"protección"* "datos"*`,
		`art. 21 "OR" NEAR(x)*`: `"art"* "21"* "OR"* "NEAR"* "x"*`,
		"Ley 39/2015":           `"Ley"* "39"* "2015"*`,
	}
	for in, want := range tests {
		if got := ftsQuery(in); got != want {
			t.Errorf("ftsQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	pub := f.lawQuestion()
	pubID, _ := f.s.CreateQuestion(ctx, pub)
	draft := f.lawQuestion()
	draft.Stem = "Borrador sobre el plazo de notificación"
	draft.Status = StatusDraft
	f.s.CreateQuestion(ctx, draft)

	// Accent- and case-insensitive, prefix match, by article reference too.
	for _, q := range []string{"NOTIFICAR resolucion", "notif", "21.2", "seis"} {
		hits, err := f.s.Search(ctx, q, 10)
		if err != nil {
			t.Fatalf("Search(%q): %v", q, err)
		}
		if len(hits) != 1 || hits[0].ID != pubID {
			t.Errorf("Search(%q) = %+v, want only the published question", q, hits)
		}
	}
	if hits, _ := f.s.Search(ctx, "inexistente", 10); len(hits) != 0 {
		t.Errorf("unexpected hits: %+v", hits)
	}

	// Edits are indexed (triggers), and the admin list uses the same search.
	pub.Stem = "Plazo máximo de caducidad"
	if err := f.s.UpdateQuestion(ctx, pubID, pub); err != nil {
		t.Fatal(err)
	}
	if hits, _ := f.s.Search(ctx, "caducidad", 10); len(hits) != 1 {
		t.Errorf("edited text not indexed: %+v", hits)
	}
	if page, _ := f.s.ListQuestions(ctx, QuestionFilter{Text: "notificación"}); page.Total != 1 {
		t.Errorf("admin list text search total = %d, want 1 (the draft)", page.Total)
	}
}

func TestPublishedQuestion(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	pub := f.lawQuestion()
	pubID, err := f.s.CreateQuestion(ctx, pub)
	if err != nil {
		t.Fatal(err)
	}
	q, err := f.s.PublishedQuestion(ctx, pubID)
	if err != nil {
		t.Fatal(err)
	}
	if q.Stem != pub.Stem || q.SourceKind != KindLaw || q.SourceTitle != "Ley 39/2015" {
		t.Errorf("PublishedQuestion = %+v", q)
	}
	if len(q.Topics) != len(pub.TopicIDs) || q.Topics[0].ID != pub.TopicIDs[0] {
		t.Errorf("topics = %+v, want %v", q.Topics, pub.TopicIDs)
	}

	draft := f.lawQuestion()
	draft.Status = StatusDraft
	draftID, _ := f.s.CreateQuestion(ctx, draft)
	for _, id := range []int64{draftID, 9999} {
		if _, err := f.s.PublishedQuestion(ctx, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("PublishedQuestion(%d) err = %v, want ErrNotFound", id, err)
		}
	}
}
