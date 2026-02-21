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
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
)

type statusJSONPayload struct {
	Tasks []statusRow `json:"tasks"`
}

func findStatusRow(t *testing.T, rows []statusRow, taskID string) statusRow {
	t.Helper()
	for _, row := range rows {
		if row.TaskID == taskID {
			return row
		}
	}
	t.Fatalf("task %q not found in status rows", taskID)
	return statusRow{}
}

func TestRunStatusJSON_IncludesRequiredFields(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	completedTask := "task-20260101-000001-aa"
	runningTask := "task-20260101-000002-bb"

	makeRun(t, root, project, completedTask, "run-001", storage.StatusCompleted, now.Add(-2*time.Minute), 0)
	if err := os.WriteFile(filepath.Join(root, project, completedTask, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}
	makeRunWithPID(t, root, project, runningTask, "run-002", storage.StatusRunning, now.Add(-time.Minute), -1, os.Getpid())

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	var payload statusJSONPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(payload.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(payload.Tasks))
	}

	completed := findStatusRow(t, payload.Tasks, completedTask)
	if completed.Status != storage.StatusCompleted {
		t.Fatalf("completed status=%q, want %q", completed.Status, storage.StatusCompleted)
	}
	if completed.ExitCode == nil || *completed.ExitCode != 0 {
		t.Fatalf("completed exit_code=%v, want 0", completed.ExitCode)
	}
	if completed.LatestRun != "run-001" {
		t.Fatalf("completed latest_run=%q, want run-001", completed.LatestRun)
	}
	if !completed.Done {
		t.Fatalf("completed done=false, want true")
	}
	if completed.PIDAlive == nil || *completed.PIDAlive {
		t.Fatalf("completed pid_alive=%v, want false", completed.PIDAlive)
	}

	running := findStatusRow(t, payload.Tasks, runningTask)
	if running.Status != storage.StatusRunning {
		t.Fatalf("running status=%q, want %q", running.Status, storage.StatusRunning)
	}
	if running.ExitCode == nil || *running.ExitCode != -1 {
		t.Fatalf("running exit_code=%v, want -1", running.ExitCode)
	}
	if running.LatestRun != "run-002" {
		t.Fatalf("running latest_run=%q, want run-002", running.LatestRun)
	}
	if running.Done {
		t.Fatalf("running done=true, want false")
	}
	if running.PIDAlive == nil || !*running.PIDAlive {
		t.Fatalf("running pid_alive=%v, want true", running.PIDAlive)
	}
}

func TestRunStatusJSON_ReconcilesStaleRunningPID(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000003-cc"
	start := time.Now().Add(-time.Minute).UTC()
	runDir := makeRunWithPID(t, root, project, task, "run-001", storage.StatusRunning, start, -1, 99999999)
	infoPath := filepath.Join(runDir, "run-info.yaml")

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, task, true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	var payload statusJSONPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(payload.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(payload.Tasks))
	}
	row := payload.Tasks[0]
	if row.Status != storage.StatusFailed {
		t.Fatalf("status=%q, want %q", row.Status, storage.StatusFailed)
	}
	if row.ExitCode == nil || *row.ExitCode != -1 {
		t.Fatalf("exit_code=%v, want -1", row.ExitCode)
	}
	if row.PIDAlive == nil || *row.PIDAlive {
		t.Fatalf("pid_alive=%v, want false", row.PIDAlive)
	}
	if row.LatestRun != "run-001" {
		t.Fatalf("latest_run=%q, want run-001", row.LatestRun)
	}

	reloaded, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		t.Fatalf("ReadRunInfo: %v", err)
	}
	if reloaded.Status != storage.StatusFailed {
		t.Fatalf("reloaded status=%q, want %q", reloaded.Status, storage.StatusFailed)
	}
	if reloaded.EndTime.IsZero() {
		t.Fatalf("expected end_time to be set after reconciliation")
	}
}

func TestRunStatusJSON_NoRunsAndDoneSemantics(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	doneTask := "task-20260101-000004-dd"
	pendingTask := "task-20260101-000005-ee"

	if err := os.MkdirAll(filepath.Join(root, project, doneTask), 0o755); err != nil {
		t.Fatalf("mkdir done task: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, project, pendingTask), 0o755); err != nil {
		t.Fatalf("mkdir pending task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, project, doneTask, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	var payload statusJSONPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(payload.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(payload.Tasks))
	}

	doneRow := findStatusRow(t, payload.Tasks, doneTask)
	if doneRow.Status != "done" {
		t.Fatalf("done task status=%q, want done", doneRow.Status)
	}
	if doneRow.ExitCode != nil {
		t.Fatalf("done task exit_code=%v, want nil", doneRow.ExitCode)
	}
	if doneRow.LatestRun != "" {
		t.Fatalf("done task latest_run=%q, want empty", doneRow.LatestRun)
	}
	if !doneRow.Done {
		t.Fatalf("done task done=false, want true")
	}
	if doneRow.PIDAlive != nil {
		t.Fatalf("done task pid_alive=%v, want nil", doneRow.PIDAlive)
	}

	pendingRow := findStatusRow(t, payload.Tasks, pendingTask)
	if pendingRow.Status != "-" {
		t.Fatalf("pending task status=%q, want '-'", pendingRow.Status)
	}
	if pendingRow.ExitCode != nil {
		t.Fatalf("pending task exit_code=%v, want nil", pendingRow.ExitCode)
	}
	if pendingRow.LatestRun != "" {
		t.Fatalf("pending task latest_run=%q, want empty", pendingRow.LatestRun)
	}
	if pendingRow.Done {
		t.Fatalf("pending task done=true, want false")
	}
	if pendingRow.PIDAlive != nil {
		t.Fatalf("pending task pid_alive=%v, want nil", pendingRow.PIDAlive)
	}
}

func TestRunStatusText_CoherentWithFields(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000006-ff"
	now := time.Now().UTC()

	makeRun(t, root, project, task, "run-001", storage.StatusFailed, now.Add(-time.Minute), 3)

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", false); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"TASK_ID", "STATUS", "EXIT_CODE", "LATEST_RUN", "DONE", "PID_ALIVE"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
	for _, want := range []string{task, "failed", "3", "run-001", "false", "false"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestRunStatusJSON_BlockedByDependencies(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	taskMain := "task-20260101-000020-aa"
	taskDep := "task-20260101-000021-bb"

	taskMainDir := filepath.Join(root, project, taskMain)
	taskDepDir := filepath.Join(root, project, taskDep)
	if err := os.MkdirAll(taskMainDir, 0o755); err != nil {
		t.Fatalf("mkdir task-main: %v", err)
	}
	if err := os.MkdirAll(taskDepDir, 0o755); err != nil {
		t.Fatalf("mkdir task-dep: %v", err)
	}
	if err := taskdeps.WriteDependsOn(taskMainDir, []string{taskDep}); err != nil {
		t.Fatalf("WriteDependsOn: %v", err)
	}

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, taskMain, true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	var payload statusJSONPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(payload.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(payload.Tasks))
	}
	row := payload.Tasks[0]
	if row.Status != "blocked" {
		t.Fatalf("status=%q, want blocked", row.Status)
	}
	if len(row.BlockedBy) != 1 || row.BlockedBy[0] != taskDep {
		t.Fatalf("blocked_by=%v, want [%s]", row.BlockedBy, taskDep)
	}
}
