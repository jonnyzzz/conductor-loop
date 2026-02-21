package api

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoutesServeEmbeddedUIFallback(t *testing.T) {
	chdir(t, t.TempDir())

	server, err := NewServer(Options{
		RootDir:          t.TempDir(),
		DisableTaskStart: true,
		Logger:           log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	rootBody := getBody(t, server.Handler(), "/")
	if !strings.Contains(rootBody, "CONDUCTOR LOOP") {
		t.Fatalf("expected embedded fallback UI, got: %q", rootBody)
	}

	uiBody := getBody(t, server.Handler(), "/ui/")
	if !strings.Contains(uiBody, "CONDUCTOR LOOP") {
		t.Fatalf("expected embedded /ui/ content, got: %q", uiBody)
	}

	appBody := getBody(t, server.Handler(), "/ui/app.js")
	if !strings.Contains(appBody, "'use strict';") {
		t.Fatalf("expected embedded JS asset, got: %q", appBody)
	}
}

func TestRoutesPreferFrontendDistWhenPresent(t *testing.T) {
	cwd := t.TempDir()
	frontendDist := filepath.Join(cwd, "frontend", "dist")
	if err := os.MkdirAll(frontendDist, 0o755); err != nil {
		t.Fatalf("mkdir frontend/dist: %v", err)
	}
	index := "<html><body>disk-frontend-dist</body></html>"
	if err := os.WriteFile(filepath.Join(frontendDist, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	chdir(t, cwd)

	server, err := NewServer(Options{
		RootDir:          t.TempDir(),
		DisableTaskStart: true,
		Logger:           log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	rootBody := getBody(t, server.Handler(), "/")
	if !strings.Contains(rootBody, "disk-frontend-dist") {
		t.Fatalf("expected frontend/dist override, got: %q", rootBody)
	}

	uiBody := getBody(t, server.Handler(), "/ui/")
	if !strings.Contains(uiBody, "disk-frontend-dist") {
		t.Fatalf("expected /ui/ to use frontend/dist override, got: %q", uiBody)
	}
}

func getBody(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s returned %d: %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
}
