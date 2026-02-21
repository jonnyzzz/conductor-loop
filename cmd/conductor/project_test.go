package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- projectList tests ---

func TestProjectListSuccess(t *testing.T) {
	ts := time.Date(2026, 2, 21, 2, 55, 0, 0, time.UTC)
	respBody := projectListAPIResponse{
		Projects: []projectSummaryResponse{
			{ID: "conductor-loop", LastActivity: ts, TaskCount: 78},
			{ID: "my-other-project", LastActivity: ts.Add(-8 * time.Hour), TaskCount: 3},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/projects" {
			t.Errorf("expected /api/projects, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := projectList(srv.URL, false); err != nil {
		t.Fatalf("projectList: %v", err)
	}
}

func TestProjectListJSONOutput(t *testing.T) {
	respBody := projectListAPIResponse{
		Projects: []projectSummaryResponse{
			{ID: "proj-a", TaskCount: 5},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := projectList(srv.URL, true); err != nil {
		t.Fatalf("projectList json: %v", err)
	}
}

func TestProjectListServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := projectList(srv.URL, false); err == nil {
		t.Fatal("expected error on 500 response")
	}
}

// --- taskList tests ---

func TestTaskListSuccess(t *testing.T) {
	ts := time.Date(2026, 2, 21, 2, 55, 0, 0, time.UTC)
	respBody := taskListAPIResponse{
		Items: []taskListItem{
			{ID: "task-20260221-025500-aaa", ProjectID: "proj", Status: "completed", LastActivity: ts, RunCount: 2},
			{ID: "task-20260221-024500-bbb", ProjectID: "proj", Status: "running", LastActivity: ts.Add(-10 * time.Minute), RunCount: 1},
			{ID: "task-20260221-023000-ccc", ProjectID: "proj", Status: "idle", RunCount: 0},
		},
		Total:   3,
		HasMore: false,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/projects/proj/tasks" {
			t.Errorf("expected /api/projects/proj/tasks, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := taskList(srv.URL, "proj", false); err != nil {
		t.Fatalf("taskList: %v", err)
	}
}

func TestTaskListJSONOutput(t *testing.T) {
	respBody := taskListAPIResponse{
		Items:   []taskListItem{{ID: "task-20260221-025500-aaa", ProjectID: "proj", Status: "completed", RunCount: 1}},
		Total:   1,
		HasMore: false,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := taskList(srv.URL, "proj", true); err != nil {
		t.Fatalf("taskList json: %v", err)
	}
}

func TestTaskListServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := taskList(srv.URL, "proj", false); err == nil {
		t.Fatal("expected error on 500 response")
	}
}

func TestTaskListHasMore(t *testing.T) {
	respBody := taskListAPIResponse{
		Items: []taskListItem{
			{ID: "task-20260221-025500-aaa", ProjectID: "proj", Status: "completed", RunCount: 1},
		},
		Total:   78,
		HasMore: true,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := taskList(srv.URL, "proj", false); err != nil {
		t.Fatalf("taskList has_more: %v", err)
	}
}

// --- cobra command wiring tests ---

func TestProjectAppearsInHelp(t *testing.T) {
	var out strings.Builder
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute()

	if !strings.Contains(out.String(), "project") {
		t.Errorf("expected 'project' in help output, got:\n%s", out.String())
	}
}

func TestTaskListAppearsInHelp(t *testing.T) {
	var out strings.Builder
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"task", "--help"})
	_ = cmd.Execute()

	if !strings.Contains(out.String(), "list") {
		t.Errorf("expected 'list' in task help output, got:\n%s", out.String())
	}
}

// --- taskDelete tests ---

func TestTaskDeleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/projects/myproject/tasks/task-20260221-120000-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := taskDelete(srv.URL, "task-20260221-120000-abc", "myproject", false); err != nil {
		t.Fatalf("taskDelete: %v", err)
	}
}

func TestTaskDeleteJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := taskDelete(srv.URL, "task-20260221-120000-abc", "proj", true); err != nil {
		t.Fatalf("taskDelete json: %v", err)
	}
}

func TestTaskDeleteConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"task has running runs"}`, http.StatusConflict)
	}))
	defer srv.Close()

	err := taskDelete(srv.URL, "task-20260221-120000-abc", "proj", false)
	if err == nil {
		t.Fatal("expected error on 409 response")
	}
	if !strings.Contains(err.Error(), "running runs") {
		t.Errorf("expected running-runs error, got: %v", err)
	}
}

func TestTaskDeleteNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	err := taskDelete(srv.URL, "task-20260221-120000-abc", "proj", false)
	if err == nil {
		t.Fatal("expected error on 404 response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

func TestTaskDeleteAppearsInHelp(t *testing.T) {
	var out strings.Builder
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"task", "--help"})
	_ = cmd.Execute()

	if !strings.Contains(out.String(), "delete") {
		t.Errorf("expected 'delete' in task help output, got:\n%s", out.String())
	}
}

// --- projectStats tests ---

func TestProjectStatsSuccess(t *testing.T) {
	respBody := projectStatsResponse{
		ProjectID:            "conductor-loop",
		TotalTasks:           42,
		TotalRuns:            150,
		RunningRuns:          2,
		CompletedRuns:        130,
		FailedRuns:           15,
		CrashedRuns:          3,
		MessageBusFiles:      43,
		MessageBusTotalBytes: 2 * 1024 * 1024,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/projects/conductor-loop/stats" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := projectStats(srv.URL, "conductor-loop", false); err != nil {
		t.Fatalf("projectStats: %v", err)
	}
}

func TestProjectStatsJSONOutput(t *testing.T) {
	respBody := projectStatsResponse{
		ProjectID: "proj",
		TotalRuns: 5,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	if err := projectStats(srv.URL, "proj", true); err != nil {
		t.Fatalf("projectStats json: %v", err)
	}
}

func TestProjectStatsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := projectStats(srv.URL, "proj", false); err == nil {
		t.Fatal("expected error on 500 response")
	}
}

func TestProjectStatsAppearsInHelp(t *testing.T) {
	var out strings.Builder
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"project", "--help"})
	_ = cmd.Execute()

	if !strings.Contains(out.String(), "stats") {
		t.Errorf("expected 'stats' in project help output, got:\n%s", out.String())
	}
}

// --- formatBytes tests ---

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{2048, "2.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{3 * 1024 * 1024, "3.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
	}
	for _, tc := range tests {
		if got := formatBytes(tc.input); got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
