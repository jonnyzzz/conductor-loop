package unit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestProcessSpawn(t *testing.T) {
	runDir := t.TempDir()
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}

	cmd, args := runnerOutputCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proc, err := pm.SpawnAgent(ctx, "test", runner.SpawnOptions{
		Command: cmd,
		Args:    args,
	})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if proc.PID <= 0 || proc.PGID <= 0 {
		t.Fatalf("expected pid/pgid, got %d/%d", proc.PID, proc.PGID)
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
}

func TestRalphLoopDONEDetection(t *testing.T) {
	taskDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}
	bus, err := messagebus.NewMessageBus(filepath.Join(taskDir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	calls := 0
	loop, err := runner.NewRalphLoop(taskDir, bus,
		runner.WithProjectTask("project", "task"),
		runner.WithWaitTimeout(500*time.Millisecond),
		runner.WithPollInterval(10*time.Millisecond),
		runner.WithRootRunner(func(ctx context.Context, attempt int) error {
			calls++
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected no root runs, got %d", calls)
	}
}

func TestOrchestrationParentChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not supported on windows")
	}
	root := t.TempDir()
	runsDir := filepath.Join(root, "runs")
	childID := "child-1"
	childDir := filepath.Join(runsDir, childID)
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir child run: %v", err)
	}

	proc, cancel := runnerSpawnSleepProcess(t, childDir, 500*time.Millisecond)
	defer cancel()
	defer func() {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
	}()

	info := &storage.RunInfo{
		RunID:       childID,
		ParentRunID: "root-run",
		ProjectID:   "project",
		TaskID:      "task",
		AgentType:   "codex",
		PID:         proc.PID,
		PGID:        proc.PGID,
		StartTime:   time.Now().UTC(),
		ExitCode:    -1,
		Status:      storage.StatusRunning,
	}
	if err := storage.WriteRunInfo(filepath.Join(childDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	children, err := runner.FindActiveChildren(root)
	if err != nil {
		t.Fatalf("FindActiveChildren: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].RunID != childID {
		t.Fatalf("unexpected child run id: %q", children[0].RunID)
	}
}

func runnerOutputCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo stdout & echo stderr 1>&2"}
	}
	return "sh", []string{"-c", "echo stdout; echo stderr 1>&2; sleep 0.1"}
}

func runnerSpawnSleepProcess(t *testing.T, runDir string, duration time.Duration) (*runner.Process, context.CancelFunc) {
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}
	cmd, args := runnerSleepCommand(duration)
	ctx, cancel := context.WithTimeout(context.Background(), duration+2*time.Second)
	proc, err := pm.SpawnAgent(ctx, "test", runner.SpawnOptions{
		Command: cmd,
		Args:    args,
	})
	if err != nil {
		cancel()
		t.Fatalf("SpawnAgent: %v", err)
	}
	return proc, cancel
}

func runnerSleepCommand(duration time.Duration) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "ping -n 2 127.0.0.1 >nul"}
	}
	return "sh", []string{"-c", fmt.Sprintf("sleep %.3f", duration.Seconds())}
}
