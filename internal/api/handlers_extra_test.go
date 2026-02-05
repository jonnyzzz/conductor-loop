package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestHandleRunInfoSuccess(t *testing.T) {
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
		Status:    storage.StatusRunning,
		StartTime: time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-yaml" {
		t.Fatalf("unexpected content-type: %q", ct)
	}
}

func TestHandleRunStopSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group handling differs on windows")
	}
	root := t.TempDir()
	server, err := NewServer(Options{
		RootDir:   root,
		APIConfig: config.APIConfig{},
		Logger:    log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	cmd := exec.Command("sh", "-c", "sleep 5")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	pgid := cmd.Process.Pid
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		StartTime: time.Now().UTC(),
		PID:       pgid,
		PGID:      pgid,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-1/stop", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("process did not exit after stop")
	}
}

func TestHandleTaskGetNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task?project_id=project", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleTaskCancelNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/task?project_id=project", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRunInfoNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/missing/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRunInfoAmbiguous(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	for _, project := range []string{"p1", "p2"} {
		runDir := filepath.Join(root, project, "task", "runs", "run-x")
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("mkdir run: %v", err)
		}
		info := &storage.RunInfo{RunID: "run-x", ProjectID: project, TaskID: "task", Status: storage.StatusCompleted}
		if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
			t.Fatalf("write run-info: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-x/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestHandleRunStopNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/missing/stop", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRunByIDUnknownSegment(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/unknown", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleTaskGetInvalidID(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	apiErr := server.handleTaskGet(rec, req, "project", "bad/one")
	if apiErr == nil || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid task id")
	}
}

func TestHandleTaskCreateInvalidJSON(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString("{bad"))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskCreateMissingFields(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{ProjectID: "project", TaskID: "task"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskCancelInvalidID(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks", nil)
	apiErr := server.handleTaskCancel(rec, req, "project", "bad/one")
	if apiErr == nil || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid task id")
	}
}

func TestListRunResponsesRootEmpty(t *testing.T) {
	if _, err := listRunResponses(""); err == nil {
		t.Fatalf("expected error for empty root")
	}
}

func TestListRunResponsesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	runs, err := listRunResponses(root)
	if err != nil {
		t.Fatalf("listRunResponses: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected no runs, got %d", len(runs))
	}
}

func TestHandleRunGetNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/missing", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRunByIDMissingID(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestStopTaskRunsActive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group handling differs on windows")
	}
	taskDir := t.TempDir()
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	cmd := exec.Command("sh", "-c", "sleep 5")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	pgid := cmd.Process.Pid
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		StartTime: time.Now().UTC(),
		PID:       pgid,
		PGID:      pgid,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	stopped, err := stopTaskRuns(taskDir)
	if err != nil {
		t.Fatalf("stopTaskRuns: %v", err)
	}
	if stopped != 1 {
		t.Fatalf("expected 1 stopped run, got %d", stopped)
	}
}
