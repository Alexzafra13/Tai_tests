package server

import (
	"net/http"
	"time"

	"github.com/alexzafra13/tai_tests/internal/auth"
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
	s.setSessionCookie(w, sess.Token, sess.Expires)
	writeJSON(w, http.StatusOK, sess.User)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := s.auth.Logout(r.Context(), c.Value); err != nil {
			s.internalError(w, err)
			return
		}
	}
	s.setSessionCookie(w, "", time.Time{})
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
func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	c := &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
	if token == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}
