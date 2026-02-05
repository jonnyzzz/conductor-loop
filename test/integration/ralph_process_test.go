package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const envOrchStubCrash = "ORCH_STUB_CRASH"

func TestAgentImmediateExit(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task-immediate"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("job task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	if err := runner.RunJob(projectID, taskID, runner.JobOptions{
		RootDir:    root,
		Agent:      "codex",
		Prompt:     "immediate",
		WorkingDir: taskDir,
	}); err != nil {
		t.Fatalf("RunJob: %v", err)
	}

	runDir := singleRunDir(t, taskDir)
	info := readRunInfo(t, runDir)
	if info.Status != storage.StatusCompleted {
		t.Fatalf("expected completed status, got %q", info.Status)
	}
}

func TestAgentCrashUpdatesRunInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based crash test is unix-only")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-crash"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("crash task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCrashStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	err := runner.RunJob(projectID, taskID, runner.JobOptions{
		RootDir:    root,
		Agent:      "codex",
		Prompt:     "crash",
		WorkingDir: taskDir,
		Environment: map[string]string{
			envOrchStubCrash: "1",
		},
	})
	if err == nil {
		t.Fatalf("expected RunJob to fail")
	}

	runDir := singleRunDir(t, taskDir)
	info := readRunInfo(t, runDir)
	if info.Status != storage.StatusFailed {
		t.Fatalf("expected failed status, got %q", info.Status)
	}
	if info.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}

func TestFindActiveChildrenMarksStaleRunsFailed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group test is unix-only")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-stale"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	pm, err := runner.NewProcessManager(t.TempDir())
	if err != nil {
		t.Fatalf("new process manager: %v", err)
	}
	proc, err := pm.SpawnAgent(context.Background(), "sleep", runner.SpawnOptions{
		Command: commandForSleepShort(),
		Args:    sleepArgsShort(),
	})
	if err != nil {
		t.Fatalf("spawn process: %v", err)
	}
	pgid := proc.PGID
	pid := proc.PID
	_ = proc.Wait()

	runID := "stale-run"
	runDir := filepath.Join(taskDir, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	info := &storage.RunInfo{
		Version:     1,
		RunID:       runID,
		ParentRunID: "root-run",
		ProjectID:   projectID,
		TaskID:      taskID,
		AgentType:   "codex",
		PID:         pid,
		PGID:        pgid,
		StartTime:   time.Now().Add(-time.Minute).UTC(),
		ExitCode:    -1,
		Status:      storage.StatusRunning,
	}
	infoPath := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(infoPath, info); err != nil {
		t.Fatalf("write run info: %v", err)
	}

	children, err := runner.FindActiveChildren(taskDir)
	if err != nil {
		t.Fatalf("FindActiveChildren: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("expected no active children, got %d", len(children))
	}

	updated, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		t.Fatalf("read run info: %v", err)
	}
	if updated.Status != storage.StatusFailed {
		t.Fatalf("expected failed status, got %q", updated.Status)
	}
	if updated.EndTime.IsZero() {
		t.Fatalf("expected end time to be set")
	}
}

func TestTerminateProcessGroup(t *testing.T) {
	pm, err := runner.NewProcessManager(t.TempDir())
	if err != nil {
		t.Fatalf("new process manager: %v", err)
	}
	proc, err := pm.SpawnAgent(context.Background(), "sleep", runner.SpawnOptions{
		Command: commandForSleep(),
		Args:    sleepArgs(),
	})
	if err != nil {
		t.Fatalf("spawn process: %v", err)
	}
	pgid := proc.PGID

	if err := runner.TerminateProcessGroup(pgid); err != nil {
		_ = proc.Cmd.Process.Kill()
		t.Fatalf("terminate process group: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()
	select {
	case <-time.After(2 * time.Second):
		_ = proc.Cmd.Process.Kill()
		t.Fatalf("timeout waiting for process to terminate")
	case <-done:
	}
}

func buildCrashStub(t *testing.T, dir string) string {
	t.Helper()

	stubPath := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}

	src := `package main

import (
	"os"
	"syscall"
)

func main() {
	if os.Getenv("` + envOrchStubCrash + `") != "" {
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		return
	}
	os.Exit(0)
}
`

	srcPath := filepath.Join(dir, "codex_crash_stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", stubPath, srcPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build stub: %v\n%s", err, out)
	}

	return stubPath
}

func commandForSleep() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sleep"
}

func sleepArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/C", "ping", "-n", "6", "127.0.0.1"}
	}
	return []string{"5"}
}

func commandForSleepShort() string {
	return "sleep"
}

func sleepArgsShort() []string {
	return []string{"0.1"}
}
