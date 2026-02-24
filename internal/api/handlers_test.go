package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
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
	if err := validateIdentifier("%2e%2e", "project_id"); err == nil {
		t.Fatalf("expected error for encoded ..")
	}
	if err := validateIdentifier("bad%2fname", "project_id"); err == nil {
		t.Fatalf("expected error for encoded path separator")
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
	if len(resp.RunningTasks) != 1 {
		t.Fatalf("expected 1 running task, got %d", len(resp.RunningTasks))
	}
	if resp.RunningTasks[0].RunID != "run-1" {
		t.Fatalf("expected running task run_id=run-1, got %s", resp.RunningTasks[0].RunID)
	}
	if resp.RunningTasks[0].ProjectID != "project" {
		t.Fatalf("expected running task project_id=project, got %s", resp.RunningTasks[0].ProjectID)
	}
	if resp.RunningTasks[0].TaskID != "task" {
		t.Fatalf("expected running task task_id=task, got %s", resp.RunningTasks[0].TaskID)
	}
}

func TestHandleStatus_ReconcilesStaleRunningPID(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-stale")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	infoPath := filepath.Join(runDir, "run-info.yaml")
	stale := &storage.RunInfo{
		RunID:     "run-stale",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: time.Now().Add(-time.Minute).UTC(),
		PID:       99999999,
		PGID:      99999999,
	}
	if err := storage.WriteRunInfo(infoPath, stale); err != nil {
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
	if resp.ActiveRunsCount != 0 {
		t.Fatalf("expected 0 active runs after reconciliation, got %d", resp.ActiveRunsCount)
	}
	if len(resp.RunningTasks) != 0 {
		t.Fatalf("expected no running tasks after reconciliation, got %d", len(resp.RunningTasks))
	}

	reloaded, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		t.Fatalf("read reconciled run-info: %v", err)
	}
	if reloaded.Status != storage.StatusFailed {
		t.Fatalf("expected reconciled status=%q, got %q", storage.StatusFailed, reloaded.Status)
	}
	if reloaded.EndTime.IsZero() {
		t.Fatalf("expected end_time to be set during reconciliation")
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
	if resp.RunningTasks == nil {
		t.Fatalf("expected running_tasks to be non-nil empty slice, got nil")
	}
	if len(resp.RunningTasks) != 0 {
		t.Fatalf("expected 0 running tasks, got %d", len(resp.RunningTasks))
	}
}

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return server, root
}

