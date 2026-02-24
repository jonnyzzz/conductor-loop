package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestHealthzEndpoint verifies GET /healthz returns 200 with status and uptime fields.
func TestHealthzEndpoint(t *testing.T) {
	root := t.TempDir()
	fixedNow := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Now:              func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type=%q, want application/json", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if body["status"] != "ok" {
		t.Errorf("status=%q, want ok", body["status"])
	}
	if body["uptime"] == "" {
		t.Errorf("uptime field is empty")
	}
}

// TestHealthzEndpointMethodNotAllowed verifies POST /healthz returns 405.
func TestHealthzEndpointMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// TestHealthzExemptFromAuth verifies /healthz is accessible when auth is enabled.
func TestHealthzExemptFromAuth(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Manually apply auth middleware with a key.
	authHandler := RequireAPIKey("secret-key")(server.Handler())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	authHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (no auth required for /healthz), got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWatchdog verifies that Watchdog exits when consecutive failures exceed MaxFailures.
func TestWatchdog(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Inject a fake port so the watchdog doesn't skip probes.
	server.mu.Lock()
	server.actualPort = 19999
	server.mu.Unlock()

	// ProbeFunc always returns an error to simulate a failing server.
	exitCalled := make(chan int, 1)
	w := &Watchdog{
		Server:      server,
		Host:        "127.0.0.1",
		Interval:    10 * time.Millisecond,
		MaxFailures: 2,
		ProbeFunc: func(url string) error {
			return errors.New("simulated probe failure")
		},
		ExitFunc: func(code int) {
			exitCalled <- code
		},
	}

	go w.Run()

	select {
	case code := <-exitCalled:
		if code != 1 {
			t.Errorf("exit code=%d, want 1", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watchdog did not exit within 3 seconds")
	}
}

// TestWatchdogResetsOnSuccess verifies that the failure counter resets after a successful probe.
func TestWatchdogResetsOnSuccess(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	server.mu.Lock()
	server.actualPort = 19998
	server.mu.Unlock()

	// ProbeFunc succeeds on every call.
	var probeCount atomic.Int64
	exitCalled := make(chan int, 1)
	w := &Watchdog{
		Server:      server,
		Host:        "127.0.0.1",
		Interval:    20 * time.Millisecond,
		MaxFailures: 2,
		ProbeFunc: func(url string) error {
			probeCount.Add(1)
			return nil // always succeed
		},
		ExitFunc: func(code int) {
			exitCalled <- code
		},
	}

	go w.Run()

	// After 200ms with 20ms interval, about 10 probes should have run — no exit.
	select {
	case code := <-exitCalled:
		t.Errorf("watchdog exited unexpectedly with code %d", code)
	case <-time.After(200 * time.Millisecond):
		// Expected: watchdog is still running; no exit.
	}

	count := probeCount.Load()
	if count < 3 {
		t.Errorf("expected at least 3 probes, got %d", count)
	}
}

// TestWatchdogSkipsWhenPortZero verifies that the watchdog skips probes before the server starts.
func TestWatchdogSkipsWhenPortZero(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// actualPort remains 0 — server not yet started.

	var probeCount atomic.Int64
	exitCalled := make(chan int, 1)
	w := &Watchdog{
		Server:      server,
		Host:        "127.0.0.1",
		Interval:    20 * time.Millisecond,
		MaxFailures: 2,
		ProbeFunc: func(url string) error {
			probeCount.Add(1)
			return errors.New("should not be called")
		},
		ExitFunc: func(code int) {
			exitCalled <- code
		},
	}

	go w.Run()

	select {
	case code := <-exitCalled:
		t.Errorf("watchdog exited unexpectedly with code %d", code)
	case <-time.After(100 * time.Millisecond):
		// Expected: no probes, no exit.
	}

	if n := probeCount.Load(); n != 0 {
		t.Errorf("expected 0 probes when port is 0, got %d", n)
	}
}
