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
	if err := runStatus(&buf, root, project, "", "", true, false); err != nil {
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
	if err := runStatus(&buf, root, project, task, "", true, false); err != nil {
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
	if err := runStatus(&buf, root, project, "", "", true, false); err != nil {
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
	if err := runStatus(&buf, root, project, "", "", false, false); err != nil {
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
	if err := runStatus(&buf, root, project, taskMain, "", true, false); err != nil {
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

func TestRunStatusJSON_StatusFilterRunning(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	now := time.Now().UTC()

	runningTask := "task-20260101-000030-aa"
	completedTask := "task-20260101-000031-bb"

	makeRunWithPID(t, root, project, runningTask, "run-001", storage.StatusRunning, now.Add(-time.Minute), -1, os.Getpid())
	makeRun(t, root, project, completedTask, "run-001", storage.StatusCompleted, now.Add(-2*time.Minute), 0)

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", "running", true, false); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	var payload statusJSONPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(payload.Tasks) != 1 {
		t.Fatalf("expected 1 filtered task, got %d", len(payload.Tasks))
	}
	if payload.Tasks[0].TaskID != runningTask {
		t.Fatalf("filtered task=%q, want %q", payload.Tasks[0].TaskID, runningTask)
	}
}

func TestRunStatusText_FilterNoMatchIncludesMessage(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000032-cc"

	makeRun(t, root, project, task, "run-001", storage.StatusCompleted, time.Now().UTC().Add(-time.Minute), 0)

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", "running", false, false); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "TASK_ID") {
		t.Fatalf("expected header in output:\n%s", output)
	}
	want := `No status rows matched --status "running" in project proj.`
	if !strings.Contains(output, want) {
		t.Fatalf("expected no-match message %q in output:\n%s", want, output)
	}
}

func TestRunStatusConcise_OutputsEquivalentFieldSet(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000033-dd"
	now := time.Now().UTC()

	makeRunWithPID(t, root, project, task, "run-001", storage.StatusRunning, now.Add(-time.Minute), -1, os.Getpid())

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", "running", false, true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if strings.Contains(output, "TASK_ID") {
		t.Fatalf("concise output must not include header:\n%s", output)
	}
	fields := strings.Split(output, "\t")
	if len(fields) != 6 {
		t.Fatalf("expected 6 concise fields, got %d: %q", len(fields), output)
	}
	if fields[0] != task {
		t.Fatalf("task_id=%q, want %q", fields[0], task)
	}
	if fields[1] != storage.StatusRunning {
		t.Fatalf("status=%q, want %q", fields[1], storage.StatusRunning)
	}
	if fields[2] != "-1" {
		t.Fatalf("exit_code=%q, want -1", fields[2])
	}
	if fields[3] != "run-001" {
		t.Fatalf("latest_run=%q, want run-001", fields[3])
	}
	if fields[4] != "false" {
		t.Fatalf("done=%q, want false", fields[4])
	}
	if fields[5] != "true" {
		t.Fatalf("pid_alive=%q, want true", fields[5])
	}
}

func TestRunStatusConcise_NoMatchPrintsMessage(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000034-ee"

	makeRun(t, root, project, task, "run-001", storage.StatusCompleted, time.Now().UTC().Add(-time.Minute), 0)

	var buf bytes.Buffer
	if err := runStatus(&buf, root, project, "", "running", false, true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := `No status rows matched --status "running" in project proj.`
	if got != want {
		t.Fatalf("concise no-match output=%q, want %q", got, want)
	}
}

func TestRunStatusRejectsConciseAndJSON(t *testing.T) {
	err := runStatus(&bytes.Buffer{}, t.TempDir(), "proj", "", "", true, true)
	if err == nil {
		t.Fatalf("expected error when --concise and --json are both set")
	}
	if !strings.Contains(err.Error(), "--concise cannot be used with --json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStatusJSON_ActivitySignals(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000035-ff"
	start := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	now := start.Add(40 * time.Minute)

	runDir := makeRunWithPID(t, root, project, task, "run-001", storage.StatusRunning, start, -1, os.Getpid())
	outputPath := filepath.Join(runDir, "output.md")
	if err := os.WriteFile(outputPath, []byte("working"), 0o644); err != nil {
		t.Fatalf("write output.md: %v", err)
	}
	outputAt := start.Add(15 * time.Minute)
	if err := os.Chtimes(outputPath, outputAt, outputAt); err != nil {
		t.Fatalf("chtimes output.md: %v", err)
	}

	taskDir := filepath.Join(root, project, task)
	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	meaningfulAt := start.Add(5 * time.Minute)
	writeTaskBusMessage(t, busPath, project, task, "FACT", "implemented status parser", meaningfulAt)
	writeTaskBusMessage(t, busPath, project, task, "PROGRESS", "still exploring edge cases", start.Add(20*time.Minute))

	var buf bytes.Buffer
	err := runStatusWithOptions(
		&buf,
		root,
		project,
		task,
		"",
		true,
		false,
		activityOptions{
			Enabled:    true,
			DriftAfter: 20 * time.Minute,
			Now: func() time.Time {
				return now
			},
		},
	)
	if err != nil {
		t.Fatalf("runStatusWithOptions: %v", err)
	}

	var payload statusJSONPayload
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(payload.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(payload.Tasks))
	}

	row := payload.Tasks[0]
	if row.Activity == nil {
		t.Fatalf("expected activity payload")
	}
	if row.Activity.LatestBusMessage == nil {
		t.Fatalf("expected latest_bus_message")
	}
	if row.Activity.LatestBusMessage.Type != "PROGRESS" {
		t.Fatalf("latest bus type=%q, want PROGRESS", row.Activity.LatestBusMessage.Type)
	}
	if row.Activity.LatestOutputActivityAt == nil || *row.Activity.LatestOutputActivityAt != outputAt.Format(time.RFC3339) {
		t.Fatalf("latest_output_activity_at=%v, want %s", row.Activity.LatestOutputActivityAt, outputAt.Format(time.RFC3339))
	}
	if row.Activity.LastMeaningfulSignalAt == nil || *row.Activity.LastMeaningfulSignalAt != meaningfulAt.Format(time.RFC3339) {
		t.Fatalf("last_meaningful_signal_at=%v, want %s", row.Activity.LastMeaningfulSignalAt, meaningfulAt.Format(time.RFC3339))
	}
	if row.Activity.MeaningfulSignalAgeSeconds == nil {
		t.Fatalf("expected meaningful_signal_age_seconds")
	}
	if got, want := *row.Activity.MeaningfulSignalAgeSeconds, int64(35*60); got != want {
		t.Fatalf("meaningful_signal_age_seconds=%d, want %d", got, want)
	}
	if !row.Activity.AnalysisDriftRisk {
		t.Fatalf("expected analysis_drift_risk=true")
	}
}

func TestRunStatusText_ActivityColumns(t *testing.T) {
	root := t.TempDir()
	project := "proj"
	task := "task-20260101-000036-gg"
	start := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)

	makeRunWithPID(t, root, project, task, "run-001", storage.StatusRunning, start, -1, os.Getpid())
	taskDir := filepath.Join(root, project, task)
	writeTaskBusMessage(t, filepath.Join(taskDir, "TASK-MESSAGE-BUS.md"), project, task, "PROGRESS", "analyzing", start.Add(2*time.Minute))

	var buf bytes.Buffer
	if err := runStatusWithOptions(
		&buf,
		root,
		project,
		task,
		"",
		false,
		false,
		activityOptions{Enabled: true, DriftAfter: 20 * time.Minute, Now: func() time.Time { return start.Add(3 * time.Minute) }},
	); err != nil {
		t.Fatalf("runStatusWithOptions: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"LAST_BUS", "LAST_OUTPUT", "MEANINGFUL_AGE", "DRIFT_RISK"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}
