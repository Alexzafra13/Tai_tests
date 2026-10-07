package server

import "net/http"

// routes is the list of API endpoints and the access each one requires.
// Access is enforced here, on the server; the frontend only hides what a
// role cannot use.
func (s *Server) routes(mux *http.ServeMux) {
	user, admin := s.requireUser, s.requireAdmin

	// Public.
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/setup", s.handleSetupStatus)
	mux.HandleFunc("POST /api/setup", s.handleSetup)

	// Any logged-in user: their account, studying and reporting doubts.
	mux.Handle("GET /api/auth/me", user(s.handleMe))
	mux.Handle("PUT /api/account/password", user(s.handleChangePassword))

	mux.Handle("GET /api/syllabus", user(s.handleSyllabus))
	mux.Handle("GET /api/sources", user(s.handleListSources))
	mux.Handle("GET /api/exams", user(s.handleExams))
	mux.Handle("GET /api/settings/scoring", user(s.handleGetScoring))

	mux.Handle("GET /api/tests", user(s.handleListTests))
	mux.Handle("POST /api/tests", user(s.handleCreateTest))
	mux.Handle("POST /api/tests/available", user(s.handleAvailable))
	mux.Handle("GET /api/tests/{id}", user(s.handleGetTest))
	mux.Handle("POST /api/tests/{id}/answer", user(s.handleAnswer))
	mux.Handle("POST /api/tests/{id}/finish", user(s.handleFinish))
	mux.Handle("POST /api/tests/{id}/abandon", user(s.handleAbandon))
	mux.Handle("POST /api/tests/{id}/report", user(s.handleReport))

	mux.Handle("GET /api/stats", user(s.handleStats))
	mux.Handle("GET /api/search", user(s.handleSearch))

	// Administrators: content, review, scoring rules and accounts.
	mux.Handle("POST /api/sources", admin(s.handleCreateSource))
	mux.Handle("GET /api/sources/{id}", admin(s.handleGetSource))
	mux.Handle("PUT /api/sources/{id}", admin(s.handleUpdateSource))
	mux.Handle("DELETE /api/sources/{id}", admin(s.handleDeleteSource))
	mux.Handle("POST /api/sources/{id}/check-quote", admin(s.handleCheckQuote))

	mux.Handle("GET /api/questions", admin(s.handleListQuestions))
	mux.Handle("POST /api/questions", admin(s.handleCreateQuestion))
	mux.Handle("GET /api/questions/{id}", admin(s.handleGetQuestion))
	mux.Handle("PUT /api/questions/{id}", admin(s.handleUpdateQuestion))
	mux.Handle("DELETE /api/questions/{id}", admin(s.handleDeleteQuestion))

	mux.Handle("GET /api/review", admin(s.handleReviewQueue))
	mux.Handle("GET /api/review/counts", admin(s.handleReviewCounts))
	mux.Handle("POST /api/review/accept-batch", admin(s.handleAcceptBatch))
	mux.Handle("POST /api/review/{id}/accept", admin(s.handleAccept))
	mux.Handle("POST /api/review/{id}/discard", admin(s.handleDiscard))
	mux.Handle("POST /api/review/{id}/restore", admin(s.handleRestore))

	mux.Handle("PUT /api/settings/scoring", admin(s.handleSetScoring))

	mux.Handle("GET /api/users", admin(s.handleListUsers))
	mux.Handle("POST /api/users", admin(s.handleCreateUser))
	mux.Handle("PUT /api/users/{id}", admin(s.handleUpdateUser))
	mux.Handle("PUT /api/users/{id}/password", admin(s.handleResetPassword))

	// Unknown API routes: 401 before login so nothing is revealed, 404 after.
	mux.Handle("/api/", user(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	}))
}
