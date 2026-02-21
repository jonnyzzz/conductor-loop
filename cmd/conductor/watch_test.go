package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestWatchAllTasksComplete verifies that watch exits immediately when all tasks are already done.
func TestWatchAllTasksComplete(t *testing.T) {
	tasks := taskListAPIResponse{
		Items: []taskListItem{
			{ID: "task-20260221-010000-aaa", ProjectID: "myproj", Status: "completed", RunCount: 2},
			{ID: "task-20260221-020000-bbb", ProjectID: "myproj", Status: "failed", RunCount: 1},
		},
		Total:   2,
		HasMore: false,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/myproj/tasks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(tasks)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := runConductorWatch(&buf, srv.URL, "myproj", nil, 10*time.Second, 50*time.Millisecond, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "All tasks completed") {
		t.Errorf("expected 'All tasks completed' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "task-20260221-010000-aaa") {
		t.Errorf("expected task ID in output, got:\n%s", output)
	}
}

// TestWatchTimeout verifies that watch exits with an error when the timeout is reached.
func TestWatchTimeout(t *testing.T) {
	tasks := taskListAPIResponse{
		Items: []taskListItem{
			{ID: "task-20260221-010000-aaa", ProjectID: "myproj", Status: "running", RunCount: 1},
		},
		Total:   1,
		HasMore: false,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(tasks)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := runConductorWatch(&buf, srv.URL, "myproj", nil, 200*time.Millisecond, 50*time.Millisecond, false)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected 'timeout' in error message, got: %v", err)
	}
}

// TestWatchSpecificTask verifies that a single specific task can be watched via the project-scoped endpoint.
func TestWatchSpecificTask(t *testing.T) {
	taskID := "task-20260221-030000-ccc"
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/projects/myproj/tasks/" + taskID
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: %s (want %s)", r.URL.Path, expectedPath)
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return completed after second poll.
		status := "running"
		done := false
		if callCount >= 2 {
			status = "completed"
			done = true
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     taskID,
			"status": status,
			"done":   done,
			"runs":   []map[string]string{{"run_id": "run-001"}},
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := runConductorWatch(&buf, srv.URL, "myproj", []string{taskID}, 10*time.Second, 50*time.Millisecond, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "All tasks completed") {
		t.Errorf("expected 'All tasks completed' in output, got:\n%s", output)
	}
	if !strings.Contains(output, taskID) {
		t.Errorf("expected task ID in output, got:\n%s", output)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", callCount)
	}
}

// TestWatchJSONOutput verifies that --json flag produces valid JSON with expected fields.
func TestWatchJSONOutput(t *testing.T) {
	taskID := "task-20260221-040000-ddd"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/projects/proj/tasks/" + taskID
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     taskID,
			"status": "completed",
			"done":   true,
			"runs":   []map[string]string{{"run_id": "run-001"}, {"run_id": "run-002"}},
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := runConductorWatch(&buf, srv.URL, "proj", []string{taskID}, 10*time.Second, 50*time.Millisecond, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	type jsonResult struct {
		Tasks []struct {
			TaskID   string `json:"task_id"`
			Status   string `json:"status"`
			RunCount int    `json:"run_count"`
			Done     bool   `json:"done"`
		} `json:"tasks"`
		AllDone bool `json:"all_done"`
	}

	var out jsonResult
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if !out.AllDone {
		t.Errorf("expected all_done=true, got false")
	}
	if len(out.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(out.Tasks))
	}
	if out.Tasks[0].TaskID != taskID {
		t.Errorf("expected task_id=%q, got %q", taskID, out.Tasks[0].TaskID)
	}
	if out.Tasks[0].Status != "completed" {
		t.Errorf("expected status='completed', got %q", out.Tasks[0].Status)
	}
	if !out.Tasks[0].Done {
		t.Errorf("expected done=true")
	}
	if out.Tasks[0].RunCount != 2 {
		t.Errorf("expected run_count=2, got %d", out.Tasks[0].RunCount)
	}
}

// TestWatchEmptyProject verifies that watch exits with an error when the project has no tasks.
func TestWatchEmptyProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(taskListAPIResponse{
			Items:   []taskListItem{},
			Total:   0,
			HasMore: false,
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := runConductorWatch(&buf, srv.URL, "empty-proj", nil, 5*time.Second, 50*time.Millisecond, false)
	if err == nil {
		t.Fatal("expected error for empty project, got nil")
	}
	if !strings.Contains(err.Error(), "no tasks found") {
		t.Errorf("expected 'no tasks found' in error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No tasks found") {
		t.Errorf("expected 'No tasks found' in output, got:\n%s", output)
	}
}

// TestWatchCmdHelp verifies that the watch command is registered and appears in help output.
func TestWatchCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"watch", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	for _, want := range []string{"watch", "project", "task", "timeout", "interval"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in watch --help output, got:\n%s", want, output)
		}
	}
}

// TestWatchAppearsInRootHelp verifies the watch command appears in the root help output.
func TestWatchAppearsInRootHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute()

	if !strings.Contains(buf.String(), "watch") {
		t.Errorf("expected 'watch' in root help output, got:\n%s", buf.String())
	}
}

// TestWatchServerError verifies that a server error is propagated.
func TestWatchServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := runConductorWatch(&buf, srv.URL, "myproj", nil, 5*time.Second, 50*time.Millisecond, false)
	if err == nil {
		t.Fatal("expected error on server 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}
