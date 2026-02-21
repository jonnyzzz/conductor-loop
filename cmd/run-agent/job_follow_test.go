package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// TestJobFollowFlagRegistered verifies that --follow is registered on the job command.
func TestJobFollowFlagRegistered(t *testing.T) {
	cmd := newJobCmd()
	flag := cmd.Flags().Lookup("follow")
	if flag == nil {
		t.Fatal("--follow flag not registered on job command")
	}
	if flag.Shorthand != "f" {
		t.Errorf("expected -f shorthand, got %q", flag.Shorthand)
	}
	if flag.DefValue != "false" {
		t.Errorf("expected default false, got %q", flag.DefValue)
	}
}

// TestJobFollowShortFlagRegistered verifies the -f shorthand is registered.
func TestJobFollowShortFlagRegistered(t *testing.T) {
	cmd := newJobCmd()
	flag := cmd.Flags().ShorthandLookup("f")
	if flag == nil {
		t.Fatal("-f shorthand not registered on job command")
	}
	if flag.Name != "follow" {
		t.Errorf("expected flag name 'follow', got %q", flag.Name)
	}
}

// TestJobFollowAllocateRunDir verifies that runner.AllocateRunDir creates a valid run directory.
func TestJobFollowAllocateRunDir(t *testing.T) {
	root := t.TempDir()
	runsDir := filepath.Join(root, "proj", "task-20260221-120000-test", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	runID, runDir, err := runner.AllocateRunDir(runsDir)
	if err != nil {
		t.Fatalf("AllocateRunDir: %v", err)
	}
	if runID == "" {
		t.Error("expected non-empty runID")
	}
	if !strings.HasPrefix(runDir, runsDir) {
		t.Errorf("expected runDir under %s, got %q", runsDir, runDir)
	}
	if _, err := os.Stat(runDir); err != nil {
		t.Errorf("run dir %s should exist: %v", runDir, err)
	}
}

// TestJobFollow_FollowsCompletedRun verifies that when a job is already complete,
// --follow streams output and exits cleanly.
func TestJobFollow_FollowsCompletedRun(t *testing.T) {
	oldPoll := followPollInterval
	oldNoData := followNoDataTimeout
	followPollInterval = 20 * time.Millisecond
	followNoDataTimeout = 500 * time.Millisecond
	defer func() {
		followPollInterval = oldPoll
		followNoDataTimeout = oldNoData
	}()

	root := t.TempDir()
	runsDir := filepath.Join(root, "proj", "task-20260221-120000-fol", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, runDir, err := runner.AllocateRunDir(runsDir)
	if err != nil {
		t.Fatalf("AllocateRunDir: %v", err)
	}

	// Write output and mark as completed (simulating a fast-completing job).
	writeFile(t, filepath.Join(runDir, "agent-stdout.txt"), "follow job output\n")
	info := &storage.RunInfo{
		RunID:     filepath.Base(runDir),
		ProjectID: "proj",
		TaskID:    "task-20260221-120000-fol",
		Status:    storage.StatusCompleted,
		ExitCode:  0,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	var out string
	var followErr error
	out = captureStdout(t, func() {
		followErr = followOutput(runDir, "stdout")
	})
	if followErr != nil {
		t.Fatalf("followOutput: %v", followErr)
	}
	if !strings.Contains(out, "follow job output") {
		t.Errorf("expected 'follow job output' in output, got: %q", out)
	}
}

// TestJobFollow_FollowsRunningThenCompletes verifies that --follow streams
// output in real-time as a job runs and exits when it completes.
func TestJobFollow_FollowsRunningThenCompletes(t *testing.T) {
	oldPoll := followPollInterval
	oldNoData := followNoDataTimeout
	followPollInterval = 20 * time.Millisecond
	followNoDataTimeout = 30 * time.Second
	defer func() {
		followPollInterval = oldPoll
		followNoDataTimeout = oldNoData
	}()

	root := t.TempDir()
	runsDir := filepath.Join(root, "proj", "task-20260221-120000-run", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, runDir, err := runner.AllocateRunDir(runsDir)
	if err != nil {
		t.Fatalf("AllocateRunDir: %v", err)
	}

	// Start with running status and initial output.
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	runInfoPath := filepath.Join(runDir, "run-info.yaml")
	writeFile(t, stdoutPath, "initial line\n")
	runningInfo := &storage.RunInfo{
		RunID:     filepath.Base(runDir),
		ProjectID: "proj",
		TaskID:    "task-20260221-120000-run",
		Status:    storage.StatusRunning,
	}
	if err := storage.WriteRunInfo(runInfoPath, runningInfo); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	// Simulate job appending output then completing.
	go func() {
		time.Sleep(60 * time.Millisecond)
		f, err := os.OpenFile(stdoutPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = f.WriteString("appended line\n")
			_ = f.Close()
		}
		time.Sleep(60 * time.Millisecond)
		doneInfo := &storage.RunInfo{
			RunID:     filepath.Base(runDir),
			ProjectID: "proj",
			TaskID:    "task-20260221-120000-run",
			Status:    storage.StatusCompleted,
			ExitCode:  0,
		}
		_ = storage.WriteRunInfo(runInfoPath, doneInfo)
	}()

	var out string
	var followErr error
	out = captureStdout(t, func() {
		followErr = followOutput(runDir, "stdout")
	})
	if followErr != nil {
		t.Fatalf("followOutput: %v", followErr)
	}
	if !strings.Contains(out, "initial line") {
		t.Errorf("expected 'initial line' in output, got: %q", out)
	}
	if !strings.Contains(out, "appended line") {
		t.Errorf("expected 'appended line' in output, got: %q", out)
	}
}

// TestJobFollow_PreallocatedRunDirUsed verifies that PreallocatedRunDir is
// accepted by runner.JobOptions without error.
func TestJobFollow_PreallocatedRunDirUsed(t *testing.T) {
	root := t.TempDir()
	runsDir := filepath.Join(root, "proj", "task-20260221-120000-pre", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_, runDir, err := runner.AllocateRunDir(runsDir)
	if err != nil {
		t.Fatalf("AllocateRunDir: %v", err)
	}

	opts := runner.JobOptions{
		PreallocatedRunDir: runDir,
	}
	// Just verify the field is set correctly; actual job execution is tested by integration tests.
	if opts.PreallocatedRunDir != runDir {
		t.Errorf("expected PreallocatedRunDir %q, got %q", runDir, opts.PreallocatedRunDir)
	}
}
