package server

import (
	"net/http"
	"strings"
	"testing"
)

// The session cookie is Secure only when the browser came over HTTPS, so
// the same install works on plain http in the LAN and behind a proxy.
func TestSessionCookieFollowsHTTPS(t *testing.T) {
	ts, _ := newTestServer(t)
	for _, tc := range []struct {
		proto  string
		secure bool
	}{{"", false}, {"https", true}, {"http", false}} {
		req, _ := http.NewRequest("POST", ts.URL+"/api/auth/login",
			strings.NewReader(`{"username":"admin","password":"secret-password"}`))
		req.Header.Set("Content-Type", "application/json")
		if tc.proto != "" {
			req.Header.Set("X-Forwarded-Proto", tc.proto)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		cookies := resp.Cookies()
		if len(cookies) != 1 || cookies[0].Secure != tc.secure {
			t.Errorf("proto %q: cookies %+v, want Secure=%v", tc.proto, cookies, tc.secure)
		}
	}
}

func TestClientZone(t *testing.T) {
	for query, want := range map[string]string{
		"tz=Atlantic/Canary": "Atlantic/Canary",
		"tz=Not/AZone":       "Europe/Madrid",
		"":                   "Europe/Madrid",
	} {
		r, _ := http.NewRequest("GET", "/api/stats?"+query, nil)
		if got := clientZone(r).String(); got != want {
			t.Errorf("%q: zone %s, want %s", query, got, want)
		}
	}
}
