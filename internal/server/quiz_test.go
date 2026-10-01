package server

import (
	"net/http"
	"testing"
)

func TestScoringSettingsAPI(t *testing.T) {
	ts, c := newTestServer(t)
	login(t, c, ts.URL)

	var sc struct {
		Max      float64 `json:"max"`
		PassMark float64 `json:"pass_mark"`
	}
	if status := doJSON(t, c, "GET", ts.URL+"/api/settings/scoring", nil, &sc); status != http.StatusOK || sc.Max != 100 {
		t.Fatalf("default scoring: %d %+v", status, sc)
	}
	var verr struct{ Fields map[string]string }
	if status := doJSON(t, c, "PUT", ts.URL+"/api/settings/scoring", map[string]any{"max": 10, "pass_mark": 50, "default_penalty": 0.25}, &verr); status != http.StatusUnprocessableEntity || verr.Fields["pass_mark"] == "" {
		t.Fatalf("invalid scoring: %d %+v", status, verr)
	}
	if status := doJSON(t, c, "PUT", ts.URL+"/api/settings/scoring", map[string]any{"max": 50, "pass_mark": 25, "default_penalty": 0.25}, nil); status != http.StatusNoContent {
		t.Fatalf("set scoring: %d", status)
	}
	doJSON(t, c, "GET", ts.URL+"/api/settings/scoring", nil, &sc)
	if sc.Max != 50 || sc.PassMark != 25 {
		t.Fatalf("scoring after update: %+v", sc)
	}
}
