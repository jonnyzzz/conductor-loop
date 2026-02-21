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

// Tests for the top-level "resume" command (run-agent resume).

func TestResume_MissingProject(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"resume", "--task", "task-20260220-153045-my-feature"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing --project")
	}
}

func TestResume_MissingTask(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"resume", "--project", "my-project"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing --task")
	}
}

func TestResume_InvalidTaskID(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"resume",
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

func TestResume_TaskDirNotExist(t *testing.T) {
	root := t.TempDir()
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"resume",
		"--project", "my-project",
		"--task", "task-20260220-153045-my-feature",
		"--root", root,
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when task directory does not exist")
	}
	if !strings.Contains(err.Error(), "task directory not found") {
		t.Errorf("expected 'task directory not found' in error, got: %v", err)
	}
}

func TestResume_DeletesDoneAndPrintsMessage(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260220-153045-my-feature"
	taskDir := filepath.Join(root, "my-project", taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	doneFile := filepath.Join(taskDir, "DONE")
	if err := os.WriteFile(doneFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{
		"resume",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	// DONE file must be deleted.
	if _, err := os.Stat(doneFile); !os.IsNotExist(err) {
		t.Error("expected DONE file to be deleted after resume")
	}

	// Verify output message.
	out := stdout.String()
	want := "Resumed task " + taskID + " (restart counter reset)"
	if !strings.Contains(out, want) {
		t.Errorf("expected stdout to contain %q, got: %q", want, out)
	}
}

func TestResume_NoDoneFile_StillSucceeds(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260220-153045-my-feature"
	taskDir := filepath.Join(root, "my-project", taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	// No DONE file

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"resume",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("resume without DONE file failed: %v", err)
	}
}

func TestResume_NoAgent_DoesNotRunTask(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260220-153045-my-feature"
	taskDir := filepath.Join(root, "my-project", taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{
		"resume",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
		// deliberately no --agent flag
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("resume without --agent failed: %v", err)
	}

	// No runs/ directory should be created since no agent was launched.
	runsDir := filepath.Join(taskDir, "runs")
	if _, err := os.Stat(runsDir); !os.IsNotExist(err) {
		t.Error("expected no runs/ directory when --agent is not specified")
	}
}

func TestResume_AppearsInHelp(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute()

	out := stdout.String()
	if !strings.Contains(out, "resume") {
		t.Errorf("expected 'resume' in root --help output, got:\n%s", out)
	}
}
