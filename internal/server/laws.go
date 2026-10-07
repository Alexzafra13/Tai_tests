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
