package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTraversalSecurityServer(t *testing.T) (*Server, string, string) {
	t.Helper()

	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return server, root, base
}

func doTraversalSecurityRequest(t *testing.T, server *Server, method, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}

func TestV1MessagesRejectTraversalProjectIDQuery(t *testing.T) {
	server, _, _ := newTraversalSecurityServer(t)

	rec := doTraversalSecurityRequest(t, server, http.MethodGet, "/api/v1/messages?project_id=%2e%2e", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestV1MessagesRejectTraversalTaskIDQuery(t *testing.T) {
	server, _, _ := newTraversalSecurityServer(t)

	rec := doTraversalSecurityRequest(t, server, http.MethodGet, "/api/v1/messages?project_id=proj&task_id=%2e%2e", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestV1MessagesStreamRejectTraversalProjectIDQuery(t *testing.T) {
	server, _, _ := newTraversalSecurityServer(t)

	rec := doTraversalSecurityRequest(t, server, http.MethodGet, "/api/v1/messages/stream?project_id=%2e%2e", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectMessagesRejectEncodedTraversalPath(t *testing.T) {
	server, _, base := newTraversalSecurityServer(t)
	outsideBusPath := filepath.Join(base, "PROJECT-MESSAGE-BUS.md")

	rec := doTraversalSecurityRequest(t, server, http.MethodPost, "/api/projects/%2e%2e/messages", []byte(`{"type":"USER","body":"escape attempt"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(outsideBusPath); !os.IsNotExist(err) {
		t.Fatalf("expected no outside-root write at %s", outsideBusPath)
	}
}

func TestTaskMessagesRejectEncodedTraversalPath(t *testing.T) {
	server, root, _ := newTraversalSecurityServer(t)
	rootBusPath := filepath.Join(root, "TASK-MESSAGE-BUS.md")

	rec := doTraversalSecurityRequest(t, server, http.MethodPost, "/api/projects/proj/tasks/%2e%2e/messages", []byte(`{"type":"USER","body":"escape attempt"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(rootBusPath); !os.IsNotExist(err) {
		t.Fatalf("expected no traversal write at %s", rootBusPath)
	}
}

func TestProjectDeleteRejectsTraversalAndPreservesOutsidePaths(t *testing.T) {
	server, _, base := newTraversalSecurityServer(t)
	outsideFile := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write outside marker: %v", err)
	}

	rec := doTraversalSecurityRequest(t, server, http.MethodDelete, "/api/projects/%2e%2e", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(outsideFile); err != nil {
		t.Fatalf("outside marker should be preserved: %v", err)
	}
}

func TestTaskDeleteRejectsTraversalAndPreservesTaskTree(t *testing.T) {
	server, root, _ := newTraversalSecurityServer(t)
	taskDir := filepath.Join(root, "proj", "task-a")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	rec := doTraversalSecurityRequest(t, server, http.MethodDelete, "/api/projects/proj/tasks/%2e%2e", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(taskDir); err != nil {
		t.Fatalf("task directory should be preserved: %v", err)
	}
}

func TestRunDeleteRejectsTraversalTaskID(t *testing.T) {
	server, _, _ := newTraversalSecurityServer(t)

	rec := doTraversalSecurityRequest(t, server, http.MethodDelete, "/api/projects/proj/tasks/%2e%2e/runs/run-1", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
