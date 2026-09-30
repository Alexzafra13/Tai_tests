package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/quiz"
	"github.com/alexzafra13/tai_tests/internal/settings"
	"github.com/alexzafra13/tai_tests/internal/users"
)

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	env := newTestEnv(t)
	return env.ts, newClient()
}

// testEnv is a running server plus direct access to its stores, to seed
// data that has no endpoint (like the syllabus).
type testEnv struct {
	ts      *httptest.Server
	content *content.Store
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>TAI</title>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
		"manifest.json": {Data: []byte("{}")},
	}
	us := users.NewStore(d)
	if _, err := us.EnsureAdmin(context.Background(), "admin", "secret-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := us.Create(context.Background(), users.CreateInput{Username: "ana", Password: "ana-password", Role: users.RoleUser}); err != nil {
		t.Fatal(err)
	}
	s := New(Deps{
		Auth:    auth.NewService(d, us, time.Hour),
		Users:   us,
		Content: content.NewStore(d),
		Quiz:    quiz.NewStore(d, settings.NewStore(d)),
		Static:  static,
		Log:     slog.New(slog.DiscardHandler),
	})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return testEnv{ts: ts, content: content.NewStore(d)}
}

// newClient returns an HTTP client with its own cookie jar, i.e. its own
// session.
func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func post(t *testing.T, c *http.Client, url, body string) *http.Response {
	t.Helper()
	resp, err := c.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func get(t *testing.T, c *http.Client, url string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestAuthFlow(t *testing.T) {
	ts, c := newTestServer(t)

	if resp, _ := get(t, c, ts.URL+"/api/auth/me"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me before login: %d, want 401", resp.StatusCode)
	}
	if resp := post(t, c, ts.URL+"/api/auth/login", `{"username":"admin","password":"nope"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login: %d, want 401", resp.StatusCode)
	}

	resp := post(t, c, ts.URL+"/api/auth/login", `{"username":"admin","password":"secret-password"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d, want 200", resp.StatusCode)
	}
	var cookie *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.CookieName {
			cookie = ck
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie missing or insecure: %+v", cookie)
	}

	if resp, _ := get(t, c, ts.URL+"/api/auth/me"); resp.StatusCode != http.StatusOK {
		t.Fatalf("me after login: %d, want 200", resp.StatusCode)
	}

	if resp := post(t, c, ts.URL+"/api/auth/logout", ``); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %d, want 204", resp.StatusCode)
	}
	if resp, _ := get(t, c, ts.URL+"/api/auth/me"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me after logout: %d, want 401", resp.StatusCode)
	}
}

func TestSPARouting(t *testing.T) {
	ts, c := newTestServer(t)

	resp, body := get(t, c, ts.URL+"/tests/123")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "<title>TAI</title>") {
		t.Fatalf("client route: %d %q, want index.html", resp.StatusCode, body)
	}

	resp, body = get(t, c, ts.URL+"/assets/app.js")
	if resp.StatusCode != http.StatusOK || body != "console.log(1)" {
		t.Fatalf("asset: %d %q", resp.StatusCode, body)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("asset Cache-Control = %q, want immutable", cc)
	}

	// Unknown API routes don't reveal anything before login.
	if resp, _ := get(t, c, ts.URL+"/api/unknown"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown api route before login: %d, want 401", resp.StatusCode)
	}
	login(t, c, ts.URL)
	if resp, _ := get(t, c, ts.URL+"/api/unknown"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown api route: %d, want 404", resp.StatusCode)
	}
}

// login logs c in as the administrator.
func login(t *testing.T, c *http.Client, base string) {
	t.Helper()
	loginAs(t, c, base, "admin", "secret-password")
}

func loginAs(t *testing.T, c *http.Client, base, username, password string) {
	t.Helper()
	body := `{"username":"` + username + `","password":"` + password + `"}`
	if resp := post(t, c, base+"/api/auth/login", body); resp.StatusCode != http.StatusOK {
		t.Fatalf("login %s: %d", username, resp.StatusCode)
	}
}
