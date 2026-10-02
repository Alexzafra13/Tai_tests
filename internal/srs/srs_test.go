package srs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
)

func TestRatingFor(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name     string
		correct  *bool
		spent    time.Duration
		doubtful bool
		want     Rating
	}{
		{"blank", nil, 0, false, Again},
		{"wrong", &no, 20 * time.Second, false, Again},
		{"wrong even if quick", &no, time.Second, false, Again},
		{"right but doubtful", &yes, 2 * time.Second, true, Hard},
		{"right and quick", &yes, 3 * time.Second, false, Easy},
		{"right", &yes, 30 * time.Second, false, Good},
		{"right with no timing", &yes, 0, false, Good},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RatingFor(tt.correct, tt.spent, tt.doubtful); got != tt.want {
				t.Errorf("RatingFor = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecordSchedulesAndCountsDue(t *testing.T) {
	ctx := context.Background()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	cs := content.NewStore(d)
	syl, _ := content.ParseSyllabus(strings.NewReader(`{"blocks":[{"code":"B1","name":"B","topics":[{"code":"T1","title":"t"}]}]}`))
	cs.LoadSyllabus(ctx, syl)
	blocks, _ := cs.Syllabus(ctx)
	src, _ := cs.CreateSource(ctx, content.SourceInput{Kind: content.KindINAPExam, Title: "e"})
	var ids []int64
	for i := range 2 {
		id, err := cs.CreateQuestion(ctx, content.QuestionInput{Stem: string(rune('a' + i)), Options: [4]string{"1", "2", "3", "4"},
			Origin: content.OriginOfficial, Author: content.AuthorImport, SourceID: src, SourceRef: "r",
			Status: content.StatusPublished, TopicIDs: []int64{blocks[0].Topics[0].ID}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}

	s := NewStore(d)
	const user = 1
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	if err := s.Record(ctx, user, ids[0], Again, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(ctx, user, ids[1], Easy, now); err != nil {
		t.Fatal(err)
	}

	// A failed question comes back the next day; an easy one days later.
	if sm, _ := s.Summary(ctx, user, now.Add(time.Hour)); sm.Tracked != 2 || sm.Due != 0 {
		t.Fatalf("an hour later: %+v, want 2 tracked, none due", sm)
	}
	sm, _ := s.Summary(ctx, user, now.Add(26*time.Hour))
	if sm.Due != 1 {
		t.Fatalf("next day: %+v, want 1 due (the failed one)", sm)
	}
	sm, _ = s.Summary(ctx, user, now.Add(60*24*time.Hour))
	if sm.Due != 2 {
		t.Fatalf("two months later: %+v, want both due", sm)
	}

	// Each answer reschedules: remembering pushes the card further away.
	var dueBefore, dueAfter string
	d.QueryRow(`SELECT due FROM review_cards WHERE question_id = ?`, ids[1]).Scan(&dueBefore)
	s.Record(ctx, user, ids[1], Good, now.Add(10*24*time.Hour))
	d.QueryRow(`SELECT due FROM review_cards WHERE question_id = ?`, ids[1]).Scan(&dueAfter)
	if dueAfter <= dueBefore {
		t.Errorf("due did not move forward: %s -> %s", dueBefore, dueAfter)
	}
}
