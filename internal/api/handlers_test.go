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
