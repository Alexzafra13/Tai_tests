package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alexzafra13/tai_tests/internal/content"
)

func (s *Server) contentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/syllabus", s.handleSyllabus)

	mux.HandleFunc("GET /api/sources", s.handleListSources)
	mux.HandleFunc("POST /api/sources", s.handleCreateSource)
	mux.HandleFunc("GET /api/sources/{id}", s.handleGetSource)
	mux.HandleFunc("PUT /api/sources/{id}", s.handleUpdateSource)
	mux.HandleFunc("DELETE /api/sources/{id}", s.handleDeleteSource)
	mux.HandleFunc("POST /api/sources/{id}/check-quote", s.handleCheckQuote)

	mux.HandleFunc("GET /api/questions", s.handleListQuestions)
	mux.HandleFunc("POST /api/questions", s.handleCreateQuestion)
	mux.HandleFunc("GET /api/questions/{id}", s.handleGetQuestion)
	mux.HandleFunc("PUT /api/questions/{id}", s.handleUpdateQuestion)
	mux.HandleFunc("DELETE /api/questions/{id}", s.handleDeleteQuestion)
}

func (s *Server) handleSyllabus(w http.ResponseWriter, r *http.Request) {
	blocks, err := s.content.Syllabus(r.Context())
	s.respond(w, blocks, err)
}

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	list, err := s.content.ListSources(r.Context(), content.SourceKind(r.URL.Query().Get("kind")))
	s.respond(w, list, err)
}

func (s *Server) handleGetSource(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	src, err := s.content.Source(r.Context(), id)
	s.respond(w, src, err)
}

func (s *Server) handleCreateSource(w http.ResponseWriter, r *http.Request) {
	var in content.SourceInput
	if !decode(w, r, &in) {
		return
	}
	id, err := s.content.CreateSource(r.Context(), in)
	s.respondCreated(w, id, err)
}

func (s *Server) handleUpdateSource(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in content.SourceInput
	if !decode(w, r, &in) {
		return
	}
	s.respondNoContent(w, s.content.UpdateSource(r.Context(), id, in))
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.respondNoContent(w, s.content.DeleteSource(r.Context(), id))
}

func (s *Server) handleCheckQuote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Quote string `json:"quote"`
	}
	if !decode(w, r, &in) {
		return
	}
	found, err := s.content.CheckQuote(r.Context(), id, in.Quote)
	s.respond(w, map[string]bool{"found": found}, err)
}

func (s *Server) handleListQuestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := content.QuestionFilter{
		Status:   content.Status(q.Get("status")),
		Origin:   content.Origin(q.Get("origin")),
		SourceID: queryInt(q.Get("source")),
		TopicID:  queryInt(q.Get("topic")),
		BlockID:  queryInt(q.Get("block")),
		Text:     q.Get("q"),
		Limit:    int(queryInt(q.Get("limit"))),
		Offset:   int(queryInt(q.Get("offset"))),
	}
	if v := q.Get("flagged"); v != "" {
		b := v == "1" || v == "true"
		f.Flagged = &b
	}
	page, err := s.content.ListQuestions(r.Context(), f)
	s.respond(w, page, err)
}

func (s *Server) handleGetQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	q, err := s.content.Question(r.Context(), id)
	s.respond(w, q, err)
}

func (s *Server) handleCreateQuestion(w http.ResponseWriter, r *http.Request) {
	var in content.QuestionInput
	if !decode(w, r, &in) {
		return
	}
	id, err := s.content.CreateQuestion(r.Context(), in)
	s.respondCreated(w, id, err)
}

func (s *Server) handleUpdateQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in content.QuestionInput
	if !decode(w, r, &in) {
		return
	}
	s.respondNoContent(w, s.content.UpdateQuestion(r.Context(), id, in))
}

func (s *Server) handleDeleteQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.respondNoContent(w, s.content.DeleteQuestion(r.Context(), id))
}

// --- helpers ---------------------------------------------------------------

// maxBody bounds request bodies; law texts pasted as sources can be large.
const maxBody = 8 << 20

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Cuerpo de la petición no válido: "+err.Error())
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id no válido")
		return 0, false
	}
	return id, true
}

func queryInt(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func (s *Server) respond(w http.ResponseWriter, v any, err error) {
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) respondCreated(w http.ResponseWriter, id int64, err error) {
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) respondNoContent(w http.ResponseWriter, err error) {
	if err != nil {
		s.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeErr maps domain errors to HTTP responses. Validation errors carry
// per-field messages the forms display inline.
func (s *Server) writeErr(w http.ResponseWriter, err error) {
	var v content.ValidationError
	switch {
	case errors.As(err, &v):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "Revisa los campos marcados", "fields": v})
	case errors.Is(err, content.ErrNotFound):
		writeError(w, http.StatusNotFound, "No encontrado")
	case errors.Is(err, content.ErrInUse):
		writeError(w, http.StatusConflict, "No se puede borrar: hay preguntas que dependen de este elemento")
	default:
		s.internalError(w, err)
	}
}
