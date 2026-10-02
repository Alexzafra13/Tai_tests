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

// defaultZone is used when the browser does not send a valid time zone.
// The app is for a Spanish exam, so Madrid is the sensible guess.
var defaultZone = mustLoadLocation("Europe/Madrid")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// clientZone is the caller's time zone from ?tz= (the app sends the
// browser's, e.g. "Atlantic/Canary"), so nothing has to be configured.
func clientZone(r *http.Request) *time.Location {
	if name := r.URL.Query().Get("tz"); name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return defaultZone
}

// handleStats returns everything the statistics page shows, for the caller.
// Days are calendar days in the caller's time zone.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	loc := clientZone(r)
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
	if out.Overview, err = s.stats.Overview(r.Context(), userID(r), loc); err != nil {
		s.writeErr(w, err)
		return
	}
	if out.Topics, err = s.stats.Topics(r.Context(), userID(r)); err != nil {
		s.writeErr(w, err)
		return
	}
	out.Timeline, err = s.stats.Timeline(r.Context(), userID(r), 30, loc)
	s.respond(w, out, err)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	hits, err := s.content.Search(r.Context(), r.URL.Query().Get("q"), 30)
	s.respond(w, hits, err)
}
