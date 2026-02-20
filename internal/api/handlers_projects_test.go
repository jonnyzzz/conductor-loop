package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func makeProjectRun(t *testing.T, root, projectID, taskID, runID string, status string, stdoutContent string) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte(stdoutContent), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:      runID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Status:     status,
		StartTime:  time.Now().UTC(),
		StdoutPath: stdoutPath,
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return info
}

func TestServeRunFileStream_UnknownFile(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "hello\n")

	url := "/api/projects/project/tasks/task/runs/run-1/stream?name=badfile"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown file, got %d", rec.Code)
	}
}

func TestServeRunFileStream_BasicContent(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "line1\nline2\n")

	ctx, cancel := context.WithCancel(context.Background())
	url := "/api/projects/project/tasks/task/runs/run-1/stream?name=stdout"
	req := httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for SSE data to arrive.
	deadline := time.After(2 * time.Second)
	for {
		if bytes.Contains(rec.Bytes(), []byte("line1")) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for stream content; got: %q", string(rec.Bytes()))
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Should also receive a done event (run is completed).
	deadline2 := time.After(2 * time.Second)
	for {
		if bytes.Contains(rec.Bytes(), []byte("event: done")) {
			break
		}
		select {
		case <-deadline2:
			t.Fatalf("timeout waiting for done event; got: %q", string(rec.Bytes()))
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("stream handler did not exit after context cancel")
	}

	body := string(rec.Bytes())
	if !strings.Contains(body, "data: line1") {
		t.Errorf("expected 'data: line1' in response, got: %q", body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream content type, got %q", ct)
	}
}

func TestServeRunFileStream_RunNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/missing-run/stream?name=stdout"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing run, got %d", rec.Code)
	}
}

func TestServeRunFile_ProjectEndpoint(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "hello world\n")

	url := "/api/projects/project/tasks/task/runs/run-1/file?name=stdout"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "hello world") {
		t.Errorf("expected file content in response, got: %q", rec.Body.String())
	}
}
