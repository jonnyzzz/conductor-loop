package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- serverStatus tests ---

func TestServerStatusSuccess(t *testing.T) {
	respBody := conductorStatusResponse{
		ActiveRunsCount:  3,
		UptimeSeconds:    125.5,
		ConfiguredAgents: []string{"claude", "codex"},
		Version:          "1.2.3",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/status" {
			t.Errorf("expected /api/v1/status, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := serverStatus(srv.URL, false); err != nil {
		t.Fatalf("serverStatus: %v", err)
	}
}

func TestServerStatusJSONOutput(t *testing.T) {
	respBody := conductorStatusResponse{
		ActiveRunsCount:  1,
		UptimeSeconds:    60,
		ConfiguredAgents: []string{"claude"},
		Version:          "dev",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := serverStatus(srv.URL, true); err != nil {
		t.Fatalf("serverStatus json: %v", err)
	}
}

func TestServerStatusServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := serverStatus(srv.URL, false); err == nil {
		t.Fatal("expected error on 500 response")
	}
}

func TestServerStatusNoAgents(t *testing.T) {
	respBody := conductorStatusResponse{
		ActiveRunsCount:  0,
		UptimeSeconds:    0,
		ConfiguredAgents: nil,
		Version:          "dev",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := serverStatus(srv.URL, false); err != nil {
		t.Fatalf("serverStatus no agents: %v", err)
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{0, "0s"},
		{45, "45s"},
		{60, "1m 0s"},
		{90, "1m 30s"},
		{3600, "1h 0m 0s"},
		{3661, "1h 1m 1s"},
		{7384, "2h 3m 4s"},
	}
	for _, tc := range cases {
		got := formatUptime(tc.seconds)
		if got != tc.want {
			t.Errorf("formatUptime(%v) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

// --- taskStop tests ---

func TestTaskStopSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tasks/task-20260220-100000-foo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(taskStopResponse{StoppedRuns: 2})
	}))
	defer srv.Close()

	if err := taskStop(srv.URL, "task-20260220-100000-foo", "", false); err != nil {
		t.Fatalf("taskStop: %v", err)
	}
}

func TestTaskStopWithProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project_id") != "my-proj" {
			t.Errorf("expected project_id=my-proj, got %q", r.URL.Query().Get("project_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(taskStopResponse{StoppedRuns: 1})
	}))
	defer srv.Close()

	if err := taskStop(srv.URL, "task-20260220-100000-foo", "my-proj", false); err != nil {
		t.Fatalf("taskStop with project: %v", err)
	}
}

func TestTaskStopJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(taskStopResponse{StoppedRuns: 0})
	}))
	defer srv.Close()

	if err := taskStop(srv.URL, "task-20260220-100000-bar", "", true); err != nil {
		t.Fatalf("taskStop json: %v", err)
	}
}

func TestTaskStopServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	if err := taskStop(srv.URL, "task-no-such", "", false); err == nil {
		t.Fatal("expected error on 404 response")
	}
}

// --- cobra command wiring tests for status and task stop ---

func TestStatusCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"status", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status --help: %v", err)
	}
}

func TestTaskStopCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task", "stop", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("task stop --help: %v", err)
	}
}

func TestStatusAppearsInHelp(t *testing.T) {
	var out strings.Builder
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute()

	if !strings.Contains(out.String(), "status") {
		t.Errorf("expected 'status' in help output, got:\n%s", out.String())
	}
}

func TestTaskStopAppearsInHelp(t *testing.T) {
	var out strings.Builder
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"task", "--help"})
	_ = cmd.Execute()

	if !strings.Contains(out.String(), "stop") {
		t.Errorf("expected 'stop' in task help output, got:\n%s", out.String())
	}
}

// --- jobSubmit tests ---

