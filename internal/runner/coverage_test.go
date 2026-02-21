//go:build !windows

package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestAllocateRunDir verifies that AllocateRunDir returns a valid run ID
// and creates the directory on disk.
func TestAllocateRunDir(t *testing.T) {
	runsDir := filepath.Join(t.TempDir(), "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir runs: %v", err)
	}

	runID, runDir, err := AllocateRunDir(runsDir)
	if err != nil {
		t.Fatalf("AllocateRunDir: %v", err)
	}
	if runID == "" {
		t.Fatal("expected non-empty run ID")
	}
	if runDir == "" {
		t.Fatal("expected non-empty run dir path")
	}
	if _, err := os.Stat(runDir); err != nil {
		t.Fatalf("run directory not created: %v", err)
	}
}

func TestAllocateRunDirError(t *testing.T) {
	// Passing a path that doesn't exist or isn't a directory should fail.
	_, _, err := AllocateRunDir("")
	if err == nil {
		t.Fatal("expected error for empty runs dir")
	}
}

// TestAllocateRunDirUniqueness ensures repeated calls produce different IDs.
func TestAllocateRunDirUniqueness(t *testing.T) {
	runsDir := filepath.Join(t.TempDir(), "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir runs: %v", err)
	}

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		runID, _, err := AllocateRunDir(runsDir)
		if err != nil {
			t.Fatalf("AllocateRunDir[%d]: %v", i, err)
		}
		if ids[runID] {
			t.Fatalf("duplicate run ID at iteration %d: %s", i, runID)
		}
		ids[runID] = true
	}
}

// TestKillProcessGroupInvalidPGID verifies that KillProcessGroup rejects
// invalid process group IDs without a syscall.
func TestKillProcessGroupInvalidPGID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("KillProcessGroup not available on windows")
	}
	err := KillProcessGroup(0)
	if err == nil {
		t.Fatal("expected error for pgid=0")
	}
	err = KillProcessGroup(-1)
	if err == nil {
		t.Fatal("expected error for pgid=-1")
	}
}

// TestKillProcessGroupNotFound verifies that killing a non-existent process
// group returns the appropriate error.
func TestKillProcessGroupNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("KillProcessGroup not available on windows")
	}
	// Use a very large PID that almost certainly doesn't exist.
	err := KillProcessGroup(999999999)
	if err == nil {
		t.Fatal("expected error for non-existent process group")
	}
}

// TestIsProcessAliveCurrentPID verifies that the current process is alive.
func TestIsProcessAliveCurrentPID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("IsProcessAlive not available on windows")
	}
	pid := os.Getpid()
	if !IsProcessAlive(pid) {
		t.Fatalf("current process (pid=%d) should be alive", pid)
	}
}

// TestIsProcessAliveInvalidPID verifies that invalid PIDs are not considered alive.
func TestIsProcessAliveInvalidPID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("IsProcessAlive not available on windows")
	}
	if IsProcessAlive(0) {
		t.Fatal("pid=0 should not be alive")
	}
	if IsProcessAlive(-1) {
		t.Fatal("pid=-1 should not be alive")
	}
	// A very large PID that should not exist.
	if IsProcessAlive(999999999) {
		t.Fatal("pid=999999999 should not be alive")
	}
}

// TestRalphLoopContextCancelledBeforeStart verifies that Run returns
// the context error when the context is already cancelled on entry.
func TestRalphLoopContextCancelledBeforeStart(t *testing.T) {
	taskDir := t.TempDir()
	bus := newMessageBus(t, taskDir)

	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(10),
		WithRestartDelay(time.Second),
		WithRootRunner(func(ctx context.Context, attempt int) error {
			t.Error("root runner should not be called when context already cancelled")
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = loop.Run(ctx)
	if err == nil {
		t.Fatal("expected context error from Run")
	}
}

// TestRalphLoopContextCancelledDuringSleep verifies that Run returns the
// context error when the context is cancelled during the restart delay.
func TestRalphLoopContextCancelledDuringSleep(t *testing.T) {
	taskDir := t.TempDir()
	bus := newMessageBus(t, taskDir)

	ran := make(chan struct{}, 1)
	loop, err := NewRalphLoop(taskDir, bus,
		WithProjectTask("project", "task"),
		WithMaxRestarts(10),
		WithRestartDelay(5*time.Second), // long delay
		WithPollInterval(10*time.Millisecond),
		WithRootRunner(func(ctx context.Context, attempt int) error {
			ran <- struct{}{}
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewRalphLoop: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- loop.Run(ctx)
	}()

	// Wait for the root runner to execute once, then cancel.
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("root runner never called")
	}
	cancel()

	select {
	case err := <-resultCh:
		if err == nil {
			t.Fatal("expected context error from Run")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}
