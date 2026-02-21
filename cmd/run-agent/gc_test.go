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

// makeBusFile creates a fake message bus file of the given size in bytes.
func makeBusFile(t *testing.T, path string, sizeBytes int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for bus file: %v", err)
	}
	data := make([]byte, sizeBytes)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write bus file: %v", err)
	}
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

			if err := runGC(root, "", tc.olderThan, false, tc.keepFailed, false, 0, false); err != nil {
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
		if err := runGC(root, "", 24*time.Hour, true, false, false, 0, false); err != nil {
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

	if err := runGC(root, "proj1", 24*time.Hour, false, false, false, 0, false); err != nil {
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

	if err := runGC(root, "", 24*time.Hour, false, false, false, 0, false); err != nil {
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
		if err := runGC(root, "", 24*time.Hour, false, false, false, 0, false); err != nil {
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

	if err := runGC(root, "", 24*time.Hour, false, false, false, 0, false); err != nil {
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

// --- Bus rotation tests ---

func TestGCRotateBus_LargeFileGetsRotated(t *testing.T) {
	root := t.TempDir()
	taskBus := filepath.Join(root, "proj", "task1", "TASK-MESSAGE-BUS.md")
	// 2MB file, threshold 1MB
	makeBusFile(t, taskBus, 2*1024*1024)

	var output string
	output = captureStdout(t, func() {
		if err := runGC(root, "", 168*time.Hour, false, false, true, 1*1024*1024, false); err != nil {
			t.Fatalf("runGC: %v", err)
		}
	})

	// Original file should be gone
	if _, err := os.Stat(taskBus); !os.IsNotExist(err) {
		t.Errorf("expected bus file to be rotated (renamed), but it still exists at original path")
	}

	// An archived file should exist
	entries, err := os.ReadDir(filepath.Join(root, "proj", "task1"))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".archived") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected an .archived file in task directory")
	}

	if !strings.Contains(output, "Rotated") {
		t.Errorf("expected 'Rotated' in output, got: %q", output)
	}
}

func TestGCRotateBus_SmallFileNotRotated(t *testing.T) {
	root := t.TempDir()
	taskBus := filepath.Join(root, "proj", "task1", "TASK-MESSAGE-BUS.md")
	// 512KB file, threshold 1MB — should NOT be rotated
	makeBusFile(t, taskBus, 512*1024)

	if err := runGC(root, "", 168*time.Hour, false, false, true, 1*1024*1024, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// File should still exist at original path
	if _, err := os.Stat(taskBus); err != nil {
		t.Errorf("expected small bus file to remain, but got: %v", err)
	}
}

func TestGCRotateBus_DryRunDoesNotRotate(t *testing.T) {
	root := t.TempDir()
	taskBus := filepath.Join(root, "proj", "task1", "TASK-MESSAGE-BUS.md")
	// 2MB file, threshold 1MB
	makeBusFile(t, taskBus, 2*1024*1024)

	var output string
	output = captureStdout(t, func() {
		if err := runGC(root, "", 168*time.Hour, true, false, true, 1*1024*1024, false); err != nil {
			t.Fatalf("runGC: %v", err)
		}
	})

	// File should still exist (dry-run, no rename)
	if _, err := os.Stat(taskBus); err != nil {
		t.Errorf("expected bus file to still exist in dry-run: %v", err)
	}

	// Should report would-rotate
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected '[dry-run]' in output, got: %q", output)
	}
	if !strings.Contains(output, "would rotate") {
		t.Errorf("expected 'would rotate' in output, got: %q", output)
	}
}

func TestGCRotateBus_ThresholdRespected(t *testing.T) {
	root := t.TempDir()

	// bus1: 3MB — should be rotated with 2MB threshold
	bus1 := filepath.Join(root, "proj", "task1", "TASK-MESSAGE-BUS.md")
	makeBusFile(t, bus1, 3*1024*1024)

	// bus2: 1MB — should NOT be rotated with 2MB threshold
	bus2 := filepath.Join(root, "proj", "task2", "TASK-MESSAGE-BUS.md")
	makeBusFile(t, bus2, 1*1024*1024)

	if err := runGC(root, "", 168*time.Hour, false, false, true, 2*1024*1024, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// bus1 should be gone (rotated)
	if _, err := os.Stat(bus1); !os.IsNotExist(err) {
		t.Errorf("expected bus1 (3MB) to be rotated with 2MB threshold")
	}

	// bus2 should remain (below threshold)
	if _, err := os.Stat(bus2); err != nil {
		t.Errorf("expected bus2 (1MB) to remain with 2MB threshold: %v", err)
	}
}

func TestGCRotateBus_ProjectBusFileRotated(t *testing.T) {
	root := t.TempDir()
	projBus := filepath.Join(root, "proj", "PROJECT-MESSAGE-BUS.md")
	// 2MB project bus file, threshold 1MB
	makeBusFile(t, projBus, 2*1024*1024)

	if err := runGC(root, "", 168*time.Hour, false, false, true, 1*1024*1024, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// Original file should be gone
	if _, err := os.Stat(projBus); !os.IsNotExist(err) {
		t.Errorf("expected project bus file to be rotated")
	}

	// An archived file should exist in proj dir
	entries, err := os.ReadDir(filepath.Join(root, "proj"))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".archived") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected an .archived file in project directory")
	}
}

func TestGCRotateBus_NotRotatedWhenFlagAbsent(t *testing.T) {
	root := t.TempDir()
	taskBus := filepath.Join(root, "proj", "task1", "TASK-MESSAGE-BUS.md")
	// Large file but --rotate-bus not set
	makeBusFile(t, taskBus, 20*1024*1024)

	if err := runGC(root, "", 168*time.Hour, false, false, false, 10*1024*1024, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// File should still exist (rotation not requested)
	if _, err := os.Stat(taskBus); err != nil {
		t.Errorf("expected bus file to remain when --rotate-bus not set: %v", err)
	}
}

// --- delete-done-tasks tests ---

// makeTaskDir creates a task directory structure for testing.
// If withDone is true, creates a DONE file. If withRuns is true, creates a runs/ subdir with one run dir.
func makeTaskDir(t *testing.T, root, project, task string, withDone bool, withRuns bool) string {
	t.Helper()
	taskDir := filepath.Join(root, project, task)
	runsDir := filepath.Join(taskDir, "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runsDir, err)
	}
	if withDone {
		if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte("done"), 0o644); err != nil {
			t.Fatalf("write DONE: %v", err)
		}
	}
	if withRuns {
		runDir := filepath.Join(runsDir, "run-001")
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("mkdir run dir: %v", err)
		}
	}
	return taskDir
}

func TestGCDeleteDoneTasksNoFlag(t *testing.T) {
	root := t.TempDir()
	// Task with DONE file and empty runs/ — but flag not set
	taskDir := makeTaskDir(t, root, "proj", "task-old-done", true, false)
	// Make task dir appear old
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(taskDir, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	if err := runGC(root, "", 24*time.Hour, false, false, false, 0, false); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// Task dir should still exist (flag not set)
	if _, err := os.Stat(taskDir); err != nil {
		t.Errorf("expected task dir to still exist when --delete-done-tasks not set: %v", err)
	}
}

func TestGCDeleteDoneTasksWithDONE(t *testing.T) {
	root := t.TempDir()
	// Task with DONE file and empty runs/
	taskDir := makeTaskDir(t, root, "proj", "task-old-done", true, false)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(taskDir, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	var output string
	output = captureStdout(t, func() {
		if err := runGC(root, "", 24*time.Hour, false, false, false, 0, true); err != nil {
			t.Fatalf("runGC: %v", err)
		}
	})

	// Task dir should be gone
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Errorf("expected task dir to be deleted, but it still exists")
	}
	if !strings.Contains(output, "Deleted 1 task directories") {
		t.Errorf("expected 'Deleted 1 task directories' in output, got: %q", output)
	}
}

func TestGCDeleteDoneTasksNoDONE(t *testing.T) {
	root := t.TempDir()
	// Task WITHOUT DONE file
	taskDir := makeTaskDir(t, root, "proj", "task-old-nodone", false, false)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(taskDir, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	if err := runGC(root, "", 24*time.Hour, false, false, false, 0, true); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// Task dir should still exist (no DONE file)
	if _, err := os.Stat(taskDir); err != nil {
		t.Errorf("expected task dir without DONE to still exist: %v", err)
	}
}

func TestGCDeleteDoneTasksDryRun(t *testing.T) {
	root := t.TempDir()
	taskDir := makeTaskDir(t, root, "proj", "task-old-done", true, false)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(taskDir, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	var output string
	output = captureStdout(t, func() {
		if err := runGC(root, "", 24*time.Hour, true, false, false, 0, true); err != nil {
			t.Fatalf("runGC dry-run: %v", err)
		}
	})

	// Task dir should NOT be deleted in dry-run mode
	if _, err := os.Stat(taskDir); err != nil {
		t.Errorf("expected task dir to still exist in dry-run: %v", err)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected '[dry-run]' in output, got: %q", output)
	}
	if !strings.Contains(output, "would delete task dir") {
		t.Errorf("expected 'would delete task dir' in output, got: %q", output)
	}
}

func TestGCDeleteDoneTasksActiveRuns(t *testing.T) {
	root := t.TempDir()
	// Task with DONE file but non-empty runs/
	taskDir := makeTaskDir(t, root, "proj", "task-old-active", true, true)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(taskDir, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	if err := runGC(root, "", 24*time.Hour, false, false, false, 0, true); err != nil {
		t.Fatalf("runGC: %v", err)
	}

	// Task dir should still exist (non-empty runs/)
	if _, err := os.Stat(taskDir); err != nil {
		t.Errorf("expected task dir with active runs to still exist: %v", err)
	}
}

// --- parseSizeBytes tests ---

func TestParseSizeBytes(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"10MB", 10 * 1024 * 1024, false},
		{"5MB", 5 * 1024 * 1024, false},
		{"100KB", 100 * 1024, false},
		{"1GB", 1 * 1024 * 1024 * 1024, false},
		{"512B", 512, false},
		{"1024", 1024, false},
		{"", 0, true},
		{"-1MB", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseSizeBytes(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("parseSizeBytes(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
