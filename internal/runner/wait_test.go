package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestProcessGroupIDInvalid(t *testing.T) {
	if _, err := ProcessGroupID(0); err == nil {
		t.Fatalf("expected error for invalid pid")
	}
}

func TestIsProcessGroupAliveInvalid(t *testing.T) {
	if _, err := isProcessGroupAlive(0); err == nil {
		t.Fatalf("expected error for invalid pgid")
	}
}

func TestIsProcessGroupAliveCurrent(t *testing.T) {
	pgid, err := ProcessGroupID(os.Getpid())
	if err != nil {
		t.Fatalf("ProcessGroupID: %v", err)
	}
	alive, err := isProcessGroupAlive(pgid)
	if err != nil {
		t.Fatalf("isProcessGroupAlive: %v", err)
	}
	if !alive {
		t.Fatalf("expected current process group to be alive")
	}
}

func TestFindActiveChildrenMissingRuns(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "task")
	children, err := FindActiveChildren(taskDir)
	if err != nil {
		t.Fatalf("FindActiveChildren: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("expected no children, got %d", len(children))
	}
}

func TestFindActiveChildrenReadDirError(t *testing.T) {
	taskDir := t.TempDir()
	runsPath := filepath.Join(taskDir, "runs")
	if err := os.WriteFile(runsPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write runs file: %v", err)
	}
	if _, err := FindActiveChildren(taskDir); err == nil {
		t.Fatalf("expected error for invalid runs directory")
	}
}

func TestFindActiveChildrenSkipsCompleted(t *testing.T) {
	taskDir := t.TempDir()
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:       "run-1",
		ProjectID:   "project",
		TaskID:      "task",
		ParentRunID: "parent",
		EndTime:     time.Now().UTC(),
		Status:      storage.StatusCompleted,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	children, err := FindActiveChildren(taskDir)
	if err != nil {
		t.Fatalf("FindActiveChildren: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("expected no children, got %d", len(children))
	}
}

func TestFindActiveChildrenActive(t *testing.T) {
	taskDir := t.TempDir()
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	pgid, err := ProcessGroupID(os.Getpid())
	if err != nil {
		t.Fatalf("ProcessGroupID: %v", err)
	}
	info := &storage.RunInfo{
		RunID:       "run-1",
		ProjectID:   "project",
		TaskID:      "task",
		ParentRunID: "parent",
		PID:         os.Getpid(),
		PGID:        pgid,
		Status:      storage.StatusRunning,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	children, err := FindActiveChildren(taskDir)
	if err != nil {
		t.Fatalf("FindActiveChildren: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
}

func TestFindActiveChildrenMarkFailed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group check differs on windows")
	}
	taskDir := t.TempDir()
	runDir := filepath.Join(taskDir, "runs", "run-2")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:       "run-2",
		ProjectID:   "project",
		TaskID:      "task",
		ParentRunID: "parent",
		PID:         12345,
		PGID:        999999,
		Status:      storage.StatusRunning,
	}
	path := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	children, err := FindActiveChildren(taskDir)
	if err != nil {
		t.Fatalf("FindActiveChildren: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("expected no children, got %d", len(children))
	}
	updated, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if updated.Status != storage.StatusFailed || updated.EndTime.IsZero() {
		t.Fatalf("expected failed run info, got %+v", updated)
	}
}

func TestWaitForChildrenTimeout(t *testing.T) {
	pgid, err := ProcessGroupID(os.Getpid())
	if err != nil {
		t.Fatalf("ProcessGroupID: %v", err)
	}
	children := []ChildProcess{{RunID: "run-1", PGID: pgid}}
	remaining, err := WaitForChildren(context.Background(), children, 20*time.Millisecond, 5*time.Millisecond)
	if err != ErrChildWaitTimeout {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if len(remaining) == 0 {
		t.Fatalf("expected remaining children")
	}
}

func TestWaitForChildrenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	children := []ChildProcess{{RunID: "run-1", PGID: 1}}
	remaining, err := WaitForChildren(ctx, children, time.Second, 10*time.Millisecond)
	if err == nil {
		t.Fatalf("expected context error")
	}
	if len(remaining) == 0 {
		t.Fatalf("expected remaining children")
	}
}

func TestWaitForChildrenValidation(t *testing.T) {
	if _, err := WaitForChildren(context.Background(), nil, 0, 0); err == nil {
		t.Fatalf("expected validation error")
	}
}
