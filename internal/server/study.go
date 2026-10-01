package server

import (
	"net/http"
	"time"

	"github.com/alexzafra13/tai_tests/internal/quiz"
	"github.com/alexzafra13/tai_tests/internal/srs"
	"github.com/alexzafra13/tai_tests/internal/stats"
)

// studySummary is what the home screen needs to suggest what to do next.
type studySummary struct {
	Review srs.Summary `json:"review"`
	// Failed counts questions whose last answer was wrong.
	Failed int `json:"failed"`
}

func (s *Server) studySummary(r *http.Request) (studySummary, error) {
	var sum studySummary
	var err error
	if sum.Review, err = s.srs.Summary(r.Context(), userID(r), time.Now()); err != nil {
		return sum, err
	}
	sum.Failed, err = s.quiz.Available(r.Context(), userID(r), quiz.Filters{Failed: true})
	return sum, err
}

func (s *Server) handleStudySummary(w http.ResponseWriter, r *http.Request) {
	sum, err := s.studySummary(r)
	s.respond(w, sum, err)
}

// handleStats returns everything the statistics page shows, for the caller.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	var out struct {
		studySummary
		Overview stats.Overview     `json:"overview"`
		Topics   []stats.TopicStats `json:"topics"`
		Timeline []stats.Day        `json:"timeline"`
	}
	var err error
	if out.studySummary, err = s.studySummary(r); err != nil {
		s.writeErr(w, err)
		return
	}
	if out.Overview, err = s.stats.Overview(r.Context(), userID(r)); err != nil {
		s.writeErr(w, err)
		return
	}
	if out.Topics, err = s.stats.Topics(r.Context(), userID(r)); err != nil {
		s.writeErr(w, err)
		return
	}
	out.Timeline, err = s.stats.Timeline(r.Context(), userID(r), 30)
	s.respond(w, out, err)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	hits, err := s.content.Search(r.Context(), r.URL.Query().Get("q"), 30)
	s.respond(w, hits, err)
}
