package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskResume_MissingProject(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task", "resume", "--task", "task-20260220-153045-my-feature"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing --project")
	}
}

func TestTaskResume_MissingTask(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task", "resume", "--project", "my-project"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing --task")
	}
}

func TestTaskResume_InvalidTaskID(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"task", "resume",
		"--project", "my-project",
		"--task", "invalid-task-id",
		"--root", t.TempDir(),
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid task ID")
	}
	if !strings.Contains(err.Error(), "invalid task ID") {
		t.Errorf("expected 'invalid task ID' in error, got: %v", err)
	}
}

func TestTaskResume_TaskDirNotExist(t *testing.T) {
	root := t.TempDir()
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"task", "resume",
		"--project", "my-project",
		"--task", "task-20260220-153045-my-feature",
		"--root", root,
		"--agent", "codex",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when task directory does not exist")
	}
	if !strings.Contains(err.Error(), "task directory not found") {
		t.Errorf("expected 'task directory not found' in error, got: %v", err)
	}
}

func TestTaskResume_TaskMDNotExist(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260220-153045-my-feature"
	taskDir := filepath.Join(root, "my-project", taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	// No TASK.md written

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"task", "resume",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
		"--agent", "codex",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when TASK.md does not exist")
	}
	if !strings.Contains(err.Error(), "TASK.md not found") {
		t.Errorf("expected 'TASK.md not found' in error, got: %v", err)
	}
}

func TestTaskResume_PrintsResumingMessage(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260220-153045-my-feature"
	taskDir := filepath.Join(root, "my-project", taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("do the thing"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	// Write DONE so RunTask exits immediately without running an agent
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	var stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"task", "resume",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
		"--agent", "codex",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	out := stderr.String()
	if !strings.Contains(out, "Resuming task: "+taskID) {
		t.Errorf("expected stderr to contain 'Resuming task: %s', got: %q", taskID, out)
	}
}

func TestTaskResume_SubcommandAppearsInHelp(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"task", "--help"})
	_ = cmd.Execute()

	out := stdout.String()
	if !strings.Contains(out, "resume") {
		t.Errorf("expected 'resume' in task --help output, got:\n%s", out)
	}
}
