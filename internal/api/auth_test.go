package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAPIKey(t *testing.T) {
	// inner is a trivial handler that always returns 200 OK.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name        string
		key         string
		path        string
		method      string
		authHeader  string
		apiKeyHdr   string
		wantStatus  int
		wantWWWAuth bool
	}{
		// Empty key: auth disabled, all requests pass through.
		{name: "no-key pass-through", key: "", path: "/api/v1/tasks", wantStatus: 200},

		// Correct Authorization: Bearer <key>.
		{name: "bearer ok", key: "secret", path: "/api/v1/tasks", authHeader: "Bearer secret", wantStatus: 200},

		// Correct X-API-Key header.
		{name: "x-api-key ok", key: "secret", path: "/api/v1/tasks", apiKeyHdr: "secret", wantStatus: 200},

		// Wrong Bearer value.
		{name: "bearer wrong", key: "secret", path: "/api/v1/tasks", authHeader: "Bearer wrong", wantStatus: 401, wantWWWAuth: true},

		// No auth header at all.
		{name: "no auth header", key: "secret", path: "/api/v1/tasks", wantStatus: 401, wantWWWAuth: true},

		// Exempt paths always pass through even without credentials.
		{name: "exempt health", key: "secret", path: "/api/v1/health", wantStatus: 200},
		{name: "exempt version", key: "secret", path: "/api/v1/version", wantStatus: 200},
		{name: "exempt metrics", key: "secret", path: "/metrics", wantStatus: 200},
		{name: "exempt ui index", key: "secret", path: "/ui/index.html", wantStatus: 200},
		{name: "exempt ui root", key: "secret", path: "/ui/", wantStatus: 200},

		// OPTIONS preflight passes through even without credentials.
		{name: "options preflight", key: "secret", path: "/api/v1/tasks", method: http.MethodOptions, wantStatus: 200},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			handler := RequireAPIKey(tc.key)(inner)

			method := tc.method
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, tc.path, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			if tc.apiKeyHdr != "" {
				req.Header.Set("X-API-Key", tc.apiKeyHdr)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantWWWAuth {
				if rr.Header().Get("WWW-Authenticate") == "" {
					t.Error("expected WWW-Authenticate header on 401 response")
				}
				if rr.Body.Len() == 0 {
					t.Error("expected non-empty JSON error body on 401 response")
				}
			}
		})
	}
}

func TestIsAuthExemptPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/health", true},
		{"/api/v1/version", true},
		{"/metrics", true},
		{"/ui/", true},
		{"/ui/index.html", true},
		{"/ui/assets/main.js", true},
		{"/api/v1/tasks", false},
		{"/api/v1/runs", false},
		{"/api/projects", false},
		{"/", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			got := isAuthExemptPath(tc.path)
			if got != tc.want {
				t.Errorf("isAuthExemptPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
