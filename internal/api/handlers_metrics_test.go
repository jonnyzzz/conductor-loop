package api

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleMetricsGet(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/plain; version=0.0.4" {
		t.Fatalf("unexpected content-type: %q", ct)
	}
	body := rec.Body.String()
	for _, metric := range []string{
		"conductor_uptime_seconds",
		"conductor_active_runs_total",
		"conductor_completed_runs_total",
		"conductor_failed_runs_total",
		"conductor_messagebus_appends_total",
		"conductor_api_requests_total",
	} {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metric %q in response body:\n%s", metric, body)
		}
	}
}

func TestHandleMetricsMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/metrics", nil)
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("method %s: expected 405, got %d", method, rec.Code)
		}
	}
}

func TestHandleMetricsRequestCounting(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Make a couple of requests to a tracked endpoint.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
	}

	// Now check metrics.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `conductor_api_requests_total{method="GET",status="200"} 3`) {
		t.Errorf("expected GET:200=3 in metrics output:\n%s", body)
	}
}
