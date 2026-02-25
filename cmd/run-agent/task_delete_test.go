package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// makeTaskRun creates a task directory with a run for testing task delete.
func makeTaskRun(t *testing.T, root, projectID, taskID, runID, status string) string {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     runID,
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    status,
		StartTime: time.Now().UTC(),
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return filepath.Join(root, projectID, taskID)
}

func TestRunTaskDelete_MissingProject(t *testing.T) {
	root := t.TempDir()
	err := runTaskDelete("", "task-20260101-120000-abc", root, false)
	if err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
}

func TestRunTaskDelete_MissingTask(t *testing.T) {
	root := t.TempDir()
	err := runTaskDelete("my-project", "", root, false)
	if err == nil {
		t.Fatal("expected error for missing task, got nil")
	}
}

func TestRunTaskDelete_TaskNotFound(t *testing.T) {
	root := t.TempDir()
	err := runTaskDelete("my-project", "task-20260101-120000-abc", root, false)
	if err == nil {
		t.Fatal("expected error for non-existent task, got nil")
	}
}

func TestRunTaskDelete_RunningRunNoForce(t *testing.T) {
	root := t.TempDir()
	taskDir := makeTaskRun(t, root, "project", "task-running", "run-1", storage.StatusCompleted)
	makeTaskRun(t, root, "project", "task-running", "run-2", storage.StatusRunning)

	err := runTaskDelete("project", "task-running", root, false)
	if err == nil {
		t.Fatal("expected error for running run without --force, got nil")
	}

	// Task directory should still exist.
	if _, statErr := os.Stat(taskDir); os.IsNotExist(statErr) {
		t.Errorf("task directory should NOT be deleted when a run is still running")
	}
}

func TestRunTaskDelete_CompletedRunsSuccess(t *testing.T) {
	root := t.TempDir()
	taskDir := makeTaskRun(t, root, "project", "task-done", "run-1", storage.StatusCompleted)
	makeTaskRun(t, root, "project", "task-done", "run-2", storage.StatusFailed)

	err := runTaskDelete("project", "task-done", root, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Task directory should be deleted.
	if _, statErr := os.Stat(taskDir); !os.IsNotExist(statErr) {
		t.Errorf("expected task directory to be deleted, but it still exists")
	}
}

func TestRunTaskDelete_RunningRunWithForce(t *testing.T) {
	root := t.TempDir()
	taskDir := makeTaskRun(t, root, "project", "task-force", "run-1", storage.StatusRunning)

	err := runTaskDelete("project", "task-force", root, true)
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}

	// Task directory should be deleted even with running run.
	if _, statErr := os.Stat(taskDir); !os.IsNotExist(statErr) {
		t.Errorf("expected task directory to be deleted with --force, but it still exists")
	}
}

func TestResolveRunsDir_Flag(t *testing.T) {
	got, err := config.ResolveRunsDir("/my/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/my/root" {
		t.Errorf("expected /my/root, got %q", got)
	}
}

func TestResolveRunsDir_Default(t *testing.T) {
	got, err := config.ResolveRunsDir("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Fatalf("get home dir: %v", homeErr)
	}
	want := filepath.Join(home, ".run-agent", "runs")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
