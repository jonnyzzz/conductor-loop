package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// makeRun creates a fake run directory with a run-info.yaml for testing.
func makeRun(t *testing.T, root, project, task, runID, status string, startTime time.Time, exitCode int) string {
	t.Helper()
	runDir := filepath.Join(root, project, task, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}
	info := &storage.RunInfo{
		RunID:     runID,
		ProjectID: project,
		TaskID:    task,
		AgentType: "claude",
		Status:    status,
		StartTime: startTime,
		ExitCode:  exitCode,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return runDir
}

func TestGCTableDriven(t *testing.T) {
	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name         string
		status       string
		startTime    time.Time
		olderThan    time.Duration
		keepFailed   bool
		exitCode     int
		expectDelete bool
	}{
		{
			name:         "old completed run is deleted",
			status:       storage.StatusCompleted,
			startTime:    old,
			olderThan:    24 * time.Hour,
			expectDelete: true,
		},
		{
			name:         "old failed run is deleted when not keeping failed",
			status:       storage.StatusFailed,
			startTime:    old,
			olderThan:    24 * time.Hour,
			exitCode:     1,
			expectDelete: true,
		},
		{
			name:         "old failed run kept when keepFailed=true",
			status:       storage.StatusFailed,
			startTime:    old,
			olderThan:    24 * time.Hour,
			keepFailed:   true,
			exitCode:     1,
			expectDelete: false,
		},
		{
			name:         "recent completed run is not deleted",
			status:       storage.StatusCompleted,
			startTime:    recent,
			olderThan:    24 * time.Hour,
			expectDelete: false,
		},
		{
			name:         "running run is never deleted",
			status:       storage.StatusRunning,
			startTime:    old,
			olderThan:    24 * time.Hour,
			expectDelete: false,
		},
		{
			name:         "zero older-than deletes nothing recent",
			status:       storage.StatusCompleted,
			startTime:    recent,
			olderThan:    0,
			expectDelete: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			runDir := makeRun(t, root, "proj", "task1", "run-test", tc.status, tc.startTime, tc.exitCode)

			if err := runGC(root, "", tc.olderThan, false, tc.keepFailed); err != nil {
				t.Fatalf("runGC: %v", err)
			}

			_, statErr := os.Stat(runDir)
			if tc.expectDelete && !os.IsNotExist(statErr) {
				t.Errorf("expected run dir to be deleted, but it still exists")
			}
			if !tc.expectDelete && statErr != nil {
				t.Errorf("expected run dir to still exist, but got: %v", statErr)
			}
		})
	}
}

func TestGCDryRunDoesNotDelete(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	runDir := makeRun(t, root, "proj", "task1", "run-old", storage.StatusCompleted, old, 0)

	var output string
	output = captureStdout(t, func() {
		if err := runGC(root, "", 24*time.Hour, true, false); err != nil {
			t.Errorf("runGC dry-run: %v", err)
		}
	})

	if _, err := os.Stat(runDir); err != nil {
		t.Errorf("expected run dir to still exist in dry-run: %v", err)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected '[dry-run]' in output, got: %q", output)
	}
	if !strings.Contains(output, "Would delete") {
		t.Errorf("expected 'Would delete' in summary, got: %q", output)
	}
}

func TestGCProjectFilter(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)

	proj1Dir := makeRun(t, root, "proj1", "task1", "run-old", storage.StatusCompleted, old, 0)
	proj2Dir := makeRun(t, root, "proj2", "task1", "run-old", storage.StatusCompleted, old, 0)

	if err := runGC(root, "proj1", 24*time.Hour, false, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	if _, err := os.Stat(proj1Dir); !os.IsNotExist(err) {
		t.Errorf("expected proj1 run dir to be deleted")
	}
	if _, err := os.Stat(proj2Dir); err != nil {
		t.Errorf("expected proj2 run dir to still exist: %v", err)
	}
}

func TestGCSkipsMissingRunInfo(t *testing.T) {
	root := t.TempDir()
	// create a run dir without run-info.yaml
	runDir := filepath.Join(root, "proj", "task1", "runs", "run-no-info")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := runGC(root, "", 24*time.Hour, false, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	if _, err := os.Stat(runDir); err != nil {
		t.Errorf("expected run dir without run-info.yaml to still exist: %v", err)
	}
}

func TestGCSummaryOutput(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)

	makeRun(t, root, "proj", "task1", "run-old1", storage.StatusCompleted, old, 0)
	makeRun(t, root, "proj", "task1", "run-old2", storage.StatusCompleted, old, 0)

	var output string
	output = captureStdout(t, func() {
		if err := runGC(root, "", 24*time.Hour, false, false); err != nil {
			t.Errorf("runGC: %v", err)
		}
	})

	if !strings.Contains(output, "Deleted 2 runs") {
		t.Errorf("expected 'Deleted 2 runs' in summary, got: %q", output)
	}
}

func TestGCMultipleTasks(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)

	dir1 := makeRun(t, root, "proj", "task1", "run-old", storage.StatusCompleted, old, 0)
	dir2 := makeRun(t, root, "proj", "task2", "run-old", storage.StatusCompleted, old, 0)

	if err := runGC(root, "", 24*time.Hour, false, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	if _, err := os.Stat(dir1); !os.IsNotExist(err) {
		t.Errorf("expected task1 run dir to be deleted")
	}
	if _, err := os.Stat(dir2); !os.IsNotExist(err) {
		t.Errorf("expected task2 run dir to be deleted")
	}
}

func TestGCCmd_Integration(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)

	makeRun(t, root, "proj", "task1", "run-old", storage.StatusCompleted, old, 0)

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"gc",
		"--root", root,
		"--older-than", "24h",
		"--project", "proj",
	})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("gc command failed: %v", runErr)
	}
	if !strings.Contains(output, "Deleted 1 runs") {
		t.Errorf("expected 'Deleted 1 runs', got: %q", output)
	}
}
