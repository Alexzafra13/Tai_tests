package server

import (
	"net/http"
	"testing"
)

func TestReviewAPI(t *testing.T) {
	ts, c := newTestServer(t)
	login(t, c, ts.URL)

	var src struct{ ID int64 }
	doJSON(t, c, "POST", ts.URL+"/api/sources", map[string]any{"kind": "inap_exam", "title": "TAI 2024"}, &src)
	var q struct{ ID int64 }
	status := doJSON(t, c, "POST", ts.URL+"/api/questions", map[string]any{
		"stem": "¿Pregunta?", "options": []string{"a", "b", "c", "d"}, "correct": 0,
		"origin": "official", "author": "import", "source_id": src.ID, "source_ref": "2024 · 1", "status": "draft",
	}, &q)
	if status != http.StatusCreated {
		t.Fatalf("create draft: %d", status)
	}

	var counts struct{ Total, Drafts int }
	doJSON(t, c, "GET", ts.URL+"/api/review/counts", nil, &counts)
	if counts.Total != 1 || counts.Drafts != 1 {
		t.Fatalf("counts = %+v", counts)
	}

	// No topics yet: accepting fails with a field error.
	var verr struct{ Fields map[string]string }
	if status := doJSON(t, c, "POST", ts.URL+"/api/review/1/accept", map[string]any{}, &verr); status != http.StatusUnprocessableEntity || verr.Fields["topic_ids"] == "" {
		t.Fatalf("accept without topics: %d %+v", status, verr)
	}

	var dec struct {
		Previous struct{ Status string }
	}
	if status := doJSON(t, c, "POST", ts.URL+"/api/review/1/discard", nil, &dec); status != http.StatusOK || dec.Previous.Status != "draft" {
		t.Fatalf("discard: %d %+v", status, dec)
	}
	if status := doJSON(t, c, "POST", ts.URL+"/api/review/1/restore", map[string]any{"status": "draft", "flagged": false, "flag_note": ""}, nil); status != http.StatusNoContent {
		t.Fatalf("restore: %d", status)
	}
	doJSON(t, c, "GET", ts.URL+"/api/review/counts", nil, &counts)
	if counts.Total != 1 {
		t.Fatalf("after undo counts = %+v", counts)
	}
}
