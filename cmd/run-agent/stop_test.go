package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// makeRunWithPID creates a fake run directory with PID/PGID set.
func makeRunWithPID(t *testing.T, root, project, task, runID, status string, startTime time.Time, exitCode, pid int) string {
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
		ExitCode:  exitCode,
		PID:       pid,
		PGID:      pid,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return runDir
}

func TestStopCmd_RunDirNotExist(t *testing.T) {
	err := runStop("/nonexistent/run/dir/that/does/not/exist", "", "", "", "", false)
	if err == nil {
		t.Fatal("expected error for non-existent run directory")
	}
}

func TestStopCmd_MissingRoot(t *testing.T) {
	err := runStop("", "", "proj", "task1", "", false)
	if err == nil {
		t.Fatal("expected error when --root not provided")
	}
}

func TestStopCmd_MissingProject(t *testing.T) {
	err := runStop("", "/some/root", "", "task1", "", false)
	if err == nil {
		t.Fatal("expected error when --project not provided")
	}
}

func TestStopCmd_MissingTask(t *testing.T) {
	err := runStop("", "/some/root", "proj", "", "", false)
	if err == nil {
		t.Fatal("expected error when --task not provided")
	}
}

func TestStopCmd_NoRunningTasks(t *testing.T) {
	root := t.TempDir()
	makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)

	err := runStop("", root, "proj", "task1", "", false)
	if err == nil {
		t.Fatal("expected error when no running tasks found")
	}
}

func TestStopCmd_NoRunsDir(t *testing.T) {
	root := t.TempDir()
	// Don't create any runs directory
	err := runStop("", root, "proj", "task1", "", false)
	if err == nil {
		t.Fatal("expected error when runs directory does not exist")
	}
}

func TestStopCmd_AlreadyCompleted(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)

	err := runStop(runDir, "", "", "", "", false)
	if err != nil {
		t.Fatalf("expected no error for already-completed run, got: %v", err)
	}
}

func TestStopCmd_AlreadyFailed(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusFailed, time.Now().Add(-time.Hour), 1)

	err := runStop(runDir, "", "", "", "", false)
	if err != nil {
		t.Fatalf("expected no error for failed run, got: %v", err)
	}
}

func TestStopCmd_RunDirWithDeadProcess(t *testing.T) {
	root := t.TempDir()

	// Start a trivial process and wait for it to exit so we have a known-dead PID.
	proc := exec.Command("true")
	if err := proc.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	deadPID := proc.Process.Pid
	_ = proc.Wait()

	runDir := makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0, deadPID)

	// Process is dead, so runStop should print a message and return nil.
	err := runStop(runDir, "", "", "", "", false)
	if err != nil {
		t.Fatalf("expected no error when process is not alive, got: %v", err)
	}
}

func TestStopCmd_ExternalOwnership(t *testing.T) {
	root := t.TempDir()
	runDir := makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0, os.Getpid())
	path := filepath.Join(runDir, "run-info.yaml")
	info, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	info.ProcessOwnership = storage.ProcessOwnershipExternal
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	err = runStop(runDir, "", "", "", "", false)
	if err == nil {
		t.Fatal("expected error for externally owned run")
	}
	if !strings.Contains(err.Error(), "externally owned") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRunDir_WithRunDir(t *testing.T) {
	root := t.TempDir()
	runDir := makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0, 0)

	resolved, err := resolveRunDir(runDir, "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved != runDir {
		t.Errorf("expected %s, got %s", runDir, resolved)
	}
}

func TestResolveRunDir_LatestRunning(t *testing.T) {
	root := t.TempDir()
	makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0, 0)
	runDir2 := makeRunWithPID(t, root, "proj", "task1", "run-002", storage.StatusRunning, time.Now(), 0, 0)

	resolved, err := resolveRunDir("", root, "proj", "task1", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved != runDir2 {
		t.Errorf("expected latest run dir %s, got %s", runDir2, resolved)
	}
}

func TestResolveRunDir_SkipsNonRunning(t *testing.T) {
	root := t.TempDir()
	makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now(), 0, 0)
	runDir2 := makeRunWithPID(t, root, "proj", "task1", "run-002", storage.StatusRunning, time.Now(), 0, 0)

	resolved, err := resolveRunDir("", root, "proj", "task1", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved != runDir2 {
		t.Errorf("expected %s, got %s", runDir2, resolved)
	}
}

func TestResolveRunDir_SpecificRunID(t *testing.T) {
	root := t.TempDir()
	runDir1 := makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0, 0)
	makeRunWithPID(t, root, "proj", "task1", "run-002", storage.StatusRunning, time.Now(), 0, 0)

	resolved, err := resolveRunDir("", root, "proj", "task1", "run-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resolved != runDir1 {
		t.Errorf("expected %s, got %s", runDir1, resolved)
	}
}

func TestResolveRunDir_SpecificRunIDNotFound(t *testing.T) {
	root := t.TempDir()
	makeRunWithPID(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0, 0)

	_, err := resolveRunDir("", root, "proj", "task1", "run-999")
	if err == nil {
		t.Fatal("expected error for non-existent run ID")
	}
}

// TestStopCmd_MissingRunInfo verifies that stop returns cleanly (nil error) when
// run-info.yaml is absent from the specified --run-dir.
func TestStopCmd_MissingRunInfo(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "proj", "task1", "runs", "run-orphan")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err := runStop(runDir, "", "", "", "", false)
	if err != nil {
		t.Fatalf("expected no error for missing run-info.yaml, got: %v", err)
	}
}

func TestStopCmd_FlagParsing_RunDir(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"stop",
		"--run-dir", "/nonexistent/dir",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent run directory")
	}
}

func TestStopCmd_FlagParsing_NoRunning(t *testing.T) {
	root := t.TempDir()
	makeRun(t, root, "proj", "task1", "run-done", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"stop",
		"--root", root,
		"--project", "proj",
		"--task", "task1",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no running tasks")
	}
}
