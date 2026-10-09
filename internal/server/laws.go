package server

import (
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleStudyTopic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	st, err := s.content.StudyTopic(r.Context(), id, time.Now())
	s.respond(w, st, err)
}

// handleLawText returns a law in force today; ?topic= limits it to the
// part studied in that topic.
func (s *Server) handleLawText(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var topic int64
	if v := r.URL.Query().Get("topic"); v != "" {
		var err error
		if topic, err = strconv.ParseInt(v, 10, 64); err != nil || topic <= 0 {
			writeError(w, http.StatusBadRequest, "Tema no válido")
			return
		}
	}
	text, err := s.content.LawText(r.Context(), id, topic, time.Now())
	s.respond(w, text, err)
}

// handleReportNote records the caller's report on a point of a topic's
// study note.
func (s *Server) handleReportNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Point   string `json:"point"`
		Excerpt string `json:"excerpt"`
		Note    string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	s.respondNoContent(w, s.content.ReportNote(r.Context(), userID(r), id, in.Point, in.Excerpt, in.Note))
}

func (s *Server) handleNoteReports(w http.ResponseWriter, r *http.Request) {
	reports, err := s.content.OpenNoteReports(r.Context())
	s.respond(w, reports, err)
}

func (s *Server) handleResolveNoteReport(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.respondNoContent(w, s.content.ResolveNoteReport(r.Context(), id))
}
