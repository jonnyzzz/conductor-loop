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

// makeRunWithEnd creates a run directory with a run-info.yaml including EndTime.
func makeRunWithEnd(t *testing.T, root, project, task, runID, status string, exitCode int, started, ended time.Time) {
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
		ExitCode:  exitCode,
		StartTime: started,
		EndTime:   ended,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
}

func TestListProjects(t *testing.T) {
	root := t.TempDir()

	for _, proj := range []string{"alpha", "beta", "gamma"} {
		if err := os.MkdirAll(filepath.Join(root, proj), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// create a file — should be ignored
	if err := os.WriteFile(filepath.Join(root, "not-a-dir.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := listProjects(&buf, root, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 projects, got %d: %v", len(lines), lines)
	}
	for i, want := range []string{"alpha", "beta", "gamma"} {
		if lines[i] != want {
			t.Errorf("line %d: got %q, want %q", i, lines[i], want)
		}
	}
}

func TestListProjectsJSON(t *testing.T) {
	root := t.TempDir()
	for _, proj := range []string{"proj-a", "proj-b"} {
		if err := os.MkdirAll(filepath.Join(root, proj), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	if err := listProjects(&buf, root, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string][]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(out["projects"]) != 2 {
		t.Errorf("expected 2 projects in JSON, got %d", len(out["projects"]))
	}
}

func TestListProjectsMissingRoot(t *testing.T) {
	err := listProjects(&bytes.Buffer{}, "/nonexistent/path/12345", false)
	if err == nil {
		t.Fatal("expected error for missing root, got nil")
	}
}

func TestListTasks(t *testing.T) {
	root := t.TempDir()
	project := "my-project"
	now := time.Now().UTC().Truncate(time.Second)

	// task-1: 2 runs, latest completed, DONE file
	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusCompleted, now.Add(-10*time.Minute), 0)
	makeRun(t, root, project, "task-20260101-000001-aa", "run-002", storage.StatusCompleted, now.Add(-4*time.Minute), 0)
	if err := os.WriteFile(filepath.Join(root, project, "task-20260101-000001-aa", "DONE"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// task-2: 1 run, failed, no DONE
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusFailed, now.Add(-20*time.Minute), 1)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("missing task-20260101-000001-aa in output")
	}
	if !strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("missing task-20260101-000002-bb in output")
	}
	if !strings.Contains(output, "completed") {
		t.Error("missing 'completed' status in output")
	}
	if !strings.Contains(output, "failed") {
		t.Error("missing 'failed' status in output")
	}
}

func TestListTasksEmpty(t *testing.T) {
	root := t.TempDir()
	project := "empty-project"
	if err := os.MkdirAll(filepath.Join(root, project), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error for empty project: %v", err)
	}

	// Should just have header, no data rows
	output := buf.String()
	if !strings.Contains(output, "TASK_ID") {
		t.Error("missing header in output")
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (header only), got %d: %v", len(lines), lines)
	}
}

func TestListTasksMissingProject(t *testing.T) {
	root := t.TempDir()
	err := listTasks(&bytes.Buffer{}, root, "nonexistent-project", "", false)
	if err == nil {
		t.Fatal("expected error for missing project directory, got nil")
	}
}

func TestListTasksDoneDetection(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now()

	makeRun(t, root, project, task, "run-001", storage.StatusCompleted, now.Add(-time.Minute), 0)

	// Without DONE file — check the data row is NOT marked DONE
	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines[1:] {
		if strings.Contains(line, task) {
			if strings.Contains(line, "DONE") {
				t.Errorf("task should not show DONE when no DONE file: %q", line)
			}
		}
	}

	// Add DONE file and check it shows up
	if err := os.WriteFile(filepath.Join(root, project, task, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines = strings.Split(strings.TrimSpace(buf.String()), "\n")
	found := false
	for _, line := range lines[1:] {
		if strings.Contains(line, task) {
			if !strings.Contains(line, "DONE") {
				t.Errorf("task line should contain DONE: %q", line)
			}
			found = true
		}
	}
	if !found {
		t.Error("task row not found in output")
	}
}

func TestListTasksJSON(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusCompleted, now.Add(-time.Minute), 0)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string][]taskRow
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(out["tasks"]) != 1 {
		t.Errorf("expected 1 task in JSON, got %d", len(out["tasks"]))
	}
	if out["tasks"][0].TaskID != "task-20260101-000001-aa" {
		t.Errorf("wrong task_id: %q", out["tasks"][0].TaskID)
	}
	if out["tasks"][0].Runs != 1 {
		t.Errorf("expected 1 run, got %d", out["tasks"][0].Runs)
	}
	if out["tasks"][0].LatestStatus != storage.StatusCompleted {
		t.Errorf("expected 'completed', got %q", out["tasks"][0].LatestStatus)
	}
}

func TestListRuns(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now().UTC().Truncate(time.Second)

	makeRunWithEnd(t, root, project, task, "run-001", storage.StatusCompleted, 0, now.Add(-5*time.Minute), now.Add(-4*time.Minute))
	makeRunWithEnd(t, root, project, task, "run-002", storage.StatusFailed, 1, now.Add(-3*time.Minute), now.Add(-2*time.Minute))
	makeRunWithEnd(t, root, project, task, "run-003", storage.StatusRunning, -1, now.Add(-time.Minute), time.Time{})

	var buf bytes.Buffer
	if err := listRuns(&buf, root, project, task, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"run-001", "run-002", "run-003", "completed", "failed", "running"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestListRunsMissingTask(t *testing.T) {
	root := t.TempDir()
	err := listRuns(&bytes.Buffer{}, root, "proj", "task-nonexistent", false)
	if err == nil {
		t.Fatal("expected error for missing task, got nil")
	}
}

func TestListRunsJSON(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now().UTC()

	makeRunWithEnd(t, root, project, task, "run-001", storage.StatusCompleted, 0, now.Add(-time.Minute), now)

	var buf bytes.Buffer
	if err := listRuns(&buf, root, project, task, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string][]runRow
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(out["runs"]) != 1 {
		t.Errorf("expected 1 run in JSON, got %d", len(out["runs"]))
	}
	if out["runs"][0].RunID != "run-001" {
		t.Errorf("wrong run_id: %q", out["runs"][0].RunID)
	}
	if out["runs"][0].Status != storage.StatusCompleted {
		t.Errorf("expected 'completed', got %q", out["runs"][0].Status)
	}
}

func TestListRunsDuration(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-65 * time.Second)

	makeRunWithEnd(t, root, project, task, "run-001", storage.StatusCompleted, 0, start, now)

	var buf bytes.Buffer
	if err := listRuns(&buf, root, project, task, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1m5s") {
		t.Errorf("expected duration 1m5s in output:\n%s", output)
	}
}

func TestListTasks_EmptyRunsNoDONE(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	taskDir := filepath.Join(root, project, task)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "unknown") {
		t.Errorf("should not show 'unknown' for task with no runs and no DONE file; got:\n%s", output)
	}
	// The tabwriter formats with spaces; check the data row has "-" for LATEST_STATUS
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + data), got %d", len(lines))
	}
	// Data line should contain "  -  " (spaces around "-" for LATEST_STATUS and DONE columns)
	if !strings.Contains(lines[1], "  -  ") {
		t.Errorf("expected '-' status in data line; got: %q", lines[1])
	}
}

