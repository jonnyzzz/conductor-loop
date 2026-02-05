package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunTaskDone(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	if err := RunTask("project", "task", TaskOptions{RootDir: root, Agent: "codex"}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}
}

func TestRunTaskMissingPrompt(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := RunTask("project", "task", TaskOptions{RootDir: root, Agent: "codex"}); err == nil {
		t.Fatalf("expected error for missing TASK.md")
	}
}

func TestRunTaskMaxRestarts(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := RunTask("project", "task", TaskOptions{
		RootDir:      root,
		Agent:        "codex",
		MaxRestarts:  1,
		RestartDelay: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected max restarts error")
	}
}
