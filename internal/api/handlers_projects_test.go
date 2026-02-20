package api

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestRunFile_OutputMdFallback(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create a run with agent-stdout.txt but no output.md
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "stdout content\n")

	url := "/api/projects/project/tasks/task/runs/run-1/file?name=output.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for fallback, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["fallback"] != "agent-stdout.txt" {
		t.Errorf("expected fallback=agent-stdout.txt, got %v", resp["fallback"])
	}
	if !strings.Contains(resp["content"].(string), "stdout content") {
		t.Errorf("expected stdout content in response, got: %v", resp["content"])
	}
	if resp["name"] != "output.md" {
		t.Errorf("expected name=output.md, got %v", resp["name"])
	}
}

func TestRunFile_OutputMdNoFallback(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create a run directory but no files (neither output.md nor agent-stdout.txt)
	runDir := filepath.Join(root, "project", "task", "runs", "run-empty")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-empty",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusCompleted,
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
		// StdoutPath intentionally left empty — no files created
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/run-empty/file?name=output.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when neither output.md nor stdout exists, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStopRun_Success(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create a running run with a non-existent PID/PGID (best-effort SIGTERM will fail, but 202 still returned).
	runDir := filepath.Join(root, "project", "task", "runs", "run-stop-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	_ = os.WriteFile(stdoutPath, []byte(""), 0o644)
	stopInfo := &storage.RunInfo{
		RunID:      "run-stop-1",
		ProjectID:  "project",
		TaskID:     "task",
		Status:     storage.StatusRunning,
		StartTime:  time.Now().UTC(),
		StdoutPath: stdoutPath,
		PID:        99999999,
		PGID:       99999999,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), stopInfo); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/run-stop-1/stop"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "SIGTERM sent") {
		t.Errorf("expected 'SIGTERM sent' in response, got: %q", rec.Body.String())
	}
}

func TestStopRun_NotRunning(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	makeProjectRun(t, root, "project", "task", "run-stop-2", storage.StatusCompleted, "hello\n")

	url := "/api/projects/project/tasks/task/runs/run-stop-2/stop"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServeTaskFile_Found(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create TASK.md in the task directory
	taskDir := filepath.Join(root, "project", "task-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	taskContent := "# My Task\n\nDo something great.\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(taskContent), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	url := "/api/projects/project/tasks/task-1/file?name=TASK.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "TASK.md" {
		t.Errorf("expected name=TASK.md, got %v", resp["name"])
	}
	if !strings.Contains(resp["content"].(string), "Do something great") {
		t.Errorf("expected task content in response, got: %v", resp["content"])
	}
}

func TestServeTaskFile_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Task directory exists but no TASK.md
	taskDir := filepath.Join(root, "project", "task-notask")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	url := "/api/projects/project/tasks/task-notask/file?name=TASK.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServeTaskFile_UnknownName(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/project/tasks/task-1/file?name=secrets.txt"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown file name, got %d: %s", rec.Code, rec.Body.String())
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

// writeProjectBus writes a minimal message bus entry to the given path.
func writeProjectBus(t *testing.T, busPath, msgID, msgType, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir bus dir: %v", err)
	}
	entry := "---\nmsg_id: " + msgID + "\ntype: " + msgType + "\nproject_id: test\nts: 2025-01-01T00:00:00Z\n---\n" + body + "\n"
	if err := os.WriteFile(busPath, []byte(entry), 0o644); err != nil {
		t.Fatalf("write bus: %v", err)
	}
}

func TestProjectMessages_ListEmpty(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	msgs, ok := resp["messages"]
	if !ok {
		t.Fatalf("expected 'messages' key in response")
	}
	if msgs == nil {
		t.Errorf("expected non-nil messages slice")
	}
}

func TestProjectMessages_ListWithMessages(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	busPath := filepath.Join(root, "proj1", "PROJECT-MESSAGE-BUS.md")
	writeProjectBus(t, busPath, "msg-001", "USER", "hello world")

	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "hello world") {
		t.Errorf("expected message body in response, got: %q", rec.Body.String())
	}
}

func TestProjectMessages_Post(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := strings.NewReader(`{"type":"USER","body":"test message"}`)
	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["msg_id"] == "" || resp["msg_id"] == nil {
		t.Errorf("expected msg_id in response, got: %v", resp)
	}
	// Verify the bus file was created.
	busPath := filepath.Join(root, "proj1", "PROJECT-MESSAGE-BUS.md")
	if _, statErr := os.Stat(busPath); os.IsNotExist(statErr) {
		t.Errorf("expected bus file to be created at %s", busPath)
	}
}

func TestProjectMessages_PostEmptyBody(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := strings.NewReader(`{"type":"USER","body":""}`)
	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectMessages_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestTaskMessages_ListEmpty(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Need a run so the project/task is recognized
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	url := "/api/projects/proj1/tasks/task-a/messages"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["messages"]; !ok {
		t.Fatalf("expected 'messages' key in response")
	}
}

func TestTaskMessages_Post(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	body := strings.NewReader(`{"type":"PROGRESS","body":"task progress"}`)
	url := "/api/projects/proj1/tasks/task-a/messages"
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["msg_id"] == "" || resp["msg_id"] == nil {
		t.Errorf("expected msg_id in response, got: %v", resp)
	}
	busPath := filepath.Join(root, "proj1", "task-a", "TASK-MESSAGE-BUS.md")
	if _, statErr := os.Stat(busPath); os.IsNotExist(statErr) {
		t.Errorf("expected task bus file to be created at %s", busPath)
	}
}

func TestProjectMessages_StreamNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// /messages/stream should return SSE headers even for non-existent bus
	ctx, cancel := context.WithCancel(context.Background())
	url := "/api/projects/proj1/messages/stream"
	req := httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	// Cancel context quickly; we just want to verify it starts streaming.
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("stream handler did not exit after context cancel")
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
}

func TestTaskRunsStream_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "hello\n")

	url := "/api/projects/project/tasks/task/runs/stream"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestTaskRunsStream_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/nonexistent/tasks/nonexistent/runs/stream"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown project/task, got %d", rec.Code)
	}
}

func TestTaskMessages_StreamMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	url := "/api/projects/proj1/tasks/task-a/messages/stream"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
