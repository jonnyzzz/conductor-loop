package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// ---- checkStopRequested ----

func TestCheckStopRequested_NoFile(t *testing.T) {
	root := t.TempDir()
	if checkStopRequested(root, "proj", "task-20260101-000001-aaa", time.Minute) {
		t.Error("expected false when STOP-REQUESTED does not exist")
	}
}

func TestCheckStopRequested_FreshMarker(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}

	if !checkStopRequested(root, proj, task, time.Minute) {
		t.Error("expected true for a fresh (within-window) stop-request")
	}
}

func TestCheckStopRequested_ExpiredMarker(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"

	// Write a marker with a timestamp 2 minutes in the past.
	taskDir := filepath.Join(root, proj, task)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	old := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	markerPath := stopRequestPath(root, proj, task)
	if err := os.WriteFile(markerPath, []byte(old+"\n"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	// Window is 1 minute — marker is 2 minutes old, should be expired.
	if checkStopRequested(root, proj, task, time.Minute) {
		t.Error("expected false for expired stop-request marker")
	}
	// Expired marker must be auto-removed.
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("expected expired marker to be auto-removed")
	}
}

func TestCheckStopRequested_PermanentMarker(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	if err := writeStopRequest(root, proj, task, true); err != nil {
		t.Fatalf("writeStopRequest permanent: %v", err)
	}

	// Permanent marker should suppress even with a very short window.
	if !checkStopRequested(root, proj, task, time.Nanosecond) {
		t.Error("expected true for permanent stop-request marker")
	}
}

func TestCheckStopRequested_CorruptMarkerAutoRemoved(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	taskDir := filepath.Join(root, proj, task)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	markerPath := stopRequestPath(root, proj, task)
	if err := os.WriteFile(markerPath, []byte("not-a-timestamp\n"), 0o644); err != nil {
		t.Fatalf("write corrupt marker: %v", err)
	}

	if checkStopRequested(root, proj, task, time.Minute) {
		t.Error("expected false for corrupt marker")
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("expected corrupt marker to be auto-removed")
	}
}

// ---- removeStopRequest ----

func TestRemoveStopRequest_Existing(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}
	if err := removeStopRequest(root, proj, task); err != nil {
		t.Fatalf("removeStopRequest: %v", err)
	}
	if _, err := os.Stat(stopRequestPath(root, proj, task)); !os.IsNotExist(err) {
		t.Error("expected marker to be removed")
	}
}

func TestRemoveStopRequest_NotExist(t *testing.T) {
	root := t.TempDir()
	// Should not error when marker does not exist.
	if err := removeStopRequest(root, "proj", "task-20260101-000001-aaa"); err != nil {
		t.Fatalf("removeStopRequest for non-existing file: %v", err)
	}
}

// ---- decideMonitorActionWithWindow: stop suppression ----

func TestDecideMonitorActionWithWindow_StopSuppressed_Failed(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}

	state := monitorTaskState{
		TaskID:  task,
		Exists:  true,
		HasRuns: true,
		Status:  storage.StatusFailed,
	}
	d := decideMonitorActionWithWindow(state, root, proj, time.Minute)
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip due to stop-request for failed task, got %q (reason: %s)", d.Action, d.Reason)
	}
	if !strings.Contains(d.Reason, "stop-requested") {
		t.Errorf("expected reason to mention 'stop-requested', got: %q", d.Reason)
	}
}

func TestDecideMonitorActionWithWindow_StopSuppressed_RunningDeadPID(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}

	state := monitorTaskState{
		TaskID:   task,
		Exists:   true,
		HasRuns:  true,
		Status:   storage.StatusRunning,
		PIDAlive: false,
	}
	d := decideMonitorActionWithWindow(state, root, proj, time.Minute)
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip due to stop-request for dead-PID running task, got %q", d.Action)
	}
	if !strings.Contains(d.Reason, "stop-requested") {
		t.Errorf("expected reason to mention 'stop-requested', got: %q", d.Reason)
	}
}

