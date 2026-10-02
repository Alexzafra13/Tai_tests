package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/users"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	sess, err := s.auth.Login(r.Context(), in.Username, in.Password)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	s.setSessionCookie(w, r, sess.Token, sess.Expires)
	writeJSON(w, http.StatusOK, sess.User)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := s.auth.Logout(r.Context(), c.Value); err != nil {
			s.internalError(w, err)
			return
		}
	}
	s.setSessionCookie(w, r, "", time.Time{})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, auth.CurrentUser(r.Context()))
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !decode(w, r, &in) {
		return
	}
	c, _ := r.Cookie(auth.CookieName)
	err := s.auth.ChangePassword(r.Context(), auth.CurrentUser(r.Context()).ID, c.Value, in.Current, in.New)
	s.respondNoContent(w, err)
}

// setSessionCookie sets the session cookie, or clears it when token is empty.
func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	c := &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	}
	if token == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}

// isHTTPS decides the cookie's Secure flag: a Secure cookie is never sent over
// plain http on the LAN. X-Forwarded-Proto covers reverse proxies (Caddy,
// Nginx); spoofing it only makes the sender's own cookie stricter.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// handleSetupStatus tells the app whether to show the first-run setup
// screen (no administrator can log in yet).
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	need, err := s.users.NeedsSetup(r.Context())
	s.respond(w, map[string]bool{"needed": need}, err)
}

// handleSetup creates the first administrator and logs them in. It only
// works once: afterwards it answers 409.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var in users.SetupInput
	if !decode(w, r, &in) {
		return
	}
	u, err := s.users.Setup(r.Context(), in)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	sess, err := s.auth.OpenSession(r.Context(), u)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.log.Info("first administrator created", "username", u.Username)
	s.setSessionCookie(w, r, sess.Token, sess.Expires)
	writeJSON(w, http.StatusOK, u)
}
