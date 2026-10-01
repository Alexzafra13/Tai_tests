// Package server wires the HTTP API and the embedded single-page app.
//
// routes.go is the single table of endpoints and who may call them; the
// handlers live in one file per area (account, users, content, quiz,
// review) and share the helpers in respond.go.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/quiz"
	"github.com/alexzafra13/tai_tests/internal/users"
)

type Deps struct {
	Auth         *auth.Service
	Users        *users.Store
	Content      *content.Store
	Quiz         *quiz.Store
	CookieSecure bool
	Static       fs.FS
	Log          *slog.Logger
}

type Server struct {
	auth         *auth.Service
	users        *users.Store
	content      *content.Store
	quiz         *quiz.Store
	cookieSecure bool
	static       fs.FS
	log          *slog.Logger
}

func New(d Deps) *Server {
	return &Server{
		auth:         d.Auth,
		users:        d.Users,
		content:      d.Content,
		quiz:         d.Quiz,
		cookieSecure: d.CookieSecure,
		static:       d.Static,
		log:          d.Log,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	mux.Handle("/", spaHandler(s.static))
	return securityHeaders(mux)
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
