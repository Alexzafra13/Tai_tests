package server

import (
	"net/http"
	"testing"
)

func TestStudyEndpoints(t *testing.T) {
	ts, c := newTestServer(t)
	login(t, c, ts.URL)

	var sum struct {
		Review struct{ Due, Tracked int }
		Failed int
	}
	if status := doJSON(t, c, "GET", ts.URL+"/api/study/summary", nil, &sum); status != http.StatusOK || sum.Review.Tracked != 0 {
		t.Fatalf("summary: %d %+v", status, sum)
	}
	var st struct {
		Overview struct{ Answered int }
		Timeline []struct{ Date string }
		Topics   []any
	}
	if status := doJSON(t, c, "GET", ts.URL+"/api/stats", nil, &st); status != http.StatusOK || len(st.Timeline) != 30 || st.Topics == nil {
		t.Fatalf("stats: %d %+v", status, st)
	}
	var hits []any
	if status := doJSON(t, c, "GET", ts.URL+"/api/search?q=ley", nil, &hits); status != http.StatusOK || hits == nil {
		t.Fatalf("search: %d %v", status, hits)
	}
	if resp, _ := get(t, newClient(), ts.URL+"/api/stats"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("stats without login: %d", resp.StatusCode)
	}
}
