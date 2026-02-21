package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func makeRunsResponse(items []runListItem, total int, hasMore bool) runsListAPIResponse {
	return runsListAPIResponse{Items: items, Total: total, HasMore: hasMore}
}

// TestTaskRunsSuccess verifies normal listing with multiple completed runs.
func TestTaskRunsSuccess(t *testing.T) {
	taskID := "task-20260221-100000-runs"
	end1 := time.Date(2026, 2, 21, 7, 5, 23, 0, time.UTC)
	end2 := time.Date(2026, 2, 21, 6, 50, 12, 0, time.UTC)
	respBody := makeRunsResponse([]runListItem{
		{
			ID:        "20260221-070000-12345-1",
			Agent:     "claude",
			Status:    "completed",
			ExitCode:  0,
			StartTime: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
			EndTime:   &end1,
		},
		{
			ID:           "20260221-065000-12344-1",
			Agent:        "claude",
			Status:       "failed",
			ExitCode:     1,
			StartTime:    time.Date(2026, 2, 21, 6, 50, 0, 0, time.UTC),
			EndTime:      &end2,
			ErrorSummary: "exit code 1: general failure",
		},
	}, 2, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		want := "/api/projects/proj/tasks/" + taskID + "/runs"
		if !strings.HasPrefix(r.URL.Path, want) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 50)
	if err != nil {
		t.Fatalf("taskRuns: %v", err)
	}

	output := buf.String()
	// Header should appear.
	if !strings.Contains(output, "RUN ID") {
		t.Errorf("expected RUN ID header in output, got:\n%s", output)
	}
	// Both runs should appear.
	if !strings.Contains(output, "20260221-070000-12345-1") {
		t.Errorf("expected first run ID in output, got:\n%s", output)
	}
	if !strings.Contains(output, "20260221-065000-12344-1") {
		t.Errorf("expected second run ID in output, got:\n%s", output)
	}
	if !strings.Contains(output, "failed") {
		t.Errorf("expected 'failed' status in output, got:\n%s", output)
	}
	if !strings.Contains(output, "exit code 1: general failure") {
		t.Errorf("expected error summary in output, got:\n%s", output)
	}
}

// TestTaskRunsEmpty verifies helpful message when task has no runs.
func TestTaskRunsEmpty(t *testing.T) {
	taskID := "task-20260221-100000-noruns"
	respBody := makeRunsResponse(nil, 0, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 50)
	if err != nil {
		t.Fatalf("taskRuns empty: %v", err)
	}
	if !strings.Contains(buf.String(), "no runs found") {
		t.Errorf("expected 'no runs found' message, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), taskID) {
		t.Errorf("expected task ID in empty message, got:\n%s", buf.String())
	}
}

// TestTaskRunsJSONOutput verifies --json passes raw JSON through.
func TestTaskRunsJSONOutput(t *testing.T) {
	taskID := "task-20260221-100000-json"
	end := time.Date(2026, 2, 21, 7, 5, 0, 0, time.UTC)
	respBody := makeRunsResponse([]runListItem{
		{ID: "20260221-070000-99999-1", Agent: "claude", Status: "completed", EndTime: &end,
			StartTime: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC)},
	}, 1, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, true, 50)
	if err != nil {
		t.Fatalf("taskRuns json: %v", err)
	}

	output := buf.String()
	// Should be parseable JSON.
	var parsed runsListAPIResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Errorf("expected valid JSON output, got:\n%s\nerr: %v", output, err)
	}
	if parsed.Total != 1 {
		t.Errorf("expected total=1, got %d", parsed.Total)
	}
}

// TestTaskRunsNotFound verifies 404 response returns meaningful error.
func TestTaskRunsNotFound(t *testing.T) {
	taskID := "task-20260221-100000-missing"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 50)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), taskID) {
		t.Errorf("expected task ID in error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// TestTaskRunsServerError verifies 500 response returns an error.
func TestTaskRunsServerError(t *testing.T) {
	taskID := "task-20260221-100000-servererr"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 50)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got: %v", err)
	}
}

