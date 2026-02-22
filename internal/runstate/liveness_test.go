package runstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestReadRunInfoWithClock_ReconcilesDeadRunningPID(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run-info.yaml")
	start := time.Now().Add(-2 * time.Minute).UTC()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: start,
		PID:       99999999,
		PGID:      99999999,
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	fixedNow := start.Add(30 * time.Second).UTC()
	got, err := ReadRunInfoWithClock(path, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("ReadRunInfoWithClock: %v", err)
	}
	if got.Status != storage.StatusFailed {
		t.Fatalf("status: got %q want %q", got.Status, storage.StatusFailed)
	}
	if got.EndTime.IsZero() {
		t.Fatalf("expected non-zero end_time after reconciliation")
	}
	if got.ErrorSummary == "" {
		t.Fatalf("expected non-empty error summary after reconciliation")
	}

	reloaded, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("ReadRunInfo: %v", err)
	}
	if reloaded.Status != storage.StatusFailed {
		t.Fatalf("persisted status: got %q want %q", reloaded.Status, storage.StatusFailed)
	}
}

func TestReadRunInfoWithClock_LeavesRunningWithoutPID(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run-info.yaml")
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	got, err := ReadRunInfoWithClock(path, time.Now)
	if err != nil {
		t.Fatalf("ReadRunInfoWithClock: %v", err)
	}
	if got.Status != storage.StatusRunning {
		t.Fatalf("status: got %q want %q", got.Status, storage.StatusRunning)
	}
}

func TestReadRunInfoWithClock_LeavesAlivePIDRunning(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run-info.yaml")
	pid := os.Getpid()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: time.Now().UTC(),
		PID:       pid,
		PGID:      pid,
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	got, err := ReadRunInfoWithClock(path, time.Now)
	if err != nil {
		t.Fatalf("ReadRunInfoWithClock: %v", err)
	}
	if got.Status != storage.StatusRunning {
		t.Fatalf("status: got %q want %q", got.Status, storage.StatusRunning)
	}
}

func TestReadRunInfoWithClock_ReconcilesDeadRunningPIDToCompletedWhenDONEExists(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}
	path := filepath.Join(runDir, "run-info.yaml")
	start := time.Now().Add(-2 * time.Minute).UTC()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: start,
		PID:       99999999,
		PGID:      99999999,
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	fixedNow := start.Add(45 * time.Second).UTC()
	got, err := ReadRunInfoWithClock(path, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("ReadRunInfoWithClock: %v", err)
	}
	if got.Status != storage.StatusCompleted {
		t.Fatalf("status: got %q want %q", got.Status, storage.StatusCompleted)
	}
	if got.ExitCode != 0 {
		t.Fatalf("exit_code: got %d want 0", got.ExitCode)
	}
	if got.EndTime.IsZero() {
		t.Fatalf("expected non-zero end_time after reconciliation")
	}
}

func TestReadRunInfoWithClock_HealsPreviouslyReconciledFailedRunWhenDONEExists(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}
	path := filepath.Join(runDir, "run-info.yaml")
	start := time.Now().Add(-2 * time.Minute).UTC()
	info := &storage.RunInfo{
		RunID:        "run-1",
		ProjectID:    "project",
		TaskID:       "task",
		Status:       storage.StatusFailed,
		ExitCode:     -1,
		StartTime:    start,
		EndTime:      start.Add(20 * time.Second),
		ErrorSummary: "reconciled stale running status: process is not alive",
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	got, err := ReadRunInfoWithClock(path, time.Now)
	if err != nil {
		t.Fatalf("ReadRunInfoWithClock: %v", err)
	}
	if got.Status != storage.StatusCompleted {
		t.Fatalf("status: got %q want %q", got.Status, storage.StatusCompleted)
	}
	if got.ExitCode != 0 {
		t.Fatalf("exit_code: got %d want 0", got.ExitCode)
	}
}
