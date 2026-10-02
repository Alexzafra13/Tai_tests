package server

import (
	"net/http"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/quiz"
)

func userID(r *http.Request) int64 { return auth.CurrentUser(r.Context()).ID }

func (s *Server) handleListTests(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, err := s.quiz.List(r.Context(), userID(r), quiz.Status(q.Get("status")), int(queryInt(q.Get("limit"))))
	s.respond(w, list, err)
}

func (s *Server) handleAvailable(w http.ResponseWriter, r *http.Request) {
	var f quiz.Filters
	if !decode(w, r, &f) {
		return
	}
	n, err := s.quiz.Available(r.Context(), userID(r), f)
	s.respond(w, map[string]int{"available": n}, err)
}

func (s *Server) handleCreateTest(w http.ResponseWriter, r *http.Request) {
	var in quiz.CreateInput
	if !decode(w, r, &in) {
		return
	}
	id, err := s.quiz.Create(r.Context(), userID(r), in)
	s.respondCreated(w, id, err)
}

func (s *Server) handleGetTest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := s.quiz.Get(r.Context(), userID(r), id)
	s.respond(w, t, err)
}

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in quiz.AnswerInput
	if !decode(w, r, &in) {
		return
	}
	sol, err := s.quiz.Answer(r.Context(), userID(r), id, in)
	if err == nil && sol == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.respond(w, sol, err)
}

func (s *Server) handleFinish(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	res, err := s.quiz.Finish(r.Context(), userID(r), id)
	s.respond(w, res, err)
}

func (s *Server) handleAbandon(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.respondNoContent(w, s.quiz.Abandon(r.Context(), userID(r), id))
}

// handleReport opens or withdraws the caller's doubt about a question of
// their test; open doubts appear in the administrators' review queue.
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Position int    `json:"position"`
		Reported bool   `json:"reported"`
		Note     string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	qid, err := s.quiz.QuestionAt(r.Context(), userID(r), id, in.Position)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	s.respondNoContent(w, s.content.SetReport(r.Context(), qid, userID(r), in.Reported, in.Note))
}

func (s *Server) handleGetScoring(w http.ResponseWriter, r *http.Request) {
	sc, err := s.quiz.Scoring(r.Context())
	s.respond(w, sc, err)
}

func (s *Server) handleSetScoring(w http.ResponseWriter, r *http.Request) {
	var sc quiz.ScoringSettings
	if !decode(w, r, &sc) {
		return
	}
	s.respondNoContent(w, s.quiz.SetScoring(r.Context(), sc))
}
