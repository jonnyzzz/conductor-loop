package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestValidateIdentifier(t *testing.T) {
	if err := validateIdentifier("", "project_id"); err == nil {
		t.Fatalf("expected error for empty identifier")
	}
	if err := validateIdentifier("bad/one", "project_id"); err == nil {
		t.Fatalf("expected error for path separator")
	}
	if err := validateIdentifier("..", "project_id"); err == nil {
		t.Fatalf("expected error for ..")
	}
	if err := validateIdentifier("ok", "project_id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathSegments(t *testing.T) {
	if parts := pathSegments("/api/v1/tasks", "/api/v1/tasks/"); parts != nil {
		t.Fatalf("expected nil for missing prefix")
	}
	parts := pathSegments("/api/v1/tasks/project/task", "/api/v1/tasks/")
	if len(parts) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(parts))
	}
}

func TestDecodeJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
	if err := decodeJSON(req, &struct{}{}); err == nil {
		t.Fatalf("expected error for invalid json")
	}
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"a":1} {}`))
	if err := decodeJSON(req, &struct{}{}); err == nil {
		t.Fatalf("expected error for trailing data")
	}
}

func TestWriteJSONNilWriter(t *testing.T) {
	if err := writeJSON(nil, http.StatusOK, map[string]string{"ok": "true"}); err == nil {
		t.Fatalf("expected error for nil writer")
	}
}

func TestListTasksAndRuns(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning, StartTime: time.Now().UTC()}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	tasks, err := listTasks(root)
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Status != "running" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	runs, err := listTaskRuns(taskDir)
	if err != nil {
		t.Fatalf("listTaskRuns: %v", err)
	}
	if len(runs) != 1 || runs[0].RunID != "run-1" {
		t.Fatalf("unexpected runs: %+v", runs)
	}
}

func TestFindTaskAmbiguous(t *testing.T) {
	root := t.TempDir()
	for _, project := range []string{"p1", "p2"} {
		taskDir := filepath.Join(root, project, "task")
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
			t.Fatalf("write TASK.md: %v", err)
		}
	}
	if _, err := findTask(root, "task"); err == nil {
		t.Fatalf("expected ambiguous error")
	}
}

func TestFindRunInfoPathAmbiguous(t *testing.T) {
	root := t.TempDir()
	for _, project := range []string{"p1", "p2"} {
		runDir := filepath.Join(root, project, "task", "runs", "run-x")
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		info := &storage.RunInfo{RunID: "run-x", ProjectID: project, TaskID: "task", Status: storage.StatusCompleted}
		if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
			t.Fatalf("write run-info: %v", err)
		}
	}
	if _, err := findRunInfoPath(root, "run-x"); err == nil {
		t.Fatalf("expected ambiguous error")
	}
}

func TestHandleStatus(t *testing.T) {
	root := t.TempDir()
	fixedTime := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	callCount := 0
	fakeClock := func() time.Time {
		callCount++
		if callCount <= 1 {
			return fixedTime
		}
		return fixedTime.Add(42 * time.Second)
	}

	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		AgentNames:       []string{"claude", "codex"},
		Version:          "v1.2.3",
		Now:              fakeClock,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create a running run and a completed run.
	runDir1 := filepath.Join(root, "project", "task", "runs", "run-1")
	runDir2 := filepath.Join(root, "project", "task", "runs", "run-2")
	if err := os.MkdirAll(runDir1, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(runDir2, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	running := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning, StartTime: fixedTime}
	if err := storage.WriteRunInfo(filepath.Join(runDir1, "run-info.yaml"), running); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	completed := &storage.RunInfo{RunID: "run-2", ProjectID: "project", TaskID: "task", Status: storage.StatusCompleted, StartTime: fixedTime, EndTime: fixedTime.Add(10 * time.Second)}
	if err := storage.WriteRunInfo(filepath.Join(runDir2, "run-info.yaml"), completed); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ActiveRunsCount != 1 {
		t.Fatalf("expected 1 active run, got %d", resp.ActiveRunsCount)
	}
	if resp.UptimeSeconds != 42 {
		t.Fatalf("expected 42s uptime, got %f", resp.UptimeSeconds)
	}
	if len(resp.ConfiguredAgents) != 2 || resp.ConfiguredAgents[0] != "claude" || resp.ConfiguredAgents[1] != "codex" {
		t.Fatalf("unexpected agents: %v", resp.ConfiguredAgents)
	}
	if resp.Version != "v1.2.3" {
		t.Fatalf("expected version v1.2.3, got %s", resp.Version)
	}
}

func TestHandleStatusMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleStatusNoAgents(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ActiveRunsCount != 0 {
		t.Fatalf("expected 0 active runs, got %d", resp.ActiveRunsCount)
	}
	if len(resp.ConfiguredAgents) != 0 {
		t.Fatalf("expected empty agents, got %v", resp.ConfiguredAgents)
	}
}

func TestHandleTaskCreate(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, APIConfig: config.APIConfig{}})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{ProjectID: "project", TaskID: "task", AgentType: "codex", Prompt: "hello"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "project", "task", "TASK.md")); err != nil {
		t.Fatalf("expected TASK.md: %v", err)
	}
}
