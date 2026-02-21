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

func TestProjectStats_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/nonexistent/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent project, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectStats_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create the project directory so it's found
	if err := os.MkdirAll(filepath.Join(root, "myproject"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	url := "/api/projects/myproject/stats"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectStats_WithTasksAndRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "stats-project"

	// Create 2 tasks with runs of different statuses
	makeProjectRun(t, root, projectID, "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, "done")
	makeProjectRun(t, root, projectID, "task-20260101-120000-aaa", "run-2", storage.StatusFailed, "fail")
	makeProjectRun(t, root, projectID, "task-20260101-130000-bbb", "run-1", storage.StatusRunning, "go")

	// Write a task message bus file
	busPath := filepath.Join(root, projectID, "task-20260101-120000-aaa", "TASK-MESSAGE-BUS.md")
	if err := os.WriteFile(busPath, []byte("bus content here"), 0o644); err != nil {
		t.Fatalf("write bus: %v", err)
	}

	// Write a project-level message bus file
	projBusPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	if err := os.WriteFile(projBusPath, []byte("project bus"), 0o644); err != nil {
		t.Fatalf("write project bus: %v", err)
	}

	url := "/api/projects/" + projectID + "/stats"
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

	checkInt := func(key string, want int) {
		t.Helper()
		val, ok := resp[key]
		if !ok {
			t.Errorf("missing key %q", key)
			return
		}
		got := int(val.(float64))
		if got != want {
			t.Errorf("key %q: got %d, want %d", key, got, want)
		}
	}

	if resp["project_id"] != projectID {
		t.Errorf("expected project_id=%q, got %v", projectID, resp["project_id"])
	}
	checkInt("total_tasks", 2)
	checkInt("total_runs", 3)
	checkInt("running_runs", 1)
	checkInt("completed_runs", 1)
	checkInt("failed_runs", 1)
	checkInt("crashed_runs", 0)
	checkInt("message_bus_files", 2)
}

func TestProjectStats_EmptyProject(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "empty-project"
	if err := os.MkdirAll(filepath.Join(root, projectID), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	url := "/api/projects/" + projectID + "/stats"
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
	if resp["project_id"] != projectID {
		t.Errorf("expected project_id=%q, got %v", projectID, resp["project_id"])
	}
	for _, key := range []string{"total_tasks", "total_runs", "running_runs", "completed_runs", "failed_runs", "crashed_runs", "message_bus_files"} {
		if val, ok := resp[key]; !ok || int(val.(float64)) != 0 {
			t.Errorf("expected %q=0, got %v", key, val)
		}
	}
}

func TestProjectStats_NonTaskDirsNotCounted(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "mixed-project"

	// Create a real task with a run
	makeProjectRun(t, root, projectID, "task-20260101-120000-real", "run-1", storage.StatusCompleted, "ok")

	// Create a directory that does NOT match the task ID format
	if err := os.MkdirAll(filepath.Join(root, projectID, "notask"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	url := "/api/projects/" + projectID + "/stats"
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
	// Only the valid task ID should be counted
	if got := int(resp["total_tasks"].(float64)); got != 1 {
		t.Errorf("expected total_tasks=1, got %d", got)
	}
	// But runs under "notask" should still be counted
	if got := int(resp["total_runs"].(float64)); got != 1 {
		t.Errorf("expected total_runs=1, got %d", got)
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

func TestRunInfoToProjectRun_AgentVersionAndErrorSummary(t *testing.T) {
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:        "run-1",
		ProjectID:    "project",
		TaskID:       "task",
		AgentType:    "claude",
		AgentVersion: "2.1.49 (Claude Code)",
		Status:       storage.StatusFailed,
		ExitCode:     1,
		StartTime:    now,
		EndTime:      now.Add(time.Minute),
		ErrorSummary: "agent reported failure",
	}
	r := runInfoToProjectRun(info)
	if r.AgentVersion != "2.1.49 (Claude Code)" {
		t.Errorf("expected AgentVersion=%q, got %q", "2.1.49 (Claude Code)", r.AgentVersion)
	}
	if r.ErrorSummary != "agent reported failure" {
		t.Errorf("expected ErrorSummary=%q, got %q", "agent reported failure", r.ErrorSummary)
	}
}

func TestRunInfoToProjectRun_EmptyOptionalFields(t *testing.T) {
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:     "run-2",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Status:    storage.StatusCompleted,
		ExitCode:  0,
		StartTime: now,
		EndTime:   now.Add(time.Minute),
	}
	r := runInfoToProjectRun(info)
	if r.AgentVersion != "" {
		t.Errorf("expected empty AgentVersion, got %q", r.AgentVersion)
	}
	if r.ErrorSummary != "" {
		t.Errorf("expected empty ErrorSummary, got %q", r.ErrorSummary)
	}
}

func TestProjectRunAPI_AgentVersionAndErrorSummary(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-versioned")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:        "run-versioned",
		ProjectID:    "project",
		TaskID:       "task",
		AgentType:    "claude",
		AgentVersion: "3.0.0 (Claude Code)",
		Status:       storage.StatusFailed,
		ExitCode:     1,
		StartTime:    now,
		EndTime:      now.Add(time.Minute),
		ErrorSummary: "agent reported failure",
		StdoutPath:   filepath.Join(runDir, "agent-stdout.txt"),
	}
	if err := os.WriteFile(info.StdoutPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/run-versioned"
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
	if resp["agent_version"] != "3.0.0 (Claude Code)" {
		t.Errorf("expected agent_version=%q, got %v", "3.0.0 (Claude Code)", resp["agent_version"])
	}
	if resp["error_summary"] != "agent reported failure" {
		t.Errorf("expected error_summary=%q, got %v", "agent reported failure", resp["error_summary"])
	}
}

// makeRunsSubdirRun creates a run at <root>/runs/<projectID>/<taskID>/runs/<runID>/
// to simulate the common layout when the server is started with the project's parent as root.
func makeRunsSubdirRun(t *testing.T, root, projectID, taskID, runID, status string) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, "runs", projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte("output"), 0o644); err != nil {
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

func TestServeTaskFile_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create task directory under runs/ subdirectory
	taskDir := filepath.Join(root, "runs", "project", "task-run-sub")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	taskContent := "# Sub-dir Task\n\nDo the work.\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(taskContent), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	url := "/api/projects/project/tasks/task-run-sub/file?name=TASK.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for task in runs/ subdir, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(resp["content"].(string), "Do the work") {
		t.Errorf("expected task content in response, got: %v", resp["content"])
	}
}

func TestProjectStats_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "runs-sub-project"

	// Create runs under <root>/runs/<projectID>/<taskID>/runs/<runID>/
	makeRunsSubdirRun(t, root, projectID, "task-20260101-120000-aaa", "run-1", storage.StatusCompleted)
	makeRunsSubdirRun(t, root, projectID, "task-20260101-130000-bbb", "run-1", storage.StatusRunning)

	url := "/api/projects/" + projectID + "/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for project in runs/ subdir, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["project_id"] != projectID {
		t.Errorf("expected project_id=%q, got %v", projectID, resp["project_id"])
	}
	if got := int(resp["total_tasks"].(float64)); got != 2 {
		t.Errorf("expected total_tasks=2, got %d", got)
	}
	if got := int(resp["total_runs"].(float64)); got != 2 {
		t.Errorf("expected total_runs=2, got %d", got)
	}
}

func TestFindProjectTaskDir_DirectPath(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "myproject", "task-20260101-120000-abc")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectTaskDir(root, "myproject", "task-20260101-120000-abc")
	if !ok {
		t.Fatalf("expected to find task dir, not found")
	}
	if found != taskDir {
		t.Errorf("expected %q, got %q", taskDir, found)
	}
}

func TestFindProjectTaskDir_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "runs", "myproject", "task-20260101-120000-abc")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectTaskDir(root, "myproject", "task-20260101-120000-abc")
	if !ok {
		t.Fatalf("expected to find task dir under runs/, not found")
	}
	if found != taskDir {
		t.Errorf("expected %q, got %q", taskDir, found)
	}
}

