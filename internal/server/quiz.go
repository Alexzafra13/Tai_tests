package server

import (
	"errors"
	"net/http"

	"github.com/alexzafra13/tai_tests/internal/quiz"
)

func (s *Server) quizRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tests", s.handleListTests)
	mux.HandleFunc("POST /api/tests", s.handleCreateTest)
	mux.HandleFunc("POST /api/tests/available", s.handleAvailable)
	mux.HandleFunc("GET /api/tests/{id}", s.handleGetTest)
	mux.HandleFunc("POST /api/tests/{id}/answer", s.handleAnswer)
	mux.HandleFunc("POST /api/tests/{id}/finish", s.handleFinish)
	mux.HandleFunc("POST /api/tests/{id}/abandon", s.handleAbandon)
	mux.HandleFunc("POST /api/tests/{id}/flag", s.handleFlag)

	mux.HandleFunc("GET /api/settings/scoring", s.handleGetScoring)
	mux.HandleFunc("PUT /api/settings/scoring", s.handleSetScoring)
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

func (s *Server) handleListTests(w http.ResponseWriter, r *http.Request) {
	list, err := s.quiz.List(r.Context(), quiz.Status(r.URL.Query().Get("status")), int(queryInt(r.URL.Query().Get("limit"))))
	s.respond(w, list, err)
}

func (s *Server) handleAvailable(w http.ResponseWriter, r *http.Request) {
	var f quiz.Filters
	if !decode(w, r, &f) {
		return
	}
	n, err := s.quiz.Available(r.Context(), f)
	s.respond(w, map[string]int{"available": n}, err)
}

func (s *Server) handleCreateTest(w http.ResponseWriter, r *http.Request) {
	var in quiz.CreateInput
	if !decode(w, r, &in) {
		return
	}
	id, err := s.quiz.Create(r.Context(), in)
	s.respondCreated(w, id, err)
}

func (s *Server) handleGetTest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := s.quiz.Get(r.Context(), id)
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
	sol, err := s.quiz.Answer(r.Context(), id, in)
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
	res, err := s.quiz.Finish(r.Context(), id)
	s.respond(w, res, err)
}

func (s *Server) handleAbandon(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.respondNoContent(w, s.quiz.Abandon(r.Context(), id))
}

func (s *Server) handleFlag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in quiz.FlagInput
	if !decode(w, r, &in) {
		return
	}
	s.respondNoContent(w, s.quiz.Flag(r.Context(), id, in))
}

// quizError maps quiz errors to responses; it reports false for errors it
// does not know.
func quizError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, quiz.ErrNotFound):
		writeError(w, http.StatusNotFound, "Test no encontrado")
	case errors.Is(err, quiz.ErrNotInProgress):
		writeError(w, http.StatusConflict, "El test ya ha terminado")
	case errors.Is(err, quiz.ErrExpired):
		writeError(w, http.StatusConflict, "Se ha agotado el tiempo")
	case errors.Is(err, quiz.ErrAlreadyAnswered):
		writeError(w, http.StatusConflict, "Esta pregunta ya está respondida")
	case errors.Is(err, quiz.ErrNoQuestions):
		writeError(w, http.StatusUnprocessableEntity, "No hay preguntas publicadas con esos filtros")
	default:
		return false
	}
	return true
}
