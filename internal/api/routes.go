package api

import (
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", http.HandlerFunc(s.handleMetrics))
	mux.Handle("/api/v1/health", s.wrap(s.handleHealth))
	mux.Handle("/api/v1/version", s.wrap(s.handleVersion))
	mux.Handle("/api/v1/status", s.wrap(s.handleStatus))

	mux.Handle("/api/v1/runs/stream/all", s.wrap(s.handleAllRunsStream))

	mux.Handle("/api/v1/tasks", s.wrap(s.handleTasks))
	mux.Handle("/api/v1/tasks/", s.wrap(s.handleTaskByID))

	mux.Handle("/api/v1/runs", s.wrap(s.handleRuns))
	mux.Handle("/api/v1/runs/", s.wrap(s.handleRunByID))

	mux.Handle("/api/v1/messages", s.wrap(s.handleMessages))
	mux.Handle("POST /api/v1/messages", s.wrap(s.handlePostMessage))
	mux.Handle("/api/v1/messages/stream", s.wrap(s.handleMessageStream))

	// Project-centric API (used by the web UI)
	mux.Handle("/api/projects", s.wrap(s.handleProjectsList))
	mux.Handle("/api/projects/", s.wrap(s.handleProjectsRouter))

	// Serve web UI static files at root if available
	if webDir, ok := findWebDir(); ok {
		if s.logger != nil {
			s.logger.Printf("serving web UI from %s at /", webDir)
		}
		mux.Handle("/", http.FileServer(http.Dir(webDir)))
	}

	handler := http.Handler(mux)
	handler = s.withAuth(handler)
	handler = s.withCORS(handler)
	handler = s.withLogging(handler)
	return handler
}

// findWebDir searches for the web UI directory containing index.html.
// frontend/dist (built React app) is preferred over web/src (simple UI fallback).
// It checks relative to the executable and to the current working directory.
func findWebDir() (string, bool) {
	var candidates []string

	// Relative to the executable (handles installed binary and go run)
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "frontend", "dist"),
			filepath.Join(base, "..", "frontend", "dist"),
			filepath.Join(base, "..", "..", "frontend", "dist"),
			filepath.Join(base, "web", "src"),
			filepath.Join(base, "..", "web", "src"),
			filepath.Join(base, "..", "..", "web", "src"),
		)
	}

	// Relative to current working directory (handles go run from project root)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "frontend", "dist"),
			filepath.Join(cwd, "web", "src"),
		)
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			abs, err := filepath.Abs(dir)
			if err == nil {
				return abs, true
			}
			return dir, true
		}
	}
	return "", false
}