func TestFindProjectTaskDir_NotFound(t *testing.T) {
	root := t.TempDir()
	_, ok := findProjectTaskDir(root, "noproject", "task-20260101-120000-abc")
	if ok {
		t.Errorf("expected not found for non-existent task dir")
	}
}

func TestFindProjectDir_DirectPath(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectDir(root, "myproject")
	if !ok {
		t.Fatalf("expected to find project dir, not found")
	}
	if found != projectDir {
		t.Errorf("expected %q, got %q", projectDir, found)
	}
}

func TestFindProjectDir_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "runs", "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectDir(root, "myproject")
	if !ok {
		t.Fatalf("expected to find project dir under runs/, not found")
	}
	if found != projectDir {
		t.Errorf("expected %q, got %q", projectDir, found)
	}
}

func TestFindProjectDir_NotFound(t *testing.T) {
	root := t.TempDir()
	_, ok := findProjectDir(root, "noproject")
	if ok {
		t.Errorf("expected not found for non-existent project dir")
	}
}

// makeProjectRunAt creates a run with a controlled start time for pagination ordering tests.
func makeProjectRunAt(t *testing.T, root, projectID, taskID, runID, status string, startTime time.Time) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte("output"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:      runID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Status:     status,
		StartTime:  startTime,
		StdoutPath: stdoutPath,
	}
	if status != storage.StatusRunning {
		info.EndTime = startTime.Add(time.Minute)
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return info
}