func TestJobSubmitSuccess(t *testing.T) {
	respBody := jobCreateResponse{
		ProjectID: "my-project",
		TaskID:    "task-20260220-100000-test",
		RunID:     "20260220-1000000000-11111",
		Status:    "started",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tasks" {
			t.Errorf("expected /api/v1/tasks, got %s", r.URL.Path)
		}
		var req jobCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.ProjectID != "my-project" {
			t.Errorf("expected project_id=my-project, got %q", req.ProjectID)
		}
		if req.AgentType != "claude" {
			t.Errorf("expected agent_type=claude, got %q", req.AgentType)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	req := jobCreateRequest{
		ProjectID: "my-project",
		TaskID:    "task-20260220-100000-test",
		AgentType: "claude",
		Prompt:    "Do the thing",
	}
	if err := jobSubmit(srv.URL, req, false, false); err != nil {
		t.Fatalf("jobSubmit: %v", err)
	}
}

func TestJobSubmitJSONOutput(t *testing.T) {
	respBody := jobCreateResponse{
		ProjectID: "proj",
		TaskID:    "task-20260220-100000-x",
		RunID:     "run-001",
		Status:    "started",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	req := jobCreateRequest{ProjectID: "proj", TaskID: "task-20260220-100000-x", AgentType: "claude", Prompt: "Hi"}
	if err := jobSubmit(srv.URL, req, false, true); err != nil {
		t.Fatalf("jobSubmit json: %v", err)
	}
}

func TestJobSubmitServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	req := jobCreateRequest{ProjectID: "p", TaskID: "t", AgentType: "a", Prompt: "x"}
	if err := jobSubmit(srv.URL, req, false, false); err == nil {
		t.Fatal("expected error on 400 response")
	}
}

func TestJobSubmitWait(t *testing.T) {
	pollCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/tasks":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(jobCreateResponse{
				ProjectID: "p", TaskID: "t", RunID: "run-999", Status: "started",
			})
		case "/api/v1/runs/run-999":
			pollCount++
			run := jobRunResponse{RunID: "run-999", Status: "running"}
			if pollCount >= 2 {
				run.Status = "completed"
				run.EndTime = time.Now()
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(run)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Override poll interval to avoid slow tests.
	old := pollInterval
	pollInterval = 0
	defer func() { pollInterval = old }()

	req := jobCreateRequest{ProjectID: "p", TaskID: "t", AgentType: "claude", Prompt: "x"}
	if err := jobSubmit(srv.URL, req, true, false); err != nil {
		t.Fatalf("jobSubmit with wait: %v", err)
	}
	if pollCount < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", pollCount)
	}
}

// --- jobList tests ---

func TestJobListSuccess(t *testing.T) {
	ts := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	respBody := jobTaskListResponse{
		Tasks: []jobTaskResponse{
			{ProjectID: "proj-a", TaskID: "task-20260220-100000-foo", Status: "running", LastActivity: ts},
			{ProjectID: "proj-b", TaskID: "task-20260220-110000-bar", Status: "idle"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tasks" {
			t.Errorf("expected /api/v1/tasks, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := jobList(srv.URL, "", false); err != nil {
		t.Fatalf("jobList: %v", err)
	}
}

func TestJobListWithProjectFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project_id") != "my-proj" {
			t.Errorf("expected project_id=my-proj, got %q", r.URL.Query().Get("project_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jobTaskListResponse{Tasks: []jobTaskResponse{}})
	}))
	defer srv.Close()

	if err := jobList(srv.URL, "my-proj", false); err != nil {
		t.Fatalf("jobList with project: %v", err)
	}
}

func TestJobListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jobTaskListResponse{Tasks: nil})
	}))
	defer srv.Close()

	if err := jobList(srv.URL, "", false); err != nil {
		t.Fatalf("jobList empty: %v", err)
	}
}

func TestJobListJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jobTaskListResponse{Tasks: []jobTaskResponse{}})
	}))
	defer srv.Close()

	if err := jobList(srv.URL, "", true); err != nil {
		t.Fatalf("jobList json: %v", err)
	}
}

func TestJobListServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := jobList(srv.URL, "", false); err == nil {
		t.Fatal("expected error on 500 response")
	}
}

// --- taskStatus tests ---

func TestTaskStatusSuccess(t *testing.T) {
	ts := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	respBody := taskDetailResponse{
		ProjectID:    "proj-a",
		TaskID:       "task-20260220-100000-foo",
		Status:       "running",
		LastActivity: ts,
		Runs: []taskRunSummary{
			{RunID: "run-001", Status: "running", StartTime: ts},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tasks/task-20260220-100000-foo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := taskStatus(srv.URL, "task-20260220-100000-foo", "", false); err != nil {
		t.Fatalf("taskStatus: %v", err)
	}
}

func TestTaskStatusWithProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project_id") != "my-proj" {
			t.Errorf("expected project_id=my-proj, got %q", r.URL.Query().Get("project_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(taskDetailResponse{
			ProjectID: "my-proj",
			TaskID:    "task-20260220-100000-bar",
			Status:    "idle",
		})
	}))
	defer srv.Close()

	if err := taskStatus(srv.URL, "task-20260220-100000-bar", "my-proj", false); err != nil {
		t.Fatalf("taskStatus with project: %v", err)
	}
}

func TestTaskStatusJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(taskDetailResponse{ProjectID: "p", TaskID: "t", Status: "idle"})
	}))
	defer srv.Close()

	if err := taskStatus(srv.URL, "t", "", true); err != nil {
		t.Fatalf("taskStatus json: %v", err)
	}
}

func TestTaskStatusNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	if err := taskStatus(srv.URL, "task-no-such", "", false); err == nil {
		t.Fatal("expected error on 404 response")
	}
}

func TestTaskStatusWithCompletedRuns(t *testing.T) {
	start := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 20, 11, 0, 0, 0, time.UTC)
	respBody := taskDetailResponse{
		ProjectID:    "proj",
		TaskID:       "task-20260220-100000-done",
		Status:       "completed",
		LastActivity: end,
		Runs: []taskRunSummary{
			{RunID: "run-001", Status: "completed", StartTime: start, EndTime: end, ExitCode: 0},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := taskStatus(srv.URL, "task-20260220-100000-done", "proj", false); err != nil {
		t.Fatalf("taskStatus completed: %v", err)
	}
}

// --- cobra command wiring tests ---

func TestJobSubmitHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job", "submit", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("job submit --help: %v", err)
	}
}

func TestJobListHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job", "list", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("job list --help: %v", err)
	}
}

func TestTaskStatusHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task", "status", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("task status --help: %v", err)
	}
}

func TestJobCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("job (no subcommand): %v", err)
	}
}

func TestTaskCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("task (no subcommand): %v", err)
	}
}

// --- loadPrompt tests ---

func TestLoadPromptInline(t *testing.T) {
	got, err := loadPrompt("hello world", "")
	if err != nil {
		t.Fatalf("loadPrompt: %v", err)
	}
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestLoadPromptFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.md")
	content := "# Task\nDo the thing.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := loadPrompt("", path)
	if err != nil {
		t.Fatalf("loadPrompt: %v", err)
	}
	if got != content {
		t.Errorf("expected file content, got %q", got)
	}
}

func TestLoadPromptBothFlagsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.md")
	_ = os.WriteFile(path, []byte("x"), 0o644)

	_, err := loadPrompt("inline text", path)
	if err == nil {
		t.Fatal("expected error when both --prompt and --prompt-file are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %v", err)
	}
}

func TestLoadPromptNeitherFlagError(t *testing.T) {
	_, err := loadPrompt("", "")
	if err == nil {
		t.Fatal("expected error when neither --prompt nor --prompt-file is set")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' in error, got: %v", err)
	}
}

func TestLoadPromptFileNotFound(t *testing.T) {
	_, err := loadPrompt("", "/nonexistent/path/prompt.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read prompt file") {
		t.Errorf("expected 'read prompt file' in error, got: %v", err)
	}
}

func TestLoadPromptEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(path, []byte("   \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadPrompt("", path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' in error, got: %v", err)
	}
}

// TestJobSubmitPromptFile verifies that --prompt-file is wired into the submit command.
func TestJobSubmitPromptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-prompt.md")
	promptContent := "Do the important task now.\n"
	if err := os.WriteFile(path, []byte(promptContent), 0o644); err != nil {
		t.Fatal(err)
	}

	var receivedPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jobCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		receivedPrompt = req.Prompt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(jobCreateResponse{
			ProjectID: "p", TaskID: "t", RunID: "r", Status: "started",
		})
	}))
	defer srv.Close()

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"job", "submit",
		"--server", srv.URL,
		"--project", "p",
		"--task", "task-20260221-120000-test",
		"--agent", "claude",
		"--prompt-file", path,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("job submit --prompt-file: %v", err)
	}
	if receivedPrompt != promptContent {
		t.Errorf("server received prompt %q, want %q", receivedPrompt, promptContent)
	}
}

// TestJobSubmitHelpShowsPromptFile verifies the help text mentions --prompt-file.
func TestJobSubmitHelpShowsPromptFile(t *testing.T) {
	cmd := newRootCmd()
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"job", "submit", "--help"})
	_ = cmd.Execute()
	if !strings.Contains(buf.String(), "prompt-file") {
		t.Errorf("expected 'prompt-file' in job submit help, got:\n%s", buf.String())
	}
}