func TestListTasks_EmptyRunsWithDONE(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000001-aa"
	taskDir := filepath.Join(root, project, task)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "unknown") {
		t.Errorf("should not show 'unknown' for done task with no runs; got:\n%s", output)
	}
	if !strings.Contains(output, "done") {
		t.Errorf("expected 'done' status for task with DONE file and no runs; got:\n%s", output)
	}
	if !strings.Contains(output, "DONE") {
		t.Errorf("expected 'DONE' in DONE column; got:\n%s", output)
	}
}

func TestListTasks_LastActivityColumn(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC().Truncate(time.Second)

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusCompleted, now.Add(-time.Minute), 0)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LAST_ACTIVITY") {
		t.Errorf("expected LAST_ACTIVITY header in output:\n%s", output)
	}
}

func TestListTasksJSON_LastActivity(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusCompleted, now.Add(-time.Minute), 0)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string][]taskRow
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(out["tasks"]) != 1 {
		t.Fatalf("expected 1 task in JSON, got %d", len(out["tasks"]))
	}
	if out["tasks"][0].LastActivity == "" {
		t.Error("expected non-empty last_activity field in JSON output")
	}
	// Verify it parses as RFC3339
	if _, err := time.Parse(time.RFC3339, out["tasks"][0].LastActivity); err != nil {
		t.Errorf("last_activity %q is not valid RFC3339: %v", out["tasks"][0].LastActivity, err)
	}
}

func TestListRequiresProjectForTask(t *testing.T) {
	var buf bytes.Buffer
	err := runList(&buf, "./runs", "", "some-task", "", false)
	if err == nil {
		t.Fatal("expected error when --task given without --project")
	}
	if !strings.Contains(err.Error(), "--task requires --project") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListTasksStatusRunning(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusRunning, now.Add(-time.Minute), -1)
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusCompleted, now.Add(-2*time.Minute), 0)
	makeRun(t, root, project, "task-20260101-000003-cc", "run-001", storage.StatusFailed, now.Add(-3*time.Minute), 1)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "running", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("running task should appear in output")
	}
	if strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("completed task should not appear with --status running")
	}
	if strings.Contains(output, "task-20260101-000003-cc") {
		t.Error("failed task should not appear with --status running")
	}
}

func TestListTasksStatusActive(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusRunning, now.Add(-time.Minute), -1)
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusCompleted, now.Add(-2*time.Minute), 0)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "active", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("running task should appear with --status active")
	}
	if strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("completed task should not appear with --status active")
	}
}

func TestListTasksStatusDone(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusCompleted, now.Add(-time.Minute), 0)
	if err := os.WriteFile(filepath.Join(root, project, "task-20260101-000001-aa", "DONE"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusRunning, now.Add(-2*time.Minute), -1)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "done", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("done task should appear with --status done")
	}
	if strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("running task should not appear with --status done")
	}
}

func TestListTasksStatusFailed(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusFailed, now.Add(-time.Minute), 1)
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusCompleted, now.Add(-2*time.Minute), 0)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "failed", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("failed task should appear with --status failed")
	}
	if strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("completed task should not appear with --status failed")
	}
}

func TestListTasksStatusEmpty(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusRunning, now.Add(-time.Minute), -1)
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusFailed, now.Add(-2*time.Minute), 1)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("task-1 should appear with empty --status")
	}
	if !strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("task-2 should appear with empty --status")
	}
}

func TestListTasksStatusInvalid(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	makeRun(t, root, project, "task-20260101-000001-aa", "run-001", storage.StatusRunning, now.Add(-time.Minute), -1)
	makeRun(t, root, project, "task-20260101-000002-bb", "run-001", storage.StatusFailed, now.Add(-2*time.Minute), 1)

	var buf bytes.Buffer
	if err := listTasks(&buf, root, project, "bogus-status", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "task-20260101-000001-aa") {
		t.Error("task-1 should appear with unknown --status (graceful degradation)")
	}
	if !strings.Contains(output, "task-20260101-000002-bb") {
		t.Error("task-2 should appear with unknown --status (graceful degradation)")
	}
}
