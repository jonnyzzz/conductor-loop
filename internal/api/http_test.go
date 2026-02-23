package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestHandleHealthAndVersion(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Version: "v1", Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTaskLifecycle(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	payload := TaskCreateRequest{ProjectID: "project", TaskID: "task", AgentType: "codex", Prompt: "prompt"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task?project_id=project", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/task?project_id=project", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "project", "task", "DONE")); err != nil {
		t.Fatalf("expected DONE file: %v", err)
	}
}

func TestRunEndpoints(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusCompleted,
		ExitCode:  0,
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
		PGID:      os.Getpid(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/info", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-1/stop", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestMessagesEndpoint(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	busPath := filepath.Join(root, "project", "PROJECT-MESSAGE-BUS.md")
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir bus dir: %v", err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	msgID, err := bus.AppendMessage(&messagebus.Message{Type: "FACT", ProjectID: "project", Body: "hello"})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?project_id=project", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/messages?project_id=project&after=missing", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	if msgID == "" {
		t.Fatalf("expected msg id")
	}
}

func TestMiddlewareAuthAndCORS(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, APIConfig: config.APIConfig{AuthEnabled: true, APIKey: "test-key", CORSOrigins: []string{"*"}}, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Protected endpoint without credentials returns 401 + CORS header.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected CORS header")
	}

	// Exempt path (health) passes through even without credentials.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for exempt path, got %d", rec.Code)
	}

	// OPTIONS preflight passes through without credentials.
	req = httptest.NewRequest(http.MethodOptions, "/api/v1/tasks", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestFanIn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fan := newFanIn(ctx)
	ch := make(chan SSEEvent, 1)
	sub := &Subscription{events: ch, close: func() { close(ch) }}
	fan.Add(sub)
	ch <- SSEEvent{Event: "log", Data: "hello"}
	select {
	case <-fan.Events():
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected fan-in event")
	}
	cancel()
	fan.Close()
}

func TestRunDiscoveryPoll(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	discovery, err := NewRunDiscovery(root, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRunDiscovery: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go discovery.Poll(ctx, 10*time.Millisecond)
	select {
	case <-discovery.NewRuns():
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected discovery event")
	}
}

func TestRunStreamCatchUp(t *testing.T) {
	runDir := t.TempDir()
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(runDir, "agent-stderr.txt")
	if err := os.WriteFile(stdoutPath, []byte("a\n"+"b\n"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := os.WriteFile(stderrPath, []byte("err\n"), 0o644); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	rs := newRunStream("run-1", runDir, "project", "task", 10*time.Millisecond, 10)
	sub := newSubscriber(4, true)
	rs.catchUp(sub, Cursor{}, Cursor{Stdout: 2, Stderr: 1})
	select {
	case <-sub.events:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected catchup event")
	}
}

func TestStreamRunAndMessages(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/stream", nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}
	_ = server.streamRun(rec, req, "run-1")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/messages/stream?project_id=project", nil).WithContext(ctx)
	rec = &recordingWriter{header: make(http.Header)}
	_ = server.streamMessages(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs/stream/all", nil).WithContext(ctx)
	rec = &recordingWriter{header: make(http.Header)}
	_ = server.streamAllRuns(rec, req)
}
