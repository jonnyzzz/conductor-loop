package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseBody builds a simple SSE response body with content lines followed by a done event.
func sseBody(contentLines []string) string {
	var sb strings.Builder
	if len(contentLines) > 0 {
		for _, line := range contentLines {
			fmt.Fprintf(&sb, "data: %s\n", line)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("event: done\ndata: run completed\n\n")
	return sb.String()
}

// TestTaskLogsNormalStreaming verifies that lines are printed to stdout until done.
func TestTaskLogsNormalStreaming(t *testing.T) {
	taskID := "task-20260221-100000-abc"
	runID := "20260221-1000000000-11111-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects/proj/tasks/" + taskID:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     taskID,
				"status": "completed",
				"runs": []map[string]string{
					{"run_id": runID, "status": "completed"},
				},
			})
		case "/api/projects/proj/tasks/" + taskID + "/runs/" + runID + "/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, sseBody([]string{"Hello", "World", "Done"}))
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, "", false, 0)
	if err != nil {
		t.Fatalf("taskLogs: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"Hello", "World", "Done"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
}

// TestTaskLogsEmptyOutput verifies that empty output (no lines, just done event) works.
func TestTaskLogsEmptyOutput(t *testing.T) {
	taskID := "task-20260221-100000-empty"
	runID := "20260221-1000000000-22222-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects/proj/tasks/" + taskID:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     taskID,
				"status": "completed",
				"runs": []map[string]string{
					{"run_id": runID, "status": "completed"},
				},
			})
		case "/api/projects/proj/tasks/" + taskID + "/runs/" + runID + "/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			// Only a done event, no content lines.
			fmt.Fprint(w, "event: done\ndata: run completed\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, "", false, 0)
	if err != nil {
		t.Fatalf("taskLogs empty output: %v", err)
	}
	// Output should be empty (or just whitespace from empty lines).
	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("expected empty output, got: %q", buf.String())
	}
}

