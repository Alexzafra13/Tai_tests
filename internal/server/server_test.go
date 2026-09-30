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
)

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
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
	s := New(Deps{
		Auth:    auth.NewService(d, "secret-password", time.Hour),
		Content: content.NewStore(d),
		Static:  static,
		Log:     slog.New(slog.DiscardHandler),
	})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	jar, _ := cookiejar.New(nil)
	return ts, &http.Client{Jar: jar}
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
	if resp := post(t, c, ts.URL+"/api/auth/login", `{"password":"nope"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login: %d, want 401", resp.StatusCode)
	}

	resp := post(t, c, ts.URL+"/api/auth/login", `{"password":"secret-password"}`)
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

func login(t *testing.T, c *http.Client, base string) {
	t.Helper()
	if resp := post(t, c, base+"/api/auth/login", `{"password":"secret-password"}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d", resp.StatusCode)
	}
}
