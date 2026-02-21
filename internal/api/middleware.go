package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
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
		s.logger.Printf("api error: %s: %v", message, err.Err)
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
		start := s.now()
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		duration := s.now().Sub(start)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		if s.logger != nil {
			s.logger.Printf("%s %s %d %dB %s", r.Method, r.URL.Path, status, recorder.bytes, duration.Truncate(time.Millisecond))
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
