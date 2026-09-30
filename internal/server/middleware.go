package server

import (
	"net/http"

	"github.com/alexzafra13/tai_tests/internal/auth"
)

// requireUser lets the request through only with a valid session, and puts
// the user in the request context (see auth.CurrentUser).
func (s *Server) requireUser(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if c, err := r.Cookie(auth.CookieName); err == nil {
			token = c.Value
		}
		u, ok, err := s.auth.UserForToken(r.Context(), token)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), u)))
	})
}

// requireAdmin is requireUser plus the administrator role.
func (s *Server) requireAdmin(next http.HandlerFunc) http.Handler {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request) {
		if !auth.CurrentUser(r.Context()).IsAdmin() {
			writeError(w, http.StatusForbidden, "Solo un administrador puede hacer esto")
			return
		}
		next(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
