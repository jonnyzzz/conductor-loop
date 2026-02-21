package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
)

func TestRunTaskWithRealAgent(t *testing.T) {
	if os.Getenv("RUN_REAL_AGENT_TESTS") != "1" {
		t.Skip("set RUN_REAL_AGENT_TESTS=1 to run real agent integration test")
	}

	agentType, reason := pickRealAgent()
	if agentType == "" {
		t.Skip(reason)
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-real-agent"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("Respond with OK and exit."), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	donePath := filepath.Join(taskDir, "DONE")
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.RunTask(projectID, taskID, runner.TaskOptions{
			RootDir:      root,
			Agent:        agentType,
			WorkingDir:   taskDir,
			WaitTimeout:  2 * time.Minute,
			PollInterval: 100 * time.Millisecond,
		})
	}()

	if _, err := waitForRunDir(taskDir, 30*time.Second); err != nil {
		t.Fatalf("wait for run dir: %v", err)
	}
	if err := os.WriteFile(donePath, []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunTask: %v", err)
		}
	case <-time.After(2 * time.Minute):
		t.Fatalf("timeout waiting for RunTask")
	}
}

func TestParentChildRunsThreeChildren(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not reliable on windows")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-parent-child"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("parent task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	childErr := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(idx int) {
			childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
				RootDir:     root,
				Agent:       "codex",
				Prompt:      fmt.Sprintf("child %d", idx),
				WorkingDir:  taskDir,
				ParentRunID: "root-run",
				Environment: map[string]string{
					envOrchStubSleepMs: "350",
					envOrchStubStdout:  fmt.Sprintf("child-%d", idx),
				},
			})
		}(i)
	}

	time.Sleep(120 * time.Millisecond)

	start := time.Now()
	err := runner.RunTask(projectID, taskID, runner.TaskOptions{
		RootDir:      root,
		Agent:        "codex",
		WorkingDir:   taskDir,
		WaitTimeout:  3 * time.Second,
		PollInterval: 25 * time.Millisecond,
		Environment: map[string]string{
			envOrchStubDoneFile: filepath.Join(taskDir, "DONE"),
		},
	})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Fatalf("expected RunTask to wait for children, elapsed %v", elapsed)
	}

	for i := 0; i < 3; i++ {
		if err := <-childErr; err != nil {
			t.Fatalf("child run failed: %v", err)
		}
	}
}

func TestRalphLoopWaitForChildren(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not reliable on windows")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-wait-children"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("wait task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	childErr := make(chan error, 1)
	go func() {
		childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
			RootDir:     root,
			Agent:       "codex",
			Prompt:      "child wait",
			WorkingDir:  taskDir,
			ParentRunID: "root-run",
			Environment: map[string]string{
				// 600ms gives RunTask a comfortable 400ms margin after the 100ms pre-sleep.
				// Previously 300ms left 0ms margin, causing intermittent flakiness.
				envOrchStubSleepMs: "600",
			},
		})
	}()

	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	err := runner.RunTask(projectID, taskID, runner.TaskOptions{
		RootDir:      root,
		Agent:        "codex",
		WorkingDir:   taskDir,
		WaitTimeout:  3 * time.Second,
		PollInterval: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("expected RunTask to wait, elapsed %v", elapsed)
	}

	if err := <-childErr; err != nil {
		t.Fatalf("child run failed: %v", err)
	}
}

func pickRealAgent() (string, string) {
	if _, err := exec.LookPath("codex"); err == nil && strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" {
		return "codex", ""
	}
	if _, err := exec.LookPath("claude"); err == nil && strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) != "" {
		return "claude", ""
	}
	return "", "codex/claude CLI with API key not available"
}

func waitForRunDir(taskDir string, timeout time.Duration) (string, error) {
	runsDir := filepath.Join(taskDir, "runs")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(runsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					return filepath.Join(runsDir, entry.Name()), nil
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out waiting for run dir")
}
