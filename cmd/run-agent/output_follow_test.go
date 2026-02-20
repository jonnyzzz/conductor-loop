package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// TestFollowOutput_AlreadyCompleted verifies that --follow exits immediately
// and prints all content when the run is already complete.
func TestFollowOutput_AlreadyCompleted(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "output.md"), "completed output\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = followOutput(runDir, "output")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "completed output") {
		t.Errorf("expected 'completed output', got: %q", out)
	}
}

// TestFollowOutput_RunningThenCompletes verifies that --follow reads content in
// real-time and exits when the run-info transitions to completed.
func TestFollowOutput_RunningThenCompletes(t *testing.T) {
	oldPoll := followPollInterval
	oldNoData := followNoDataTimeout
	followPollInterval = 20 * time.Millisecond
	followNoDataTimeout = 30 * time.Second
	defer func() {
		followPollInterval = oldPoll
		followNoDataTimeout = oldNoData
	}()

	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0)
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	runInfoPath := filepath.Join(runDir, "run-info.yaml")

	writeFile(t, stdoutPath, "initial line\n")

	go func() {
		time.Sleep(60 * time.Millisecond)
		f, err := os.OpenFile(stdoutPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = f.WriteString("appended line\n")
			_ = f.Close()
		}
		time.Sleep(60 * time.Millisecond)
		info := &storage.RunInfo{
			RunID:     "run-001",
			ProjectID: "proj",
			TaskID:    "task1",
			Status:    storage.StatusCompleted,
		}
		_ = storage.WriteRunInfo(runInfoPath, info)
	}()

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = followOutput(runDir, "output")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "initial line") {
		t.Errorf("expected 'initial line' in output, got: %q", out)
	}
	if !strings.Contains(out, "appended line") {
		t.Errorf("expected 'appended line' in output, got: %q", out)
	}
}

// TestFollowOutput_NoOutputFile_ReturnsError verifies that --follow returns an
// error when the output file does not appear within followFileWaitTimeout.
func TestFollowOutput_NoOutputFile_ReturnsError(t *testing.T) {
	oldWait := followFileWaitTimeout
	oldPoll := followPollInterval
	followFileWaitTimeout = 50 * time.Millisecond
	followPollInterval = 10 * time.Millisecond
	defer func() {
		followFileWaitTimeout = oldWait
		followPollInterval = oldPoll
	}()

	root := t.TempDir()
	// Create run with running status but no stdout file.
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0)

	err := followOutput(runDir, "stdout")
	if err == nil {
		t.Fatal("expected error when output file doesn't exist")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// TestFollowOutput_NoDataTimeout verifies that --follow exits cleanly (nil error)
// when no new data arrives for followNoDataTimeout duration.
func TestFollowOutput_NoDataTimeout(t *testing.T) {
	oldPoll := followPollInterval
	oldNoData := followNoDataTimeout
	followPollInterval = 20 * time.Millisecond
	followNoDataTimeout = 100 * time.Millisecond
	defer func() {
		followPollInterval = oldPoll
		followNoDataTimeout = oldNoData
	}()

	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusRunning, time.Now(), 0)
	writeFile(t, filepath.Join(runDir, "agent-stdout.txt"), "some output\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = followOutput(runDir, "stdout")
	})
	if runErr != nil {
		t.Fatalf("expected nil error on no-data timeout, got: %v", runErr)
	}
	if !strings.Contains(out, "some output") {
		t.Errorf("expected 'some output', got: %q", out)
	}
}

// TestFollowOutput_FlagIntegration verifies that the --follow (-f) cobra flag
// wires up correctly to followOutput for a completed run.
func TestFollowOutput_FlagIntegration(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "output.md"), "flag integration output\n")

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"output",
		"--root", root,
		"--project", "proj",
		"--task", "task1",
		"--follow",
	})

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("output --follow failed: %v", runErr)
	}
	if !strings.Contains(out, "flag integration output") {
		t.Errorf("expected 'flag integration output', got: %q", out)
	}
}

// TestFollowOutput_ShortFlag verifies the -f shorthand works.
func TestFollowOutput_ShortFlag(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "output.md"), "short flag output\n")

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"output",
		"--root", root,
		"--project", "proj",
		"--task", "task1",
		"-f",
	})

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("output -f failed: %v", runErr)
	}
	if !strings.Contains(out, "short flag output") {
		t.Errorf("expected 'short flag output', got: %q", out)
	}
}
