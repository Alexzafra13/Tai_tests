package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// Students can study and report doubts, but every content, review, scoring
// and account-management endpoint is refused on the server.
func TestUserCannotReachAdminEndpoints(t *testing.T) {
	ts, _ := newTestServer(t)
	ana := newClient()
	loginAs(t, ana, ts.URL, "ana", "ana-password")

	var me struct{ Username, Role string }
	if status := doJSON(t, ana, "GET", ts.URL+"/api/auth/me", nil, &me); status != http.StatusOK || me.Role != "user" {
		t.Fatalf("me: %d %+v", status, me)
	}

	adminOnly := []struct{ method, path string }{
		{"GET", "/api/questions"},
		{"POST", "/api/questions"},
		{"GET", "/api/questions/1"},
		{"POST", "/api/sources"},
		{"GET", "/api/sources/1"},
		{"GET", "/api/review"},
		{"GET", "/api/review/counts"},
		{"POST", "/api/review/1/accept"},
		{"PUT", "/api/settings/scoring"},
		{"GET", "/api/users"},
		{"POST", "/api/users"},
		{"PUT", "/api/users/1/password"},
	}
	for _, e := range adminOnly {
		if status := doJSON(t, ana, e.method, ts.URL+e.path, map[string]any{}, nil); status != http.StatusForbidden {
			t.Errorf("%s %s as user: %d, want 403", e.method, e.path, status)
		}
	}

	for _, path := range []string{"/api/syllabus", "/api/sources", "/api/exams", "/api/tests", "/api/settings/scoring"} {
		if status := doJSON(t, ana, "GET", ts.URL+path, nil, nil); status != http.StatusOK {
			t.Errorf("GET %s as user: %d, want 200", path, status)
		}
	}
}

func TestAdminManagesUsers(t *testing.T) {
	ts, admin := newTestServer(t)
	login(t, admin, ts.URL)

	var created struct{ ID int64 }
	status := doJSON(t, admin, "POST", ts.URL+"/api/users", map[string]any{
		"username": "luis", "display_name": "Luis", "password": "luis-password", "role": "user",
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create user: %d", status)
	}
	luis := newClient()
	loginAs(t, luis, ts.URL, "luis", "luis-password")

	// Resetting the password closes Luis's open session.
	if status := doJSON(t, admin, "PUT", ts.URL+"/api/users/"+strconv.FormatInt(created.ID, 10)+"/password", map[string]any{"password": "otra-password"}, nil); status != http.StatusNoContent {
		t.Fatalf("reset password: %d", status)
	}
	if status := doJSON(t, luis, "GET", ts.URL+"/api/auth/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("session after reset: %d, want 401", status)
	}

	// The last administrator cannot demote itself.
	if status := doJSON(t, admin, "PUT", ts.URL+"/api/users/1", map[string]any{"display_name": "", "role": "user", "active": true}, nil); status != http.StatusConflict {
		t.Errorf("demote last admin: %d, want 409", status)
	}
}

func TestChangeOwnPassword(t *testing.T) {
	ts, _ := newTestServer(t)
	ana := newClient()
	loginAs(t, ana, ts.URL, "ana", "ana-password")

	var verr struct{ Fields map[string]string }
	if status := doJSON(t, ana, "PUT", ts.URL+"/api/account/password", map[string]any{"current_password": "wrong", "new_password": "nueva-password"}, &verr); status != http.StatusUnprocessableEntity || verr.Fields["current_password"] == "" {
		t.Fatalf("wrong current password: %d %+v", status, verr)
	}
	if status := doJSON(t, ana, "PUT", ts.URL+"/api/account/password", map[string]any{"current_password": "ana-password", "new_password": "nueva-password"}, nil); status != http.StatusNoContent {
		t.Fatalf("change password: %d", status)
	}
	loginAs(t, newClient(), ts.URL, "ana", "nueva-password")
}

// Each user sees only their own tests, and doubts reach the admin's queue
// with who raised them.
func TestTestsAndReportsPerUser(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	base := env.ts.URL
	syl, _ := content.ParseSyllabus(strings.NewReader(`{"blocks":[{"code":"B1","name":"B","topics":[{"code":"T1","title":"t"}]}]}`))
	if _, err := env.content.LoadSyllabus(ctx, syl); err != nil {
		t.Fatal(err)
	}
	blocks, _ := env.content.Syllabus(ctx)
	src, _ := env.content.CreateSource(ctx, content.SourceInput{Kind: content.KindINAPExam, Title: "TAI 2024"})
	if _, err := env.content.CreateQuestion(ctx, content.QuestionInput{Stem: "¿?", Options: [4]string{"a", "b", "c", "d"},
		Origin: content.OriginOfficial, Author: content.AuthorImport, SourceID: src, SourceRef: "2024 · 1",
		Status: content.StatusPublished, TopicIDs: []int64{blocks[0].Topics[0].ID}}); err != nil {
		t.Fatal(err)
	}

	ana, admin := newClient(), newClient()
	loginAs(t, ana, base, "ana", "ana-password")
	login(t, admin, base)

	var test struct{ ID int64 }
	newTest := map[string]any{"mode": "practice", "count": 1, "penalty": 0, "time_limit_min": 0, "filters": map[string]any{}}
	if status := doJSON(t, ana, "POST", base+"/api/tests", newTest, &test); status != http.StatusCreated {
		t.Fatalf("create test: %d", status)
	}
	testURL := base + "/api/tests/" + strconv.FormatInt(test.ID, 10)

	if status := doJSON(t, admin, "GET", testURL, nil, nil); status != http.StatusNotFound {
		t.Errorf("another user reads the test: %d, want 404", status)
	}
	var list []any
	doJSON(t, admin, "GET", base+"/api/tests", nil, &list)
	if len(list) != 0 {
		t.Errorf("another user lists %d tests", len(list))
	}

	if status := doJSON(t, ana, "POST", testURL+"/report", map[string]any{"position": 0, "reported": true, "note": "dudo"}, nil); status != http.StatusNoContent {
		t.Fatalf("report: %d", status)
	}
	var queue struct {
		Items []struct {
			Reports []struct{ Username, Note string }
		}
	}
	doJSON(t, admin, "GET", base+"/api/review?kind=reported", nil, &queue)
	if len(queue.Items) != 1 || len(queue.Items[0].Reports) != 1 || queue.Items[0].Reports[0].Username != "ana" {
		t.Fatalf("review queue = %+v", queue)
	}
	if status := doJSON(t, admin, "POST", testURL+"/report", map[string]any{"position": 0, "reported": true}, nil); status != http.StatusNotFound {
		t.Errorf("reporting on another user's test: %d, want 404", status)
	}
}