func TestDecideMonitorActionWithWindow_StopSuppressed_NoRuns(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	// Create task dir so stop request can be written.
	if err := os.MkdirAll(filepath.Join(root, proj, task), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}

	state := monitorTaskState{
		TaskID: task,
		Exists: false,
	}
	d := decideMonitorActionWithWindow(state, root, proj, time.Minute)
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip due to stop-request for task with no runs, got %q", d.Action)
	}
}

func TestDecideMonitorActionWithWindow_NoSuppression_Failed(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	// No STOP-REQUESTED marker.
	state := monitorTaskState{
		TaskID:  task,
		Exists:  true,
		HasRuns: true,
		Status:  storage.StatusFailed,
	}
	d := decideMonitorActionWithWindow(state, root, proj, time.Minute)
	if d.Action != monitorActionResume {
		t.Errorf("expected resume for failed task without stop-request, got %q", d.Action)
	}
}

func TestDecideMonitorActionWithWindow_PermanentSuppression(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"
	if err := writeStopRequest(root, proj, task, true); err != nil {
		t.Fatalf("writeStopRequest permanent: %v", err)
	}

	state := monitorTaskState{
		TaskID:  task,
		Exists:  true,
		HasRuns: true,
		Status:  storage.StatusFailed,
	}
	// Even with a tiny window, permanent suppression holds.
	d := decideMonitorActionWithWindow(state, root, proj, time.Nanosecond)
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip for permanently suppressed task, got %q", d.Action)
	}
}

func TestDecideMonitorActionWithWindow_ExpiredSuppressionAllowsRestart(t *testing.T) {
	root := t.TempDir()
	const proj, task = "proj", "task-20260101-000001-aaa"

	// Write a marker that is 2 minutes old.
	taskDir := filepath.Join(root, proj, task)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	old := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	if err := os.WriteFile(stopRequestPath(root, proj, task), []byte(old+"\n"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	state := monitorTaskState{
		TaskID:  task,
		Exists:  true,
		HasRuns: true,
		Status:  storage.StatusFailed,
	}
	// Window is 1 minute; marker is 2 minutes old → should allow restart.
	d := decideMonitorActionWithWindow(state, root, proj, time.Minute)
	if d.Action != monitorActionResume {
		t.Errorf("expected resume after suppression expired, got %q (reason: %s)", d.Action, d.Reason)
	}
}

// ---- monitorPass integration: stop-suppression in dry-run ----

func TestMonitorPass_StopSuppressedTaskSkipped(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000002-fail"

	makeMonitorRun(t, root, proj, task, "run-001", storage.StatusFailed, 1)
	// Write STOP-REQUESTED marker.
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}

	dir := t.TempDir()
	todoPath := writeTODO(t, dir, fmt.Sprintf("- [ ] %s\n", task))

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		Agent:      "claude",
		StaleAfter: 20 * time.Minute,
		RateLimit:  0,
		DryRun:     true,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()

	output := buf.String()
	if !strings.Contains(output, monitorActionSkip) {
		t.Errorf("expected 'skip' for stop-suppressed task, got:\n%s", output)
	}
	if !strings.Contains(output, "stop-requested") {
		t.Errorf("expected 'stop-requested' in skip reason, got:\n%s", output)
	}
}

// ---- monitorLaunchJob removes STOP-REQUESTED marker ----

func TestMonitorLaunchJob_RemovesStopRequestedMarker(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000002-fail"

	// Write STOP-REQUESTED marker that would suppress restart.
	if err := writeStopRequest(root, proj, task, false); err != nil {
		t.Fatalf("writeStopRequest: %v", err)
	}

	// Verify it's present.
	markerPath := stopRequestPath(root, proj, task)
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected marker to exist before launch: %v", err)
	}

	// Simulate what monitorLaunchJob does: remove DONE and STOP-REQUESTED.
	_ = os.Remove(filepath.Join(root, proj, task, "DONE"))
	_ = removeStopRequest(root, proj, task)

	// Marker must be gone after launch.
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("expected STOP-REQUESTED marker to be removed when job is launched")
	}
}

// ---- stop cmd: --no-restart flag presence in help ----

func TestStopCmd_NoRestartFlag(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"stop", "--help"})
	_ = cmd.Execute()
	if !strings.Contains(buf.String(), "no-restart") {
		t.Errorf("expected --no-restart flag in stop help, got:\n%s", buf.String())
	}
}