// TestTaskRunsRunningDuration verifies "running" shown when end_time is nil.
func TestTaskRunsRunningDuration(t *testing.T) {
	taskID := "task-20260221-100000-running"
	respBody := makeRunsResponse([]runListItem{
		{
			ID:        "20260221-070000-11111-1",
			Agent:     "claude",
			Status:    "running",
			StartTime: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
			EndTime:   nil, // still running
		},
	}, 1, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 50)
	if err != nil {
		t.Fatalf("taskRuns running: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "running") {
		t.Errorf("expected 'running' in duration column, got:\n%s", output)
	}
	// Exit code column should show "-" for running runs.
	if !strings.Contains(output, "-") {
		t.Errorf("expected '-' for exit code of running run, got:\n%s", output)
	}
}

// TestFormatRunDuration verifies duration formatting logic.
func TestFormatRunDuration(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      *time.Time
		expected string
	}{
		{
			name:     "running (no end)",
			start:    time.Now(),
			end:      nil,
			expected: "running",
		},
		{
			name:  "under one minute",
			start: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
			end: func() *time.Time {
				t := time.Date(2026, 2, 21, 7, 0, 45, 0, time.UTC)
				return &t
			}(),
			expected: "45s",
		},
		{
			name:  "5 minutes 23 seconds",
			start: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
			end: func() *time.Time {
				t := time.Date(2026, 2, 21, 7, 5, 23, 0, time.UTC)
				return &t
			}(),
			expected: "5m23s",
		},
		{
			name:  "exactly one minute",
			start: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
			end: func() *time.Time {
				t := time.Date(2026, 2, 21, 7, 1, 0, 0, time.UTC)
				return &t
			}(),
			expected: "1m0s",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatRunDuration(tc.start, tc.end)
			if got != tc.expected {
				t.Errorf("formatRunDuration = %q, want %q", got, tc.expected)
			}
		})
	}
}

// TestTaskRunsErrorSummaryTruncation verifies long error summaries are truncated to 40 chars.
func TestTaskRunsErrorSummaryTruncation(t *testing.T) {
	taskID := "task-20260221-100000-truncate"
	longError := strings.Repeat("x", 80) // 80 chars, should be truncated to 40
	end := time.Date(2026, 2, 21, 7, 5, 0, 0, time.UTC)
	respBody := makeRunsResponse([]runListItem{
		{
			ID:           "20260221-070000-22222-1",
			Agent:        "claude",
			Status:       "failed",
			ExitCode:     1,
			StartTime:    time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
			EndTime:      &end,
			ErrorSummary: longError,
		},
	}, 1, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 50)
	if err != nil {
		t.Fatalf("taskRuns truncation: %v", err)
	}

	output := buf.String()
	// The full 80-char string should NOT appear.
	if strings.Contains(output, longError) {
		t.Errorf("expected long error to be truncated, but full string appears in:\n%s", output)
	}
	// But 40 chars of it should appear.
	if !strings.Contains(output, strings.Repeat("x", 40)) {
		t.Errorf("expected truncated 40-char prefix in output, got:\n%s", output)
	}
}

// TestTaskRunsHasMore verifies pagination message is shown when has_more is true.
func TestTaskRunsHasMore(t *testing.T) {
	taskID := "task-20260221-100000-hasmore"
	end := time.Date(2026, 2, 21, 7, 5, 0, 0, time.UTC)
	respBody := makeRunsResponse([]runListItem{
		{ID: "20260221-070000-33333-1", Agent: "claude", Status: "completed",
			StartTime: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC), EndTime: &end},
	}, 42, true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskRuns(&buf, srv.URL, "proj", taskID, false, 1)
	if err != nil {
		t.Fatalf("taskRuns has_more: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "42") {
		t.Errorf("expected total count (42) in pagination message, got:\n%s", output)
	}
	if !strings.Contains(output, "--limit") {
		t.Errorf("expected '--limit' hint in pagination message, got:\n%s", output)
	}
}

// TestTaskRunsAppearsInTaskHelp verifies the runs subcommand is listed in task help.
func TestTaskRunsAppearsInTaskHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"task", "--help"})
	_ = cmd.Execute()

	if !strings.Contains(buf.String(), "runs") {
		t.Errorf("expected 'runs' in task --help output, got:\n%s", buf.String())
	}
}

// TestTaskRunsCmdHelp verifies the runs command shows correct usage and flags.
func TestTaskRunsCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"task", "runs", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	for _, want := range []string{"runs", "project", "limit", "json", "server"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in task runs --help output, got:\n%s", want, output)
		}
	}
}

// TestTaskRunsLimitQueryParam verifies that --limit is passed as a query parameter.
func TestTaskRunsLimitQueryParam(t *testing.T) {
	taskID := "task-20260221-100000-limit"

	var receivedLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"items":[],"total":0,"has_more":false}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	_ = taskRuns(&buf, srv.URL, "proj", taskID, false, 25)

	if receivedLimit != "25" {
		t.Errorf("expected limit=25 in query, got %q", receivedLimit)
	}
}