func TestProjectTasksPagination_Default(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-20260101-140000-ccc", "run-1", storage.StatusCompleted, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["total"] == nil {
		t.Fatalf("expected 'total' field in paginated response")
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 50 {
		t.Errorf("expected limit=50, got %v", resp["limit"])
	}
	if int(resp["offset"].(float64)) != 0 {
		t.Errorf("expected offset=0, got %v", resp["offset"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false for 3 items with limit=50")
	}
	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatalf("expected 'items' array, got %T", resp["items"])
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestProjectTasksPagination_LimitOffset(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-20260101-140000-ccc", "run-1", storage.StatusCompleted, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks?limit=2&offset=0", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 2 {
		t.Errorf("expected limit=2, got %v", resp["limit"])
	}
	if !resp["has_more"].(bool) {
		t.Errorf("expected has_more=true")
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestProjectTasksPagination_OffsetBeyondTotal(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks?limit=50&offset=10", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 1 {
		t.Errorf("expected total=1, got %v", resp["total"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false")
	}
	items := resp["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items for offset beyond total, got %d", len(items))
	}
}

func TestProjectTasksPagination_LimitClamped(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks?limit=9999", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["limit"].(float64)) != 500 {
		t.Errorf("expected limit clamped to 500, got %v", resp["limit"])
	}
}

func TestProjectTaskRunsPagination_Default(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-a", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-a", "run-2", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-a", "run-3", storage.StatusRunning, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-a/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 50 {
		t.Errorf("expected limit=50, got %v", resp["limit"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false")
	}
	items := resp["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestProjectTaskRunsPagination_LimitOffset(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-b", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-b", "run-2", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-b", "run-3", storage.StatusCompleted, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-b/runs?limit=2&offset=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 2 {
		t.Errorf("expected limit=2, got %v", resp["limit"])
	}
	if int(resp["offset"].(float64)) != 1 {
		t.Errorf("expected offset=1, got %v", resp["offset"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false for offset=1 limit=2 total=3")
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestProjectTaskRunsPagination_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-nonexistent/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent task, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectTaskRunsPagination_SortNewestFirst(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Create runs in "wrong" order - oldest first
	makeProjectRunAt(t, root, "proj", "task-c", "run-old", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-c", "run-new", storage.StatusCompleted, base.Add(2*time.Hour))
	makeProjectRunAt(t, root, "proj", "task-c", "run-mid", storage.StatusCompleted, base.Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-c/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// First item should be newest (run-new)
	first := items[0].(map[string]interface{})
	if first["id"] != "run-new" {
		t.Errorf("expected first item to be run-new (newest), got %v", first["id"])
	}
	// Last item should be oldest (run-old)
	last := items[2].(map[string]interface{})
	if last["id"] != "run-old" {
		t.Errorf("expected last item to be run-old (oldest), got %v", last["id"])
	}
}

func TestDeleteTask_Success(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task-del", "run-1", storage.StatusCompleted, "output\n")
	makeProjectRun(t, root, "project", "task-del", "run-2", storage.StatusFailed, "fail\n")

	taskDir := filepath.Join(root, "project", "task-del")
	if _, statErr := os.Stat(taskDir); os.IsNotExist(statErr) {
		t.Fatalf("task directory should exist before delete")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-del", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, statErr := os.Stat(taskDir); !os.IsNotExist(statErr) {
		t.Errorf("expected task directory to be deleted, but it still exists at %s", taskDir)
	}
}

func TestDeleteTask_RunningConflict(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task-running", "run-1", storage.StatusCompleted, "done\n")
	makeProjectRun(t, root, "project", "task-running", "run-2", storage.StatusRunning, "go\n")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-running", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for task with running runs, got %d: %s", rec.Code, rec.Body.String())
	}

	// Task directory should still exist.
	taskDir := filepath.Join(root, "project", "task-running")
	if _, statErr := os.Stat(taskDir); os.IsNotExist(statErr) {
		t.Errorf("task directory should NOT be deleted when a run is still running")
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-nonexistent", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent task, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteRun_Success(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-del-1", storage.StatusCompleted, "output\n")

	runDir := filepath.Join(root, "project", "task", "runs", "run-del-1")
	if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
		t.Fatalf("run directory should exist before delete")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-del-1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, statErr := os.Stat(runDir); !os.IsNotExist(statErr) {
		t.Errorf("expected run directory to be deleted, but it still exists at %s", runDir)
	}
}

func TestDeleteRun_Running(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-del-2", storage.StatusRunning, "output\n")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-del-2", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for running run, got %d: %s", rec.Code, rec.Body.String())
	}

	// Run directory should still exist.
	runDir := filepath.Join(root, "project", "task", "runs", "run-del-2")
	if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
		t.Errorf("run directory should NOT be deleted for a running run")
	}
}

func TestDeleteRun_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-nonexistent", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskResume_WithDONE(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create task dir with TASK.md and a DONE file.
	taskDir := filepath.Join(root, "project", "task-resume")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/task-resume/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["project_id"] != "project" {
		t.Fatalf("expected project_id=project, got %v", result["project_id"])
	}
	if result["task_id"] != "task-resume" {
		t.Fatalf("expected task_id=task-resume, got %v", result["task_id"])
	}
	if result["resumed"] != true {
		t.Fatalf("expected resumed=true, got %v", result["resumed"])
	}

	// DONE file must be removed.
	if _, err := os.Stat(filepath.Join(taskDir, "DONE")); !os.IsNotExist(err) {
		t.Fatalf("expected DONE file to be removed")
	}
}

func TestHandleTaskResume_NoDONE(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create task dir with TASK.md but no DONE file.
	taskDir := filepath.Join(root, "project", "task-nodone")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/task-nodone/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskResume_TaskNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/nonexistent-task/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskResume_WrongMethod(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks/task-x/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectGC_DryRun(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc", "task-gc", "run-1", storage.StatusCompleted, past)
	makeProjectRunAt(t, root, "proj-gc", "task-gc", "run-2", storage.StatusFailed, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc/gc?older_than=1h&dry_run=true", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", resp["dry_run"])
	}
	if int(resp["deleted_runs"].(float64)) != 2 {
		t.Errorf("expected deleted_runs=2, got %v", resp["deleted_runs"])
	}
	// Run directories should still exist (dry run).
	for _, runID := range []string{"run-1", "run-2"} {
		runDir := filepath.Join(root, "proj-gc", "task-gc", "runs", runID)
		if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
			t.Errorf("run %s should not be deleted in dry-run mode", runID)
		}
	}
}

func TestProjectGC_DeletesOldRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc2", "task-gc2", "run-1", storage.StatusCompleted, past)
	makeProjectRunAt(t, root, "proj-gc2", "task-gc2", "run-2", storage.StatusFailed, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc2/gc?older_than=1h&dry_run=false", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["dry_run"] != false {
		t.Errorf("expected dry_run=false, got %v", resp["dry_run"])
	}
	if int(resp["deleted_runs"].(float64)) != 2 {
		t.Errorf("expected deleted_runs=2, got %v", resp["deleted_runs"])
	}
	// Run directories should be deleted.
	for _, runID := range []string{"run-1", "run-2"} {
		runDir := filepath.Join(root, "proj-gc2", "task-gc2", "runs", runID)
		if _, statErr := os.Stat(runDir); !os.IsNotExist(statErr) {
			t.Errorf("run %s should be deleted", runID)
		}
	}
}

func TestProjectGC_SkipsRunning(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc3", "task-gc3", "run-running", storage.StatusRunning, past)
	makeProjectRunAt(t, root, "proj-gc3", "task-gc3", "run-done", storage.StatusCompleted, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc3/gc?older_than=1h", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["deleted_runs"].(float64)) != 1 {
		t.Errorf("expected deleted_runs=1 (only the completed run), got %v", resp["deleted_runs"])
	}
	// Running run should still exist.
	runningDir := filepath.Join(root, "proj-gc3", "task-gc3", "runs", "run-running")
	if _, statErr := os.Stat(runningDir); os.IsNotExist(statErr) {
		t.Errorf("running run should not be deleted")
	}
}

func TestProjectGC_KeepFailed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc4", "task-gc4", "run-completed", storage.StatusCompleted, past)
	makeProjectRunAt(t, root, "proj-gc4", "task-gc4", "run-failed", storage.StatusFailed, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc4/gc?older_than=1h&keep_failed=true", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["deleted_runs"].(float64)) != 1 {
		t.Errorf("expected deleted_runs=1 (only completed), got %v", resp["deleted_runs"])
	}
	// Failed run should still exist.
	failedDir := filepath.Join(root, "proj-gc4", "task-gc4", "runs", "run-failed")
	if _, statErr := os.Stat(failedDir); os.IsNotExist(statErr) {
		t.Errorf("failed run should not be deleted when keep_failed=true")
	}
}

func TestProjectGC_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/gc?older_than=1h", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectGC_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/gc", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectGC_InvalidDuration(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj/gc?older_than=notaduration", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
