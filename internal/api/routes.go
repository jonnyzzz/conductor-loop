package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	webui "github.com/jonnyzzz/conductor-loop/web"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", http.HandlerFunc(s.handleMetrics))
	mux.Handle("/api/v1/health", s.wrap(s.handleHealth))
	mux.Handle("/api/v1/version", s.wrap(s.handleVersion))
	mux.Handle("/api/v1/status", s.wrap(s.handleStatus))
	mux.Handle("/api/v1/admin/self-update", s.wrap(s.handleSelfUpdate))

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
	mux.Handle("/api/projects/home-dirs", s.wrap(s.handleProjectHomeDirs))
	mux.Handle("/api/projects/", s.wrap(s.handleProjectsRouter))

	// Serve web UI from frontend/dist when present, otherwise from embedded assets.
	if uiFS, source, ok := findWebFS(); ok {
		if s.logger != nil {
			s.logger.Printf("serving web UI from %s", source)
			obslog.Log(s.logger, "INFO", "api", "ui_assets_source_selected",
				obslog.F("source", source),
			)
		}
		fileServer := http.FileServer(uiFS)
		mux.Handle("/", fileServer)
		mux.Handle("/ui/", http.StripPrefix("/ui/", fileServer))
		mux.Handle("/ui", http.RedirectHandler("/ui/", http.StatusMovedPermanently))
	}

	handler := http.Handler(mux)
	handler = s.withAuth(handler)
	handler = s.withCORS(handler)
	handler = s.withLogging(handler)
	return handler
}

// findWebFS selects the UI file system.
// frontend/dist (built React app) is preferred; embedded web/src is fallback.
func findWebFS() (http.FileSystem, string, bool) {
	if webDir, ok := findFrontendDistDir(); ok {
		return http.Dir(webDir), webDir, true
	}
	embedded, err := webui.FileSystem()
	if err != nil {
		return nil, fmt.Sprintf("embedded web UI unavailable: %v", err), false
	}
	return embedded, "embedded:web/src", true
}

// findFrontendDistDir searches for frontend/dist containing index.html.
// It checks relative to the executable and to the current working directory.
func findFrontendDistDir() (string, bool) {
	var candidates []string

	// Relative to the executable (handles installed binary and go run).
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "frontend", "dist"),
			filepath.Join(base, "..", "frontend", "dist"),
			filepath.Join(base, "..", "..", "frontend", "dist"),
		)
	}

	// Relative to current working directory.
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "frontend", "dist"),
			filepath.Join(cwd, "..", "frontend", "dist"),
			filepath.Join(cwd, "..", "..", "frontend", "dist"),
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