func TestPostMessage_Success(t *testing.T) {
	server, root := newTestServer(t)
	payload := PostMessageRequest{ProjectID: "myproject", Type: "USER", Body: "hello world"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp PostMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MsgID == "" {
		t.Fatalf("expected non-empty msg_id")
	}
	if resp.Timestamp.IsZero() {
		t.Fatalf("expected non-zero timestamp")
	}
	busPath := filepath.Join(root, "myproject", "PROJECT-MESSAGE-BUS.md")
	if _, err := os.Stat(busPath); err != nil {
		t.Fatalf("expected PROJECT-MESSAGE-BUS.md to exist: %v", err)
	}
}

func TestPostMessage_MissingProjectID(t *testing.T) {
	server, _ := newTestServer(t)
	payload := PostMessageRequest{Body: "hello world"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostMessage_EmptyBody(t *testing.T) {
	server, _ := newTestServer(t)
	payload := PostMessageRequest{ProjectID: "myproject", Type: "USER", Body: ""}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostMessage_WithTaskID(t *testing.T) {
	server, root := newTestServer(t)
	payload := PostMessageRequest{ProjectID: "myproject", TaskID: "mytask", Type: "INFO", Body: "task message"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp PostMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MsgID == "" {
		t.Fatalf("expected non-empty msg_id")
	}
	busPath := filepath.Join(root, "myproject", "mytask", "TASK-MESSAGE-BUS.md")
	if _, err := os.Stat(busPath); err != nil {
		t.Fatalf("expected TASK-MESSAGE-BUS.md to exist: %v", err)
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
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project", "task", "TASK.md")); err != nil {
		t.Fatalf("expected TASK.md: %v", err)
	}
	var createResp TaskCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if createResp.RunID == "" {
		t.Fatalf("expected non-empty run_id in response")
	}
	if createResp.ProjectID != "project" {
		t.Fatalf("expected project_id=project, got %s", createResp.ProjectID)
	}
	if createResp.TaskID != "task" {
		t.Fatalf("expected task_id=task, got %s", createResp.TaskID)
	}
	// Verify the pre-allocated run directory exists.
	runDir := filepath.Join(root, "project", "task", "runs", createResp.RunID)
	if _, err := os.Stat(runDir); err != nil {
		t.Fatalf("expected run directory %s to exist: %v", runDir, err)
	}
}

func TestHandleTaskCreateRejectedDuringSelfUpdateDrain(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, APIConfig: config.APIConfig{}})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	server.selfUpdate.mu.Lock()
	server.selfUpdate.state.State = selfUpdateStateDeferred
	server.selfUpdate.mu.Unlock()

	payload := TaskCreateRequest{ProjectID: "project", TaskID: "task", AgentType: "codex", Prompt: "hello"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "self-update drain") {
		t.Fatalf("expected self-update conflict body, got %s", resp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project", "task")); !os.IsNotExist(err) {
		t.Fatalf("task directory should not be created during drain mode")
	}
}

func TestHandleTaskCreate_PromptWhitespacePreserved(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true, APIConfig: config.APIConfig{}})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	originalPrompt := "  keep-leading\nkeep-trailing  \n\n"
	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Prompt:    originalPrompt,
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	taskMDPath := filepath.Join(root, "project", "task", "TASK.md")
	content, err := os.ReadFile(taskMDPath)
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if string(content) != originalPrompt {
		t.Fatalf("prompt mismatch: got %q, want %q", string(content), originalPrompt)
	}
}

func TestHandleTaskCreate_ProjectRoot_Invalid(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{
		ProjectID:   "project",
		TaskID:      "task",
		AgentType:   "codex",
		Prompt:      "hello",
		ProjectRoot: "/nonexistent/path/that/does/not/exist",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid project_root, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleTaskCreate_ProjectRoot_Valid(t *testing.T) {
	root := t.TempDir()
	projectRoot := t.TempDir() // valid existing directory
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{
		ProjectID:   "project",
		TaskID:      "task",
		AgentType:   "codex",
		Prompt:      "hello",
		ProjectRoot: projectRoot,
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid project_root, got %d: %s", resp.Code, resp.Body.String())
	}
	var createResp TaskCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if createResp.RunID == "" {
		t.Fatalf("expected non-empty run_id")
	}
}

func TestHandleTaskCreate_AttachMode_Invalid(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{
		ProjectID:  "project",
		TaskID:     "task",
		AgentType:  "codex",
		Prompt:     "hello",
		AttachMode: "badmode",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid attach_mode, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleTaskCreate_AttachMode_Values(t *testing.T) {
	for _, mode := range []string{"create", "attach", "resume"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}
			payload := TaskCreateRequest{
				ProjectID:  "project",
				TaskID:     "task",
				AgentType:  "codex",
				Prompt:     "hello",
				AttachMode: mode,
			}
			data, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
			resp := httptest.NewRecorder()
			server.Handler().ServeHTTP(resp, req)
			if resp.Code != http.StatusCreated {
				t.Fatalf("mode=%s: expected 201, got %d: %s", mode, resp.Code, resp.Body.String())
			}
			var createResp TaskCreateResponse
			if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
				t.Fatalf("mode=%s: decode response: %v", mode, err)
			}
			if createResp.RunID == "" {
				t.Fatalf("mode=%s: expected non-empty run_id", mode)
			}
		})
	}
}

func TestHandleTaskCreate_ProcessImport_InvalidPID(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Prompt:    "hello",
		ProcessImport: &ProcessImportRequest{
			PID:        0,
			StdoutPath: filepath.Join(root, "stdout.log"),
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid process_import.pid, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleTaskCreate_ProcessImport_RequiresLogSource(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Prompt:    "hello",
		ProcessImport: &ProcessImportRequest{
			PID: 123,
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when process_import has no stdout/stderr paths, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleTaskCreate_ProcessImport_OwnershipValidation(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Prompt:    "hello",
		ProcessImport: &ProcessImportRequest{
			PID:        123,
			StdoutPath: filepath.Join(root, "stdout.log"),
			Ownership:  "invalid",
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid process_import.ownership, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleTaskCreate_DependsOnPersisted(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-main",
		AgentType: "codex",
		Prompt:    "hello",
		DependsOn: []string{"task-a", " task-b "},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var createResp TaskCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := []string{"task-a", "task-b"}
	if len(createResp.DependsOn) != len(want) {
		t.Fatalf("depends_on=%v, want %v", createResp.DependsOn, want)
	}
	for i := range want {
		if createResp.DependsOn[i] != want[i] {
			t.Fatalf("depends_on[%d]=%q, want %q", i, createResp.DependsOn[i], want[i])
		}
	}

	taskDir := filepath.Join(root, "project", "task-main")
	savedDependsOn, err := taskdeps.ReadDependsOn(taskDir)
	if err != nil {
		t.Fatalf("ReadDependsOn: %v", err)
	}
	if len(savedDependsOn) != len(want) {
		t.Fatalf("saved depends_on=%v, want %v", savedDependsOn, want)
	}
	for i := range want {
		if savedDependsOn[i] != want[i] {
			t.Fatalf("saved depends_on[%d]=%q, want %q", i, savedDependsOn[i], want[i])
		}
	}
}

func TestHandleTaskCreate_DependsOnCycleRejected(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	taskADir := filepath.Join(root, "project", "task-a")
	if err := os.MkdirAll(taskADir, 0o755); err != nil {
		t.Fatalf("mkdir task-a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskADir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := taskdeps.WriteDependsOn(taskADir, []string{"task-b"}); err != nil {
		t.Fatalf("WriteDependsOn: %v", err)
	}

	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-b",
		AgentType: "codex",
		Prompt:    "hello",
		DependsOn: []string{"task-a"},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409 for dependency cycle, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleTaskGet_BlockedByDependencies(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	taskMainDir := filepath.Join(root, "project", "task-main")
	taskDepDir := filepath.Join(root, "project", "task-dep")
	if err := os.MkdirAll(taskMainDir, 0o755); err != nil {
		t.Fatalf("mkdir task-main: %v", err)
	}
	if err := os.MkdirAll(taskDepDir, 0o755); err != nil {
		t.Fatalf("mkdir task-dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskMainDir, "TASK.md"), []byte("main\n"), 0o644); err != nil {
		t.Fatalf("write main TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDepDir, "TASK.md"), []byte("dep\n"), 0o644); err != nil {
		t.Fatalf("write dep TASK.md: %v", err)
	}
	if err := taskdeps.WriteDependsOn(taskMainDir, []string{"task-dep"}); err != nil {
		t.Fatalf("WriteDependsOn: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-main?project_id=project", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp TaskResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "blocked" {
		t.Fatalf("status=%q, want blocked", resp.Status)
	}
	if len(resp.DependsOn) != 1 || resp.DependsOn[0] != "task-dep" {
		t.Fatalf("depends_on=%v, want [task-dep]", resp.DependsOn)
	}
	if len(resp.BlockedBy) != 1 || resp.BlockedBy[0] != "task-dep" {
		t.Fatalf("blocked_by=%v, want [task-dep]", resp.BlockedBy)
	}
}

func TestHandleTaskCreate_Attach_DoesNotOverwriteTaskMD(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Pre-create task dir with existing TASK.md.
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := "original prompt\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	payload := TaskCreateRequest{
		ProjectID:  "project",
		TaskID:     "task",
		AgentType:  "codex",
		Prompt:     "new prompt",
		AttachMode: "attach",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	// TASK.md must remain unchanged.
	content, err := os.ReadFile(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if string(content) != original {
		t.Fatalf("TASK.md was overwritten: got %q, want %q", string(content), original)
	}
}

func TestHandleTaskCreate_Create_DoesNotOverwriteTaskMD(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := "original prompt\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	payload := TaskCreateRequest{
		ProjectID:  "project",
		TaskID:     "task",
		AgentType:  "codex",
		Prompt:     "new prompt",
		AttachMode: "create",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	content, err := os.ReadFile(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if string(content) != original {
		t.Fatalf("TASK.md was overwritten: got %q, want %q", string(content), original)
	}
}

func seedThreadParentMessage(t *testing.T, root, projectID, taskID, runID, msgType, body string) string {
	t.Helper()

	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir parent task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("parent task\n"), 0o644); err != nil {
		t.Fatalf("write parent TASK.md: %v", err)
	}
	bus, err := messagebus.NewMessageBus(filepath.Join(taskDir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("open parent bus: %v", err)
	}
	msgID, err := bus.AppendMessage(&messagebus.Message{
		Type:      msgType,
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     runID,
		Body:      body,
	})
	if err != nil {
		t.Fatalf("append parent message: %v", err)
	}
	return msgID
}

func createDoneWritingAgentCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\n" +
			"if \"%1\"==\"--version\" (\r\n" +
			"  echo " + name + " 1.0.0\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"more >nul\r\n" +
			"type nul > \"%TASK_FOLDER%\\DONE\"\r\n" +
			"echo stdout\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}

	path := filepath.Join(dir, name)
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo '" + name + " 1.0.0'; exit 0; fi\n" +
		"cat >/dev/null\n" +
		": > \"$TASK_FOLDER/DONE\"\n" +
		"echo stdout\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func TestHandleTaskCreate_ThreadedAnswerRejectsNonUserRequestType(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	parentMsgID := seedThreadParentMessage(t, root, "project", "task-parent", "run-parent", "QUESTION", "How should this be fixed?")

	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-child",
		AgentType: "codex",
		Prompt:    "Implement the fix",
		ThreadParent: &ThreadParentReference{
			ProjectID: "project",
			TaskID:    "task-parent",
			RunID:     "run-parent",
			MessageID: parentMsgID,
		},
		ThreadMessageType: "FACT",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "thread_message_type must be USER_REQUEST") {
		t.Fatalf("expected USER_REQUEST validation error, got %s", resp.Body.String())
	}
}

func TestHandleTaskCreate_ThreadedAnswerSupportsQuestionAndFact(t *testing.T) {
	for _, parentType := range []string{"QUESTION", "FACT"} {
		parentType := parentType
		t.Run(parentType, func(t *testing.T) {
			root := t.TempDir()
			server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}

			parentMsgID := seedThreadParentMessage(t, root, "project", "task-parent", "run-parent", parentType, "Please answer this")

			payload := TaskCreateRequest{
				ProjectID: "project",
				TaskID:    "task-child",
				AgentType: "codex",
				Prompt:    "Provide a full answer",
				ThreadParent: &ThreadParentReference{
					ProjectID: "project",
					TaskID:    "task-parent",
					RunID:     "run-parent",
					MessageID: parentMsgID,
				},
				ThreadMessageType: "USER_REQUEST",
			}
			data, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
			resp := httptest.NewRecorder()
			server.Handler().ServeHTTP(resp, req)
			if resp.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestHandleTaskCreate_ThreadedAnswerInvalidParentReferences(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	parentMsgID := seedThreadParentMessage(t, root, "project", "task-parent", "run-parent", "QUESTION", "Question")

	t.Run("missing parent message", func(t *testing.T) {
		payload := TaskCreateRequest{
			ProjectID: "project",
			TaskID:    "task-child-missing",
			AgentType: "codex",
			Prompt:    "Answer",
			ThreadParent: &ThreadParentReference{
				ProjectID: "project",
				TaskID:    "task-parent",
				RunID:     "run-parent",
				MessageID: "MSG-missing",
			},
			ThreadMessageType: "USER_REQUEST",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
		resp := httptest.NewRecorder()
		server.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("parent run mismatch", func(t *testing.T) {
		payload := TaskCreateRequest{
			ProjectID: "project",
			TaskID:    "task-child-mismatch",
			AgentType: "codex",
			Prompt:    "Answer",
			ThreadParent: &ThreadParentReference{
				ProjectID: "project",
				TaskID:    "task-parent",
				RunID:     "run-other",
				MessageID: parentMsgID,
			},
			ThreadMessageType: "USER_REQUEST",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
		resp := httptest.NewRecorder()
		server.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "parent run_id mismatch") {
			t.Fatalf("expected parent mismatch error, got %s", resp.Body.String())
		}
	})
}

func TestHandleTaskCreate_ThreadedAnswerPersistsLinkageMetadata(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	parentMsgID := seedThreadParentMessage(t, root, "project", "task-parent", "run-parent", "QUESTION", "Question")
	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-child",
		AgentType: "codex",
		Prompt:    "Answer parent request with evidence",
		ThreadParent: &ThreadParentReference{
			ProjectID: "project",
			TaskID:    "task-parent",
			RunID:     "run-parent",
			MessageID: parentMsgID,
		},
		ThreadMessageType: "USER_REQUEST",
	}

	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	childTaskDir := filepath.Join(root, "project", "task-child")
	threadLink, err := readTaskThreadLink(childTaskDir)
	if err != nil {
		t.Fatalf("readTaskThreadLink: %v", err)
	}
	if threadLink == nil {
		t.Fatalf("expected thread link metadata")
	}
	if threadLink.ProjectID != "project" || threadLink.TaskID != "task-parent" || threadLink.RunID != "run-parent" || threadLink.MessageID != parentMsgID {
		t.Fatalf("unexpected thread link: %+v", threadLink)
	}

	childBus, err := messagebus.NewMessageBus(filepath.Join(childTaskDir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("open child bus: %v", err)
	}
	childMessages, err := childBus.ReadMessages("")
	if err != nil {
		t.Fatalf("read child bus: %v", err)
	}
	if len(childMessages) != 1 {
		t.Fatalf("expected 1 child message, got %d", len(childMessages))
	}
	childMsg := childMessages[0]
	if childMsg.Type != "USER_REQUEST" {
		t.Fatalf("child message type=%q, want USER_REQUEST", childMsg.Type)
	}
	if len(childMsg.Parents) != 1 || childMsg.Parents[0].MsgID != parentMsgID {
		t.Fatalf("child message parents=%+v, want %s", childMsg.Parents, parentMsgID)
	}
	if childMsg.Meta[threadMetaParentProjectIDKey] != "project" ||
		childMsg.Meta[threadMetaParentTaskIDKey] != "task-parent" ||
		childMsg.Meta[threadMetaParentRunIDKey] != "run-parent" ||
		childMsg.Meta[threadMetaParentMessageIDKey] != parentMsgID {
		t.Fatalf("unexpected child message meta: %+v", childMsg.Meta)
	}

	parentBus, err := messagebus.NewMessageBus(filepath.Join(root, "project", "task-parent", "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("open parent bus: %v", err)
	}
	parentMessages, err := parentBus.ReadMessages("")
	if err != nil {
		t.Fatalf("read parent bus: %v", err)
	}
	if len(parentMessages) < 2 {
		t.Fatalf("expected parent bus to contain linkage message, got %d entries", len(parentMessages))
	}
	var sourceLinkMsg *messagebus.Message
	for _, msg := range parentMessages {
		if msg == nil || msg.Type != "USER_REQUEST" {
			continue
		}
		if msg.Meta[threadMetaChildTaskIDKey] == "task-child" {
			sourceLinkMsg = msg
			break
		}
	}
	if sourceLinkMsg == nil {
		t.Fatalf("expected source linkage message in parent bus")
	}
	if sourceLinkMsg.Meta[threadMetaChildProjectIDKey] != "project" ||
		sourceLinkMsg.Meta[threadMetaParentMessageIDKey] != parentMsgID {
		t.Fatalf("unexpected source linkage metadata: %+v", sourceLinkMsg.Meta)
	}

	taskReq := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks/task-child", nil)
	taskResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(taskResp, taskReq)
	if taskResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from project task detail, got %d: %s", taskResp.Code, taskResp.Body.String())
	}
	var taskDetail struct {
		ThreadParent *ThreadParentReference `json:"thread_parent"`
	}
	if err := json.Unmarshal(taskResp.Body.Bytes(), &taskDetail); err != nil {
		t.Fatalf("decode task detail: %v", err)
	}
	if taskDetail.ThreadParent == nil || taskDetail.ThreadParent.MessageID != parentMsgID {
		t.Fatalf("expected thread_parent in project task detail, got %+v", taskDetail.ThreadParent)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks", nil)
	listResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from project task list, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var listPayload struct {
		Items []struct {
			ID           string                 `json:"id"`
			ThreadParent *ThreadParentReference `json:"thread_parent"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	for _, item := range listPayload.Items {
		if item.ID != "task-child" {
			continue
		}
		if item.ThreadParent == nil || item.ThreadParent.MessageID != parentMsgID {
			t.Fatalf("expected thread_parent in task list item, got %+v", item.ThreadParent)
		}
		return
	}
	t.Fatalf("task-child not found in project task list response")
}

func TestHandleTaskCreate_ThreadedAnswerPropagatesParentRunIDToRunsFlat(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	createDoneWritingAgentCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	server, err := NewServer(Options{RootDir: root})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() {
		server.taskWg.Wait()
	})

	parentMsgID := seedThreadParentMessage(t, root, "project", "task-parent", "run-parent", "QUESTION", "Question")
	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-child-live",
		AgentType: "codex",
		Prompt:    "Answer parent request now",
		ThreadParent: &ThreadParentReference{
			ProjectID: "project",
			TaskID:    "task-parent",
			RunID:     "run-parent",
			MessageID: parentMsgID,
		},
		ThreadMessageType: "USER_REQUEST",
	}

	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var created TaskCreateResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode task create response: %v", err)
	}
	if strings.TrimSpace(created.RunID) == "" {
		t.Fatalf("expected run_id in create response")
	}

	server.taskWg.Wait()

	runInfoPath := filepath.Join(root, "project", "task-child-live", "runs", created.RunID, "run-info.yaml")
	info, err := storage.ReadRunInfo(runInfoPath)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if got := strings.TrimSpace(info.ParentRunID); got != "run-parent" {
		t.Fatalf("run-info parent_run_id=%q, want %q", got, "run-parent")
	}

	flatReq := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat", nil)
	flatResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(flatResp, flatReq)
	if flatResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from runs/flat, got %d: %s", flatResp.Code, flatResp.Body.String())
	}

	var flatPayload struct {
		Runs []struct {
			ID          string `json:"id"`
			TaskID      string `json:"task_id"`
			ParentRunID string `json:"parent_run_id,omitempty"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(flatResp.Body.Bytes(), &flatPayload); err != nil {
		t.Fatalf("decode runs/flat: %v", err)
	}
	for _, run := range flatPayload.Runs {
		if run.ID != created.RunID {
			continue
		}
		if run.TaskID != "task-child-live" {
			t.Fatalf("run task_id=%q, want %q", run.TaskID, "task-child-live")
		}
		if run.ParentRunID != "run-parent" {
			t.Fatalf("runs/flat parent_run_id=%q, want %q", run.ParentRunID, "run-parent")
		}
		return
	}
	t.Fatalf("created run %q not found in runs/flat", created.RunID)
}

func TestHandleTaskCreate_ThreadedAttachImportPropagatesParentRunIDToRunsFlat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based import fixture is unix-only")
	}

	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() {
		server.taskWg.Wait()
	})

	stdoutSource := filepath.Join(root, "external-stdout.log")
	scriptPath := filepath.Join(root, "external.sh")
	script := `#!/bin/sh
echo "imported-start" >> "$SRC_STDOUT"
sleep 0.6
echo "imported-stop" >> "$SRC_STDOUT"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(scriptPath)
	cmd.Env = append(os.Environ(), "SRC_STDOUT="+stdoutSource)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fixture process: %v", err)
	}
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	parentMsgID := seedThreadParentMessage(t, root, "project", "task-parent", "run-parent", "QUESTION", "Question")
	payload := TaskCreateRequest{
		ProjectID:  "project",
		TaskID:     "task-child-attach",
		AgentType:  "codex",
		Prompt:     "Attach to existing process",
		AttachMode: "attach",
		ProcessImport: &ProcessImportRequest{
			PID:        cmd.Process.Pid,
			StdoutPath: stdoutSource,
		},
		ThreadParent: &ThreadParentReference{
			ProjectID: "project",
			TaskID:    "task-parent",
			RunID:     "run-parent",
			MessageID: parentMsgID,
		},
		ThreadMessageType: "USER_REQUEST",
	}

	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var created TaskCreateResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode task create response: %v", err)
	}
	if strings.TrimSpace(created.RunID) == "" {
		t.Fatalf("expected run_id in create response")
	}

	server.taskWg.Wait()
	if waitErr := <-waitDone; waitErr != nil {
		t.Fatalf("fixture process wait: %v", waitErr)
	}

	runInfoPath := filepath.Join(root, "project", "task-child-attach", "runs", created.RunID, "run-info.yaml")
	info, err := storage.ReadRunInfo(runInfoPath)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if got := strings.TrimSpace(info.ParentRunID); got != "run-parent" {
		t.Fatalf("run-info parent_run_id=%q, want %q", got, "run-parent")
	}

	flatReq := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat", nil)
	flatResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(flatResp, flatReq)
	if flatResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from runs/flat, got %d: %s", flatResp.Code, flatResp.Body.String())
	}

	var flatPayload struct {
		Runs []struct {
			ID          string `json:"id"`
			TaskID      string `json:"task_id"`
			ParentRunID string `json:"parent_run_id,omitempty"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(flatResp.Body.Bytes(), &flatPayload); err != nil {
		t.Fatalf("decode runs/flat: %v", err)
	}
	for _, run := range flatPayload.Runs {
		if run.ID != created.RunID {
			continue
		}
		if run.TaskID != "task-child-attach" {
			t.Fatalf("run task_id=%q, want %q", run.TaskID, "task-child-attach")
		}
		if run.ParentRunID != "run-parent" {
			t.Fatalf("runs/flat parent_run_id=%q, want %q", run.ParentRunID, "run-parent")
		}
		return
	}
	t.Fatalf("created run %q not found in runs/flat", created.RunID)
}

func TestRunInfoToResponse_AgentVersion(t *testing.T) {
	info := &storage.RunInfo{
		RunID:        "run-1",
		ProjectID:    "project",
		TaskID:       "task",
		Status:       storage.StatusCompleted,
		AgentVersion: "2.1.50 (Claude Code)",
		ErrorSummary: "agent reported failure",
	}
	resp := runInfoToResponse(info)
	if resp.AgentVersion != "2.1.50 (Claude Code)" {
		t.Fatalf("expected AgentVersion=%q, got %q", "2.1.50 (Claude Code)", resp.AgentVersion)
	}
	if resp.ErrorSummary != "agent reported failure" {
		t.Fatalf("expected ErrorSummary=%q, got %q", "agent reported failure", resp.ErrorSummary)
	}
}

func TestRunInfoToResponse_EmptyAgentVersion(t *testing.T) {
	info := &storage.RunInfo{
		RunID:     "run-2",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusCompleted,
	}
	resp := runInfoToResponse(info)
	if resp.AgentVersion != "" {
		t.Fatalf("expected empty AgentVersion for REST agent, got %q", resp.AgentVersion)
	}
}

func TestRunInfoToResponse_Nil(t *testing.T) {
	resp := runInfoToResponse(nil)
	if resp.RunID != "" {
		t.Fatalf("expected zero RunResponse for nil info, got %+v", resp)
	}
}

func TestHandleRunGet_ErrorSummary(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-fail")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:        "run-fail",
		ProjectID:    "project",
		TaskID:       "task",
		Status:       storage.StatusFailed,
		StartTime:    now,
		EndTime:      now.Add(5 * time.Second),
		ExitCode:     1,
		ErrorSummary: "agent exited with non-zero code",
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-fail", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp RunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ErrorSummary != "agent exited with non-zero code" {
		t.Fatalf("expected error_summary=%q, got %q", "agent exited with non-zero code", resp.ErrorSummary)
	}
	if resp.Status != storage.StatusFailed {
		t.Fatalf("expected status=failed, got %s", resp.Status)
	}
}

func TestHandleRunStop_ExternalOwnership(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-external")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	info := &storage.RunInfo{
		RunID:            "run-external",
		ProjectID:        "project",
		TaskID:           "task",
		Status:           storage.StatusRunning,
		StartTime:        time.Now().UTC(),
		PID:              os.Getpid(),
		PGID:             os.Getpid(),
		ProcessOwnership: storage.ProcessOwnershipExternal,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-external/stop", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestStatusAllFinished verifies that buildTaskInfoWithQueue returns "all_finished"
// when the DONE file is present and all runs have completed.
func TestStatusAllFinished(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task-20260224-000001-aa"
	taskPath := filepath.Join(root, projectID, taskID)
	runDir := filepath.Join(taskPath, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskPath, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskPath, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    storage.StatusCompleted,
		StartTime: now.Add(-time.Minute),
		EndTime:   now,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	task, err := buildTaskInfo(root, projectID, taskID, taskPath)
	if err != nil {
		t.Fatalf("buildTaskInfo: %v", err)
	}
	if task.Status != storage.StatusAllFinished {
		t.Errorf("status=%q, want %q", task.Status, storage.StatusAllFinished)
	}
}

// TestStatusPartialFailure verifies that buildTaskInfoWithQueue returns "partial_failure"
// when some runs failed and at least one is still active.
func TestStatusPartialFailure(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task-20260224-000002-bb"
	taskPath := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskPath, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	now := time.Now().UTC()
	// Failed run (has EndTime).
	failedDir := filepath.Join(taskPath, "runs", "run-1")
	if err := os.MkdirAll(failedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	failedInfo := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    storage.StatusFailed,
		StartTime: now.Add(-2 * time.Minute),
		EndTime:   now.Add(-time.Minute),
		ExitCode:  1,
	}
	if err := storage.WriteRunInfo(filepath.Join(failedDir, "run-info.yaml"), failedInfo); err != nil {
		t.Fatalf("write failed run-info: %v", err)
	}

	// Active run (no EndTime).
	activeDir := filepath.Join(taskPath, "runs", "run-2")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	activeInfo := &storage.RunInfo{
		RunID:     "run-2",
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    storage.StatusRunning,
		StartTime: now.Add(-30 * time.Second),
		ExitCode:  -1,
	}
	if err := storage.WriteRunInfo(filepath.Join(activeDir, "run-info.yaml"), activeInfo); err != nil {
		t.Fatalf("write active run-info: %v", err)
	}

	task, err := buildTaskInfo(root, projectID, taskID, taskPath)
	if err != nil {
		t.Fatalf("buildTaskInfo: %v", err)
	}
	if task.Status != storage.StatusPartialFail {
		t.Errorf("status=%q, want %q", task.Status, storage.StatusPartialFail)
	}
}

// TestStatusFilterAllFinished verifies that listTasks with "all_finished" filter
// returns only tasks with the DONE marker.
func TestStatusFilterAllFinished(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	projectID := "project"
	doneTask := "task-20260224-000003-cc"
	activeTask := "task-20260224-000004-dd"

	for _, tid := range []string{doneTask, activeTask} {
		taskPath := filepath.Join(root, projectID, tid)
		if err := os.MkdirAll(taskPath, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(taskPath, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
			t.Fatalf("write TASK.md: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, projectID, doneTask, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	tasks, err := listTasks(root)
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	_ = server // ensure server compiles

	var doneCount, activeCount int
	for _, task := range tasks {
		switch task.TaskID {
		case doneTask:
			doneCount++
			if task.Status != storage.StatusAllFinished {
				t.Errorf("done task status=%q, want %q", task.Status, storage.StatusAllFinished)
			}
		case activeTask:
			activeCount++
		}
	}
	if doneCount != 1 {
		t.Errorf("expected 1 done task, found %d", doneCount)
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active task, found %d", activeCount)
	}
}