// TestTaskLogsAutoDetectLatestRun verifies that the latest run is selected automatically.
func TestTaskLogsAutoDetectLatestRun(t *testing.T) {
	taskID := "task-20260221-100000-detect"
	runID1 := "20260221-1000000000-33333-1"
	runID2 := "20260221-1100000000-44444-1"

	var streamedRunID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/projects/proj/tasks/"+taskID:
			// Return task with two runs; second one is running (should be selected).
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     taskID,
				"status": "running",
				"runs": []map[string]string{
					{"run_id": runID1, "status": "completed"},
					{"run_id": runID2, "status": "running"},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/stream"):
			// Record which run was streamed.
			parts := strings.Split(r.URL.Path, "/")
			for i, p := range parts {
				if p == "runs" && i+1 < len(parts) {
					streamedRunID = parts[i+1]
					break
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, sseBody([]string{"output line"}))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, "", false, 0)
	if err != nil {
		t.Fatalf("taskLogs auto-detect: %v", err)
	}

	if streamedRunID != runID2 {
		t.Errorf("expected run %q to be streamed (it's running), got %q", runID2, streamedRunID)
	}
}

// TestTaskLogsExplicitRunID verifies that --run overrides auto-detection.
func TestTaskLogsExplicitRunID(t *testing.T) {
	taskID := "task-20260221-100000-explicit"
	runID := "20260221-1000000000-55555-1"

	var streamedRunID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			parts := strings.Split(r.URL.Path, "/")
			for i, p := range parts {
				if p == "runs" && i+1 < len(parts) {
					streamedRunID = parts[i+1]
					break
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, sseBody([]string{"explicit run output"}))
			return
		}
		t.Errorf("unexpected request (should not fetch task detail when --run is given): %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, runID, false, 0)
	if err != nil {
		t.Fatalf("taskLogs explicit run: %v", err)
	}

	if streamedRunID != runID {
		t.Errorf("expected run %q to be streamed, got %q", runID, streamedRunID)
	}
	if !strings.Contains(buf.String(), "explicit run output") {
		t.Errorf("expected 'explicit run output' in output, got:\n%s", buf.String())
	}
}

// TestTaskLogsTaskNotFound verifies that a 404 on task detail returns an error.
func TestTaskLogsTaskNotFound(t *testing.T) {
	taskID := "task-20260221-100000-missing"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, "", false, 0)
	if err == nil {
		t.Fatal("expected error for 404 task, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// TestTaskLogsStreamNotFound verifies that a 404 on the stream endpoint returns an error.
func TestTaskLogsStreamNotFound(t *testing.T) {
	taskID := "task-20260221-100000-nostream"
	runID := "20260221-1000000000-66666-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			http.Error(w, `{"error":"run not found"}`, http.StatusNotFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, runID, false, 0)
	if err == nil {
		t.Fatal("expected error for 404 stream, got nil")
	}
}

// TestTaskLogsTail verifies that --tail limits to the last N lines.
func TestTaskLogsTail(t *testing.T) {
	taskID := "task-20260221-100000-tail"
	runID := "20260221-1000000000-77777-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects/proj/tasks/" + taskID:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     taskID,
				"status": "completed",
				"runs":   []map[string]string{{"run_id": runID, "status": "completed"}},
			})
		case "/api/projects/proj/tasks/" + taskID + "/runs/" + runID + "/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, sseBody([]string{"line1", "line2", "line3", "line4", "line5"}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, "", false, 3)
	if err != nil {
		t.Fatalf("taskLogs tail: %v", err)
	}

	output := buf.String()
	// Should NOT contain early lines.
	if strings.Contains(output, "line1") || strings.Contains(output, "line2") {
		t.Errorf("expected early lines to be excluded with --tail 3, got:\n%s", output)
	}
	// Should contain last 3 lines.
	for _, want := range []string{"line3", "line4", "line5"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in tail output, got:\n%s", want, output)
		}
	}
}

// TestTaskLogsNoRuns verifies error when task has no runs.
func TestTaskLogsNoRuns(t *testing.T) {
	taskID := "task-20260221-100000-noruns"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     taskID,
			"status": "idle",
			"runs":   []map[string]string{},
		})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, "", false, 0)
	if err == nil {
		t.Fatal("expected error for task with no runs, got nil")
	}
	if !strings.Contains(err.Error(), "no runs found") {
		t.Errorf("expected 'no runs found' in error, got: %v", err)
	}
}

// TestTaskLogsHeartbeat verifies that heartbeat events are ignored (not printed).
func TestTaskLogsHeartbeat(t *testing.T) {
	taskID := "task-20260221-100000-heartbeat"
	runID := "20260221-1000000000-88888-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			// Send heartbeat then content then done.
			fmt.Fprint(w, "event: heartbeat\ndata: {}\n\n")
			fmt.Fprint(w, "data: real content\n\n")
			fmt.Fprint(w, "event: done\ndata: run completed\n\n")
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := taskLogs(&buf, srv.URL, "proj", taskID, runID, false, 0)
	if err != nil {
		t.Fatalf("taskLogs heartbeat: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "{}") {
		t.Errorf("heartbeat data should not appear in output, got:\n%s", output)
	}
	if !strings.Contains(output, "real content") {
		t.Errorf("expected 'real content' in output, got:\n%s", output)
	}
}

// TestTaskLogsCmdHelp verifies that the logs subcommand is registered and shows help.
func TestTaskLogsCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"task", "logs", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	for _, want := range []string{"logs", "project", "follow", "tail", "run"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in task logs --help output, got:\n%s", want, output)
		}
	}
}

// TestTaskLogsAppearsInTaskHelp verifies the logs subcommand is listed in task help.
func TestTaskLogsAppearsInTaskHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"task", "--help"})
	_ = cmd.Execute()

	if !strings.Contains(buf.String(), "logs") {
		t.Errorf("expected 'logs' in task --help output, got:\n%s", buf.String())
	}
}
