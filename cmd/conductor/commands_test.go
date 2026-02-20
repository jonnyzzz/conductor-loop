package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
