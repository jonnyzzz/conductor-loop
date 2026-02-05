package unit_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRalphLoopDONEWithChildren(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not supported on windows")
	}

	taskDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	runsDir := filepath.Join(taskDir, "runs")
	childRunID := "run-child"
	childDir := filepath.Join(runsDir, childRunID)
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("create child run dir: %v", err)
	}

	proc, cancel := spawnSleepProcess(t, childDir, 300*time.Millisecond)
	defer cancel()
	defer func() {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
	}()

	writeChildRunInfo(t, childDir, childRunID, "root-run", proc)

	bus := newMessageBus(t, taskDir)

	rootCalls := 0
	loop, err := runner.NewRalphLoop(taskDir, bus,
		runner.WithProjectTask("project", "task"),
		runner.WithMaxRestarts(2),
		runner.WithWaitTimeout(2*time.Second),
		runner.WithPollInterval(25*time.Millisecond),
		runner.WithRootRunner(func(ctx context.Context, attempt int) error {
			rootCalls++
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- loop.Run(context.Background())
	}()

	select {
	case err := <-resultCh:
		t.Fatalf("expected loop to wait for children, got %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("wait child: %v", err)
	}

	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("loop error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("loop did not finish after child exit")
	}

	if rootCalls != 0 {
		t.Fatalf("expected root runner not called, got %d", rootCalls)
	}
}

func TestRalphLoopDONEWithoutChildren(t *testing.T) {
	taskDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	bus := newMessageBus(t, taskDir)

	rootCalls := 0
	loop, err := runner.NewRalphLoop(taskDir, bus,
		runner.WithProjectTask("project", "task"),
		runner.WithMaxRestarts(2),
		runner.WithWaitTimeout(500*time.Millisecond),
		runner.WithPollInterval(10*time.Millisecond),
		runner.WithRootRunner(func(ctx context.Context, attempt int) error {
			rootCalls++
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	start := time.Now()
	if err := loop.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("expected quick completion, took %v", elapsed)
	}
	if rootCalls != 0 {
		t.Fatalf("expected root runner not called, got %d", rootCalls)
	}
}

func TestRalphLoopRestartLogic(t *testing.T) {
	taskDir := t.TempDir()
	bus := newMessageBus(t, taskDir)

	rootCalls := 0
	loop, err := runner.NewRalphLoop(taskDir, bus,
		runner.WithProjectTask("project", "task"),
		runner.WithMaxRestarts(5),
		runner.WithWaitTimeout(500*time.Millisecond),
		runner.WithPollInterval(10*time.Millisecond),
		runner.WithRestartDelay(10*time.Millisecond),
		runner.WithRootRunner(func(ctx context.Context, attempt int) error {
			rootCalls++
			if rootCalls == 2 {
				if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
					return err
				}
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	if err := loop.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rootCalls != 2 {
		t.Fatalf("expected 2 root runs, got %d", rootCalls)
	}

	messages := readMessages(t, bus)
	startLogs := 0
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if strings.Contains(strings.ToLower(msg.Body), "starting root agent") {
			startLogs++
		}
	}
	if startLogs != rootCalls {
		t.Fatalf("expected %d restart logs, got %d", rootCalls, startLogs)
	}
}

func TestChildWaitTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not supported on windows")
	}

	runDir := t.TempDir()
	proc, cancel := spawnSleepProcess(t, runDir, 2*time.Second)
	defer cancel()
	defer func() {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
	}()

	runInfoPath := writeChildRunInfo(t, runDir, "run-child", "root-run", proc)

	children := []runner.ChildProcess{
		{
			RunID:       "run-child",
			PID:         proc.PID,
			PGID:        proc.PGID,
			RunInfoPath: runInfoPath,
		},
	}

	remaining, err := runner.WaitForChildren(context.Background(), children, 50*time.Millisecond, 10*time.Millisecond)
	if !errors.Is(err, runner.ErrChildWaitTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining child, got %d", len(remaining))
	}
}

func spawnSleepProcess(t *testing.T, runDir string, duration time.Duration) (*runner.Process, context.CancelFunc) {
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}
	cmd, args := sleepCommand(duration)
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

func sleepCommand(duration time.Duration) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "ping -n 2 127.0.0.1 >nul"}
	}
	return "sh", []string{"-c", fmt.Sprintf("sleep %.3f", duration.Seconds())}
}

func writeChildRunInfo(t *testing.T, runDir, runID, parentRunID string, proc *runner.Process) string {
	info := &storage.RunInfo{
		RunID:       runID,
		ParentRunID: parentRunID,
		ProjectID:   "project",
		TaskID:      "task",
		AgentType:   "codex",
		PID:         proc.PID,
		PGID:        proc.PGID,
		StartTime:   time.Now().UTC(),
		ExitCode:    -1,
		Status:      storage.StatusRunning,
	}
	path := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return path
}

func newMessageBus(t *testing.T, taskDir string) *messagebus.MessageBus {
	path := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	return bus
}

func readMessages(t *testing.T, bus *messagebus.MessageBus) []*messagebus.Message {
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	return messages
}
