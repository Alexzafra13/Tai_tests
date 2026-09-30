package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alexzafra13/tai_tests/internal/content"
)

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
	if quizError(w, err) {
		return
	}
	var v content.ValidationError
	switch {
	case errors.As(err, &v):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "Revisa los campos marcados", "fields": v})
	case errors.Is(err, content.ErrNotFound):
		writeError(w, http.StatusNotFound, "No encontrado")
	case errors.Is(err, content.ErrHasHistory):
		writeError(w, http.StatusConflict, "Esta pregunta ya tiene respuestas: márcala como descartada en lugar de borrarla")
	case errors.Is(err, content.ErrInUse):
		writeError(w, http.StatusConflict, "No se puede borrar: hay preguntas que dependen de este elemento")
	default:
		s.internalError(w, err)
	}
}
