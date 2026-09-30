// Package server wires the HTTP API and the embedded single-page app.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/quiz"
)

type Deps struct {
	Auth         *auth.Service
	Content      *content.Store
	Quiz         *quiz.Store
	CookieSecure bool
	Static       fs.FS
	Log          *slog.Logger
}

type Server struct {
	auth         *auth.Service
	content      *content.Store
	quiz         *quiz.Store
	cookieSecure bool
	static       fs.FS
	log          *slog.Logger
}

func New(d Deps) *Server {
	return &Server{auth: d.Auth, content: d.Content, quiz: d.Quiz, cookieSecure: d.CookieSecure, static: d.Static, log: d.Log}
}

func (s *Server) Handler() http.Handler {
	// Everything under /api/ requires a session except these routes.
	public := http.NewServeMux()
	public.HandleFunc("GET /api/health", s.handleHealth)
	public.HandleFunc("POST /api/auth/login", s.handleLogin)
	public.HandleFunc("POST /api/auth/logout", s.handleLogout)

	private := http.NewServeMux()
	private.HandleFunc("GET /api/auth/me", s.handleMe)
	s.contentRoutes(private)
	s.quizRoutes(private)
	s.reviewRoutes(private)
	private.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	public.Handle("/api/", s.requireAuth(private))

	root := http.NewServeMux()
	root.Handle("/api/", public)
	root.Handle("/", spaHandler(s.static))
	return securityHeaders(root)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, expires, err := s.auth.Login(r.Context(), req.Password)
	switch {
	case errors.Is(err, auth.ErrBadPassword):
		writeError(w, http.StatusUnauthorized, "Contraseña incorrecta")
		return
	case errors.Is(err, auth.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "Demasiados intentos fallidos. Espera unos minutos.")
		return
	case err != nil:
		s.internalError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := s.auth.Logout(r.Context(), c.Value); err != nil {
			s.internalError(w, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if c, err := r.Cookie(auth.CookieName); err == nil {
			token = c.Value
		}
		ok, err := s.auth.Valid(r.Context(), token)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.log.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Run serves h on addr until ctx is cancelled, then shuts down gracefully.
func Run(ctx context.Context, addr string, h http.Handler, log *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Info("shutting down")
		return srv.Shutdown(shutdownCtx)
	}
}
