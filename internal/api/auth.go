package api

import (
	"net/http"
	"strings"
)

// RequireAPIKey returns HTTP middleware that enforces API key authentication.
// If key is empty, the returned middleware is a no-op pass-through (auth disabled).
// Requests to exempt paths (/api/v1/health, /api/v1/version, /metrics, /ui/) always pass through.
// The key is accepted via "Authorization: Bearer <key>" or "X-API-Key: <key>" headers.
// Unauthorized requests receive a 401 response with a JSON error body and WWW-Authenticate header.
func RequireAPIKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt paths always pass through regardless of auth.
			if isAuthExemptPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// OPTIONS preflight always passes through.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Check Authorization: Bearer <key>.
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				if strings.TrimPrefix(auth, "Bearer ") == key {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check X-API-Key: <key>.
			if r.Header.Get("X-API-Key") == key {
				next.ServeHTTP(w, r)
				return
			}

			// Reject with 401 Unauthorized.
			w.Header().Set("WWW-Authenticate", `Bearer realm="conductor"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","message":"valid API key required"}`))
		})
	}
}

// isAuthExemptPath reports whether the given path is exempt from API key checks.
func isAuthExemptPath(path string) bool {
	return path == "/api/v1/health" ||
		path == "/api/v1/version" ||
		path == "/metrics" ||
		path == "/healthz" ||
		strings.HasPrefix(path, "/ui/")
}
