package api

import "net/http"

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/health", s.wrap(s.handleHealth))
	mux.Handle("/api/v1/version", s.wrap(s.handleVersion))

	mux.Handle("/api/v1/runs/stream/all", s.wrap(s.handleAllRunsStream))

	mux.Handle("/api/v1/tasks", s.wrap(s.handleTasks))
	mux.Handle("/api/v1/tasks/", s.wrap(s.handleTaskByID))

	mux.Handle("/api/v1/runs", s.wrap(s.handleRuns))
	mux.Handle("/api/v1/runs/", s.wrap(s.handleRunByID))

	mux.Handle("/api/v1/messages", s.wrap(s.handleMessages))
	mux.Handle("/api/v1/messages/stream", s.wrap(s.handleMessageStream))

	handler := http.Handler(mux)
	handler = s.withAuth(handler)
	handler = s.withCORS(handler)
	handler = s.withLogging(handler)
	return handler
}
