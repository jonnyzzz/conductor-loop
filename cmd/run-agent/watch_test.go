package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func init() {
	// Speed up watch polling for tests.
	watchPollInterval = 50 * time.Millisecond
}

// makeWatchRun creates a task run directory with a run-info.yaml for watch tests.
func makeWatchRun(t *testing.T, root, project, task, runID, status string, startTime, endTime time.Time, exitCode int) string {
	t.Helper()
	runDir := filepath.Join(root, project, task, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}
	info := &storage.RunInfo{
		RunID:     runID,
		ProjectID: project,
		TaskID:    task,
		AgentType: "claude",
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		ExitCode:  exitCode,
	}
	infoPath := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(infoPath, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return infoPath
}

// updateRunStatus rewrites run-info.yaml with a new status (simulates task completion).
func updateRunStatus(t *testing.T, infoPath, newStatus string, exitCode int) {
	t.Helper()
	info, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		t.Fatalf("read run-info for update: %v", err)
	}
	info.Status = newStatus
	info.ExitCode = exitCode
	info.EndTime = time.Now().UTC()
	if err := storage.WriteRunInfo(infoPath, info); err != nil {
		t.Fatalf("update run-info: %v", err)
	}
}

// TestWatchEmptyTaskList verifies that runWatch returns an error when no tasks are given.
func TestWatchEmptyTaskList(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	err := runWatch(&buf, root, "my-project", nil, 5*time.Second, false)
	if err == nil {
		t.Fatal("expected error for empty task list, got nil")
	}
	if !strings.Contains(err.Error(), "--task") {
		t.Errorf("expected error to mention --task, got: %v", err)
	}
}

// TestWatchSingleCompletedTask verifies that watch exits immediately when the task is already done.
func TestWatchSingleCompletedTask(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now().UTC()

	makeWatchRun(t, root, project, task, "run-001", storage.StatusCompleted, now.Add(-time.Minute), now, 0)

	var buf bytes.Buffer
	start := time.Now()
	err := runWatch(&buf, root, project, []string{task}, 30*time.Second, false)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should exit quickly (well under 1 second), not wait the full timeout.
	if elapsed > 2*time.Second {
		t.Errorf("expected fast exit, but took %s", elapsed)
	}
	output := buf.String()
	if !strings.Contains(output, "completed") {
		t.Errorf("expected 'completed' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "All tasks complete") {
		t.Errorf("expected 'All tasks complete' in output, got:\n%s", output)
	}
}

// TestWatchSingleRunningTaskCompletesInTime verifies that watch waits for a running task to finish.
func TestWatchSingleRunningTaskCompletesInTime(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000002-bb"
	now := time.Now().UTC()

	infoPath := makeWatchRun(t, root, project, task, "run-001", storage.StatusRunning, now, time.Time{}, 0)

	// Complete the task after 150ms in a goroutine.
	go func() {
		time.Sleep(150 * time.Millisecond)
		updateRunStatus(t, infoPath, storage.StatusCompleted, 0)
	}()

	var buf bytes.Buffer
	err := runWatch(&buf, root, project, []string{task}, 10*time.Second, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "All tasks complete") {
		t.Errorf("expected 'All tasks complete' in output, got:\n%s", output)
	}
}

// TestWatchMultipleTasksAllComplete verifies that watch exits when all tasks reach a terminal state.
func TestWatchMultipleTasksAllComplete(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task1 := "task-20260101-000001-aa"
	task2 := "task-20260101-000002-bb"
	task3 := "task-20260101-000003-cc"
	now := time.Now().UTC()

	// task1: already completed
	makeWatchRun(t, root, project, task1, "run-001", storage.StatusCompleted, now.Add(-2*time.Minute), now.Add(-time.Minute), 0)

	// task2: running, will complete after 100ms
	infoPath2 := makeWatchRun(t, root, project, task2, "run-001", storage.StatusRunning, now, time.Time{}, 0)

	// task3: running, will fail after 200ms
	infoPath3 := makeWatchRun(t, root, project, task3, "run-001", storage.StatusRunning, now, time.Time{}, 0)

	go func() {
		time.Sleep(100 * time.Millisecond)
		updateRunStatus(t, infoPath2, storage.StatusCompleted, 0)
	}()
	go func() {
		time.Sleep(200 * time.Millisecond)
		updateRunStatus(t, infoPath3, storage.StatusFailed, 1)
	}()

	var buf bytes.Buffer
	err := runWatch(&buf, root, project, []string{task1, task2, task3}, 10*time.Second, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "All tasks complete") {
		t.Errorf("expected 'All tasks complete' in output, got:\n%s", output)
	}
}

// TestWatchTimeout verifies that watch exits with an error when the timeout is reached.
func TestWatchTimeout(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now().UTC()

	// Create a running task that never completes.
	makeWatchRun(t, root, project, task, "run-001", storage.StatusRunning, now, time.Time{}, 0)

	var buf bytes.Buffer
	err := runWatch(&buf, root, project, []string{task}, 200*time.Millisecond, false)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected error to mention 'timeout', got: %v", err)
	}
}

// TestWatchJSONOutput verifies that --json flag produces valid JSON with expected fields.
func TestWatchJSONOutput(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now().UTC()

	makeWatchRun(t, root, project, task, "run-001", storage.StatusCompleted, now.Add(-time.Minute), now, 0)

	var buf bytes.Buffer
	err := runWatch(&buf, root, project, []string{task}, 5*time.Second, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	type jsonPayload struct {
		Tasks []struct {
			TaskID  string  `json:"task_id"`
			Status  string  `json:"status"`
			Elapsed float64 `json:"elapsed"`
			Done    bool    `json:"done"`
		} `json:"tasks"`
		AllDone bool `json:"all_done"`
	}

	var out jsonPayload
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if !out.AllDone {
		t.Errorf("expected all_done=true, got false")
	}
	if len(out.Tasks) != 1 {
		t.Fatalf("expected 1 task in JSON, got %d", len(out.Tasks))
	}
	if out.Tasks[0].TaskID != task {
		t.Errorf("expected task_id %q, got %q", task, out.Tasks[0].TaskID)
	}
	if out.Tasks[0].Status != storage.StatusCompleted {
		t.Errorf("expected status 'completed', got %q", out.Tasks[0].Status)
	}
	if !out.Tasks[0].Done {
		t.Errorf("expected done=true")
	}
}

// TestWatchCmd_HelpText verifies that the watch command is registered and shows help.
func TestWatchCmd_HelpText(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"watch", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	for _, want := range []string{"watch", "project", "task", "timeout"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in help output, got:\n%s", want, output)
		}
	}
}
