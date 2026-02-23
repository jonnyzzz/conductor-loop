package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
)

type apiError struct {
	Status  int
	Code    string
	Message string
	Details map[string]string
	Err     error
}

type errorResponse struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func apiErrorBadRequest(message string) *apiError {
	return &apiError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: message}
}

func apiErrorNotFound(message string) *apiError {
	return &apiError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: message}
}

func apiErrorConflict(message string, details map[string]string) *apiError {
	return &apiError{Status: http.StatusConflict, Code: "CONFLICT", Message: message, Details: details}
}

func apiErrorMethodNotAllowed() *apiError {
	return &apiError{Status: http.StatusMethodNotAllowed, Code: "METHOD_NOT_ALLOWED", Message: "method not allowed"}
}

func apiErrorForbidden(message string) *apiError {
	return &apiError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

func apiErrorInternal(message string, err error) *apiError {
	return &apiError{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: message, Err: err}
}

func (s *Server) writeError(w http.ResponseWriter, err *apiError) {
	if err == nil || w == nil {
		return
	}
	status := err.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}
	code := err.Code
	if code == "" {
		code = "INTERNAL"
	}
	message := err.Message
	if message == "" {
		message = http.StatusText(status)
	}
	if err.Err != nil && s != nil && s.logger != nil {
		obslog.Log(s.logger, "ERROR", "api", "request_error",
			obslog.F("status", status),
			obslog.F("code", code),
			obslog.F("message", message),
			obslog.F("error", err.Err),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	payload := errorResponse{
		Error: errorPayload{
			Code:    code,
			Message: message,
			Details: err.Details,
		},
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if requestID == "" {
			now := time.Now().UTC()
			if s.now != nil {
				now = s.now().UTC()
			}
			requestID = newRequestID(now)
		}
		ctx := withRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		start := s.now()
		recorder := &responseRecorder{ResponseWriter: w}
		recorder.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(recorder, r)
		duration := s.now().Sub(start)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		if s.logger != nil {
			projectID, taskID, runID := extractLogIdentifiers(r)
			obslog.Log(s.logger, "INFO", "api", "request_completed",
				obslog.F("request_id", requestID),
				obslog.F("correlation_id", requestID),
				obslog.F("method", r.Method),
				obslog.F("path", r.URL.Path),
				obslog.F("status", status),
				obslog.F("bytes", recorder.bytes),
				obslog.F("duration_ms", duration.Truncate(time.Millisecond).Milliseconds()),
				obslog.F("project_id", projectID),
				obslog.F("task_id", taskID),
				obslog.F("run_id", runID),
			)
		}
		s.metrics.RecordRequest(r.Method, status)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	origins := s.apiConfig.CORSOrigins
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		allowedOrigin := ""
		if origin != "" && originAllowed(origin, origins) {
			if containsWildcard(origins) {
				allowedOrigin = "*"
			} else {
				allowedOrigin = origin
			}
		}
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Conductor-Client")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	key := ""
	if s.apiConfig.AuthEnabled {
		key = s.apiConfig.APIKey
	}
	return RequireAPIKey(key)(next)
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" || trimmed == origin {
			return true
		}
	}
	return false
}

func containsWildcard(allowed []string) bool {
	for _, candidate := range allowed {
		if strings.TrimSpace(candidate) == "*" {
			return true
		}
	}
	return false
}

func extractLogIdentifiers(r *http.Request) (projectID, taskID, runID string) {
	if r == nil || r.URL == nil {
		return "", "", ""
	}
	projectID = strings.TrimSpace(r.URL.Query().Get("project_id"))
	taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	runID = strings.TrimSpace(r.URL.Query().Get("run_id"))

	path := strings.TrimSpace(r.URL.Path)
	if strings.HasPrefix(path, "/api/projects/") {
		parts := splitPath(path, "/api/projects/")
		if len(parts) > 0 && projectID == "" {
			projectID = strings.TrimSpace(parts[0])
		}
		if len(parts) > 2 && parts[1] == "tasks" && taskID == "" {
			taskID = strings.TrimSpace(parts[2])
		}
		if len(parts) > 4 && parts[3] == "runs" && runID == "" {
			runID = strings.TrimSpace(parts[4])
		}
	}

	if runID == "" && strings.HasPrefix(path, "/api/v1/runs/") {
		parts := pathSegments(path, "/api/v1/runs/")
		if len(parts) > 0 {
			runID = strings.TrimSpace(parts[0])
		}
	}

	if taskID == "" && strings.HasPrefix(path, "/api/v1/tasks/") {
		parts := pathSegments(path, "/api/v1/tasks/")
		if len(parts) > 0 {
			taskID = strings.TrimSpace(parts[0])
		}
	}

	return projectID, taskID, runID
}
