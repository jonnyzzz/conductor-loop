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
