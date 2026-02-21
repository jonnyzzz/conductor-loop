package unit_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestHandlerErrorResponses(t *testing.T) {
	root := t.TempDir()
	logger := log.New(io.Discard, "", 0)
	server, err := api.NewServer(api.Options{RootDir: root, DisableTaskStart: true, Logger: logger})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// 400: invalid JSON
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString("{bad"))
	badRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", badRec.Code)
	}

	// 404: missing task
	notFoundReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/missing?project_id=project", nil)
	notFoundRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", notFoundRec.Code)
	}

	// 500: invalid run-info in task runs
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(filepath.Join(taskDir, "runs", "run-1"), 0o755); err != nil {
		t.Fatalf("mkdir runs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "runs", "run-1", "run-info.yaml"), []byte("invalid: ["), 0o644); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	internalReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task?project_id=project", nil)
	internalRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(internalRec, internalReq)
	if internalRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", internalRec.Code)
	}
}

func TestSSETailerPollInterval(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "agent-stdout.txt")
	if err := os.WriteFile(filePath, []byte(""), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	events := make(chan api.LogLine, 1)
	pollInterval := 100 * time.Millisecond
	tailer, err := api.NewTailer(filePath, "run-1", "stdout", pollInterval, 0, events)
	if err != nil {
		t.Fatalf("NewTailer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tailer.Start(ctx)

	start := time.Now()
	if err := os.WriteFile(filePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	select {
	case <-events:
		elapsed := time.Since(start)
		if elapsed < pollInterval/2 {
			t.Fatalf("expected poll interval delay, got %v", elapsed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for tailer event")
	}

	tailer.Stop()
}

func TestRunDiscoveryLatency(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	discovery, err := api.NewRunDiscovery(root, time.Second)
	if err != nil {
		t.Fatalf("NewRunDiscovery: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	go discovery.Poll(ctx, time.Second)

	select {
	case <-discovery.NewRuns():
		elapsed := time.Since(start)
		if elapsed < 900*time.Millisecond {
			t.Fatalf("expected ~1s discovery interval, got %v", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for discovery")
	}
}

