package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/quiz"
	"github.com/alexzafra13/tai_tests/internal/users"
	"github.com/alexzafra13/tai_tests/internal/validate"

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

// errorResponses maps domain errors to HTTP responses (Spanish messages
// shown to the user). Validation errors are handled separately because they
// carry per-field messages.
var errorResponses = []struct {
	err     error
	status  int
	message string
}{
	{content.ErrNotFound, http.StatusNotFound, "No encontrado"},
	{content.ErrHasHistory, http.StatusConflict, "Esta pregunta ya tiene respuestas: márcala como descartada en lugar de borrarla"},
	{content.ErrInUse, http.StatusConflict, "No se puede borrar: hay preguntas que dependen de este elemento"},
	{quiz.ErrNotFound, http.StatusNotFound, "Test no encontrado"},
	{quiz.ErrNotInProgress, http.StatusConflict, "El test ya ha terminado"},
	{quiz.ErrExpired, http.StatusConflict, "Se ha agotado el tiempo"},
	{quiz.ErrAlreadyAnswered, http.StatusConflict, "Esta pregunta ya está respondida"},
	{quiz.ErrNoQuestions, http.StatusUnprocessableEntity, "No hay preguntas publicadas con esos filtros"},
	{users.ErrNotFound, http.StatusNotFound, "Usuario no encontrado"},
	{users.ErrLastAdmin, http.StatusConflict, "Tiene que quedar al menos un administrador activo"},
	{users.ErrAlreadySetUp, http.StatusConflict, "La aplicación ya está configurada: inicia sesión"},
	{auth.ErrBadCredentials, http.StatusUnauthorized, "Usuario o contraseña incorrectos"},
	{auth.ErrRateLimited, http.StatusTooManyRequests, "Demasiados intentos fallidos. Espera unos minutos."},
}

// writeErr turns an error into a response: validation errors become 422
// with the field messages the forms display inline, known domain errors use
// errorResponses, and anything else is logged as an internal error.
func (s *Server) writeErr(w http.ResponseWriter, err error) {
	var v validate.Errors
	if errors.As(err, &v) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "Revisa los campos marcados", "fields": v})
		return
	}
	if errors.Is(err, auth.ErrWrongPassword) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "Revisa los campos marcados",
			"fields": validate.Errors{"current_password": "La contraseña actual no es correcta"}})
		return
	}
	for _, e := range errorResponses {
		if errors.Is(err, e.err) {
			writeError(w, e.status, e.message)
			return
		}
	}
	s.internalError(w, err)
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.log.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
