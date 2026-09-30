package server

import (
	"net/http"

	"github.com/alexzafra13/tai_tests/internal/content"
)

func (s *Server) handleReviewQueue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := s.content.ReviewQueue(r.Context(), content.ReviewKind(q.Get("kind")), queryInt(q.Get("source")),
		int(queryInt(q.Get("offset"))), int(queryInt(q.Get("limit"))))
	s.respond(w, page, err)
}

func (s *Server) handleReviewCounts(w http.ResponseWriter, r *http.Request) {
	c, err := s.content.ReviewCounts(r.Context())
	s.respond(w, c, err)
}

// Decisions return the previous state so the client can offer "undo".
type decisionResponse struct {
	Previous content.ReviewState `json:"previous"`
}

func (s *Server) handleAccept(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		TopicIDs []int64 `json:"topic_ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	prev, err := s.content.Accept(r.Context(), id, in.TopicIDs)
	s.respond(w, decisionResponse{prev}, err)
}

func (s *Server) handleDiscard(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	prev, err := s.content.Discard(r.Context(), id)
	s.respond(w, decisionResponse{prev}, err)
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var st content.ReviewState
	if !decode(w, r, &st) {
		return
	}
	s.respondNoContent(w, s.content.RestoreReview(r.Context(), id, st))
}

func (s *Server) handleAcceptBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.IDs) > 500 {
		writeError(w, http.StatusBadRequest, "Como máximo 500 preguntas por lote")
		return
	}
	res, err := s.content.AcceptBatch(r.Context(), in.IDs)
	s.respond(w, map[string]any{"results": res}, err)
}
