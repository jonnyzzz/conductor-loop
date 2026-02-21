package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTaskResumeSuccess verifies that taskResume handles a 200 OK response correctly.
func TestTaskResumeSuccess(t *testing.T) {
	resp := taskResumeResponse{
		ProjectID: "my-project",
		TaskID:    "task-20260221-100000-abc",
		Resumed:   true,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/api/projects/my-project/tasks/task-20260221-100000-abc/resume"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	if err := taskResume(srv.URL, "task-20260221-100000-abc", "my-project", false); err != nil {
		t.Fatalf("taskResume: %v", err)
	}
}

func TestTaskResumeJSONOutput(t *testing.T) {
	resp := taskResumeResponse{
		ProjectID: "proj",
		TaskID:    "task-20260221-100000-xyz",
		Resumed:   true,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	if err := taskResume(srv.URL, "task-20260221-100000-xyz", "proj", true); err != nil {
		t.Fatalf("taskResume JSON: %v", err)
	}
}

func TestTaskResumeNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintln(w, "task not found")
	}))
	defer srv.Close()

	err := taskResume(srv.URL, "task-20260221-100000-missing", "proj", false)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error, got: %v", err)
	}
}

func TestTaskResumeBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "task already running")
	}))
	defer srv.Close()

	err := taskResume(srv.URL, "task-20260221-100000-abc", "proj", false)
	if err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestTaskResumeServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "internal server error")
	}))
	defer srv.Close()

	err := taskResume(srv.URL, "task-20260221-100000-abc", "proj", false)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestTaskResumeConnectionError(t *testing.T) {
	err := taskResume("http://127.0.0.1:1", "task-20260221-100000-abc", "proj", false)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

// TestWaitForRunStartTimesOut verifies that waitForRunStart returns a timeout error
// when no runs are found within the deadline.
// resolveLatestRunID returns "no runs found" when task exists but has empty runs list.
func TestWaitForRunStartTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a task with no runs — resolveLatestRunID returns "no runs found" error.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `{"runs":[]}`)
	}))
	defer srv.Close()

	origInterval := followRetryInterval
	followRetryInterval = 5 * time.Millisecond
	defer func() { followRetryInterval = origInterval }()

	_, err := waitForRunStart(srv.URL, "proj", "task-20260221-100000-abc", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected 'timed out' in error, got: %v", err)
	}
}

// TestWaitForRunStartSucceeds verifies that waitForRunStart returns a run ID
// when a run is found. resolveLatestRunID expects {"runs":[...]} JSON.
func TestWaitForRunStartSucceeds(t *testing.T) {
	taskDetail := map[string]interface{}{
		"runs": []map[string]interface{}{
			{"run_id": "run-20260221-1000000000-12345", "status": "running"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(taskDetail)
	}))
	defer srv.Close()

	origInterval := followRetryInterval
	followRetryInterval = 5 * time.Millisecond
	defer func() { followRetryInterval = origInterval }()

	runID, err := waitForRunStart(srv.URL, "proj", "task-20260221-100000-abc", 2*time.Second)
	if err != nil {
		t.Fatalf("waitForRunStart: %v", err)
	}
	if runID != "run-20260221-1000000000-12345" {
		t.Fatalf("expected specific run ID, got %q", runID)
	}
}
