package server

import (
	"net/http"

	"github.com/alexzafra13/tai_tests/internal/users"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	list, err := s.users.List(r.Context())
	s.respond(w, list, err)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in users.CreateInput
	if !decode(w, r, &in) {
		return
	}
	id, err := s.users.Create(r.Context(), in)
	s.respondCreated(w, id, err)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in users.UpdateInput
	if !decode(w, r, &in) {
		return
	}
	s.respondNoContent(w, s.users.Update(r.Context(), id, in))
}

// handleResetPassword sets a user's password (e.g. when they forget it) and
// closes their sessions.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.users.SetPassword(r.Context(), id, in.Password); err != nil {
		s.writeErr(w, err)
		return
	}
	s.respondNoContent(w, s.auth.EndSessions(r.Context(), id))
}
