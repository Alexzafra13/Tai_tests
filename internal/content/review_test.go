package content

import (
	"context"
	"testing"
)

func TestReviewQueue(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	published := f.lawQuestion()
	pubID, _ := f.s.CreateQuestion(ctx, published)

	draft := f.lawQuestion()
	draft.Status, draft.TopicIDs = StatusDraft, nil
	draftID, err := f.s.CreateQuestion(ctx, draft)
	if err != nil {
		t.Fatal(err)
	}

	flagged := f.lawQuestion()
	flagged.Flagged, flagged.FlagNote = true, "¿seguro que son seis?"
	flaggedID, _ := f.s.CreateQuestion(ctx, flagged)

	counts, err := f.s.ReviewCounts(ctx)
	if err != nil || counts != (ReviewCounts{Flagged: 1, Drafts: 1, Total: 2}) {
		t.Fatalf("counts = %+v, %v", counts, err)
	}

	page, err := f.s.ReviewQueue(ctx, ReviewAll, 0, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Items[0].ID != flaggedID || page.Items[1].ID != draftID {
		t.Fatalf("queue order: %+v", ids(page))
	}
	ex := page.Items[0].Excerpt
	if ex == nil || ex.Match != "Este plazo no podrá exceder de seis meses" || ex.Before == "" {
		t.Errorf("excerpt = %+v", ex)
	}
	for _, it := range page.Items {
		if it.ID == pubID {
			t.Error("published, unflagged question in the queue")
		}
	}

	// A draft without topics cannot be accepted until it gets one.
	if _, err := f.s.Accept(ctx, draftID, nil); err == nil {
		t.Fatal("accepted a question without topics")
	}
	prev, err := f.s.Accept(ctx, draftID, []int64{f.topicIDs[1]})
	if err != nil {
		t.Fatal(err)
	}
	if prev.Status != StatusDraft {
		t.Errorf("previous state = %+v", prev)
	}
	q, _ := f.s.Question(ctx, draftID)
	if q.Status != StatusPublished || len(q.TopicIDs) != 1 || q.TopicIDs[0] != f.topicIDs[1] {
		t.Errorf("accepted question = %+v", q)
	}

	// Accepting a flagged question clears the flag; undo brings it back.
	prev, _ = f.s.Accept(ctx, flaggedID, nil)
	if q, _ := f.s.Question(ctx, flaggedID); q.Flagged {
		t.Error("flag not cleared on accept")
	}
	if err := f.s.RestoreReview(ctx, flaggedID, prev); err != nil {
		t.Fatal(err)
	}
	if q, _ := f.s.Question(ctx, flaggedID); !q.Flagged || q.FlagNote != "¿seguro que son seis?" {
		t.Errorf("undo lost the flag: %+v", q)
	}

	if _, err := f.s.Discard(ctx, flaggedID); err != nil {
		t.Fatal(err)
	}
	if counts, _ := f.s.ReviewCounts(ctx); counts.Total != 0 {
		t.Errorf("queue not empty after decisions: %+v", counts)
	}
}

func TestAcceptBatchReportsFailures(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	good := f.lawQuestion()
	good.Status = StatusDraft
	goodID, _ := f.s.CreateQuestion(ctx, good)
	bad := good
	bad.TopicIDs = nil
	badID, _ := f.s.CreateQuestion(ctx, bad)

	res, err := f.s.AcceptBatch(ctx, []int64{goodID, badID, 999})
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].OK || res[1].OK || res[1].Errors["topic_ids"] == "" || res[2].OK {
		t.Fatalf("batch results = %+v", res)
	}
}

func ids(p ReviewPage) []int64 {
	var out []int64
	for _, it := range p.Items {
		out = append(out, it.ID)
	}
	return out
}
