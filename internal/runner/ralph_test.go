package runner

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
	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(2),
		WithWaitTimeout(2*time.Second),
		WithPollInterval(25*time.Millisecond),
		WithRootRunner(func(ctx context.Context, attempt int) error {
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
	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(2),
		WithWaitTimeout(500*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
		WithRootRunner(func(ctx context.Context, attempt int) error {
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
	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(5),
		WithWaitTimeout(500*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
		WithRestartDelay(10*time.Millisecond),
		WithRootRunner(func(ctx context.Context, attempt int) error {
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

	children := []ChildProcess{{
		RunID:       "run-child",
		PID:         proc.PID,
		PGID:        proc.PGID,
		RunInfoPath: runInfoPath,
	}}

	remaining, err := WaitForChildren(context.Background(), children, 50*time.Millisecond, 10*time.Millisecond)
	if !errors.Is(err, ErrChildWaitTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining child, got %d", len(remaining))
	}
}

func TestInferProjectTaskID(t *testing.T) {
	project, task := inferProjectTaskID(filepath.Join("root", "project", "task"))
	if project != "project" || task != "task" {
		t.Fatalf("unexpected ids: %s/%s", project, task)
	}
}

func TestInferProjectTaskIDEmpty(t *testing.T) {
	project, task := inferProjectTaskID("")
	if project != "" || task != "" {
		t.Fatalf("expected empty ids, got %s/%s", project, task)
	}
	root := string(filepath.Separator)
	project, task = inferProjectTaskID(root)
	if project != "" || task != "" {
		t.Fatalf("expected empty ids for root, got %s/%s", project, task)
	}
}

func TestChildRunIDsSkipsEmpty(t *testing.T) {
	children := []ChildProcess{{RunID: ""}, {RunID: "run-1"}}
	ids := childRunIDs(children)
	if !strings.Contains(ids, "run-1") {
		t.Fatalf("expected run id in output: %s", ids)
	}
}

func TestAppendMessageValidation(t *testing.T) {
	bus, err := messagebus.NewMessageBus(filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	loop := &RalphLoop{messagebus: bus, projectID: "project", taskID: "task"}
	if err := loop.appendMessage("", "body"); err == nil {
		t.Fatalf("expected error for empty message type")
	}
	if err := loop.appendMessage("INFO", ""); err == nil {
		t.Fatalf("expected error for empty message body")
	}
}

func TestNewRalphLoopValidation(t *testing.T) {
	if _, err := NewRalphLoop("", nil); err == nil {
		t.Fatalf("expected error for empty run dir")
	}
	dir := t.TempDir()
	bus, err := messagebus.NewMessageBus(filepath.Join(dir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if _, err := NewRalphLoop(dir, nil); err == nil {
		t.Fatalf("expected error for nil message bus")
	}
	_, err = NewRalphLoop(dir, bus,
		WithProjectTask("project", "task"),
		WithWaitTimeout(-1),
		WithRootRunner(func(ctx context.Context, attempt int) error { return nil }),
	)
	if err == nil {
		t.Fatalf("expected error for negative wait timeout")
	}
}

func TestNewRalphLoopRootRunnerNil(t *testing.T) {
	dir := t.TempDir()
	bus, err := messagebus.NewMessageBus(filepath.Join(dir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if _, err := NewRalphLoop(dir, bus, WithProjectTask("project", "task")); err == nil {
		t.Fatalf("expected error for nil root runner")
	}
}

func TestNewRalphLoopInvalidOptions(t *testing.T) {
	dir := t.TempDir()
	bus, err := messagebus.NewMessageBus(filepath.Join(dir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if _, err := NewRalphLoop(dir, bus,
		WithProjectTask("project", "task"),
		WithPollInterval(-1),
		WithRootRunner(func(ctx context.Context, attempt int) error { return nil }),
	); err == nil {
		t.Fatalf("expected error for negative poll interval")
	}
	if _, err := NewRalphLoop(dir, bus,
		WithProjectTask("project", "task"),
		WithRestartDelay(-1),
		WithRootRunner(func(ctx context.Context, attempt int) error { return nil }),
	); err == nil {
		t.Fatalf("expected error for negative restart delay")
	}
	if _, err := NewRalphLoop(dir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(-1),
		WithRootRunner(func(ctx context.Context, attempt int) error { return nil }),
	); err == nil {
		t.Fatalf("expected error for negative max restarts")
	}
}

func TestRalphLoopRootErrorMaxRestarts(t *testing.T) {
	taskDir := t.TempDir()
	bus := newMessageBus(t, taskDir)

	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(1),
		WithPollInterval(10*time.Millisecond),
		WithRestartDelay(10*time.Millisecond),
		WithRootRunner(func(ctx context.Context, attempt int) error {
			return errors.New("boom")
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	if err := loop.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "max restarts") {
		t.Fatalf("expected max restarts error, got %v", err)
	}
}

func TestRalphLoopHandleDoneTimeout(t *testing.T) {
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

	proc, cancel := spawnSleepProcess(t, childDir, 2*time.Second)
	defer cancel()
	defer func() {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
	}()
	writeChildRunInfo(t, childDir, childRunID, "root-run", proc)

	bus := newMessageBus(t, taskDir)
	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithWaitTimeout(50*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
		WithRootRunner(func(ctx context.Context, attempt int) error { return nil }),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	if err := loop.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRalphLoopDoneIsDirectory(t *testing.T) {
	taskDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(taskDir, "DONE"), 0o755); err != nil {
		t.Fatalf("mkdir DONE: %v", err)
	}
	bus := newMessageBus(t, taskDir)
	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithRootRunner(func(ctx context.Context, attempt int) error { return nil }),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}
	if err := loop.Run(context.Background()); err == nil {
		t.Fatalf("expected error for DONE directory")
	}
}

func spawnSleepProcess(t *testing.T, runDir string, duration time.Duration) (*Process, context.CancelFunc) {
	pm, err := NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}
	cmd, args := sleepCommand(duration)
	ctx, cancel := context.WithTimeout(context.Background(), duration+2*time.Second)
	proc, err := pm.SpawnAgent(ctx, "test", SpawnOptions{
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

func writeChildRunInfo(t *testing.T, runDir, runID, parentRunID string, proc *Process) string {
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
