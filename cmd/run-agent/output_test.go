package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// writeFile writes content to a file, creating parent dirs as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestRunOutput_OutputMD(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "output.md"), "hello from output.md\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "", "output", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "hello from output.md") {
		t.Errorf("expected output to contain 'hello from output.md', got: %q", out)
	}
}

func TestRunOutput_FallbackToStdout(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	// No output.md — only agent-stdout.txt
	writeFile(t, filepath.Join(runDir, "agent-stdout.txt"), "fallback stdout content\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "", "output", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "fallback stdout content") {
		t.Errorf("expected fallback stdout content, got: %q", out)
	}
}

func TestRunOutput_TailFlag(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, strings.Repeat("x", i)) // distinct lines
	}
	content := strings.Join(lines, "\n") + "\n"
	writeFile(t, filepath.Join(runDir, "output.md"), content)

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "", "output", 3)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	// Should contain last 3 lines (lines 8, 9, 10 — i.e. 8, 9, 10 x's)
	for i := 8; i <= 10; i++ {
		want := strings.Repeat("x", i)
		if !strings.Contains(out, want) {
			t.Errorf("expected tail output to contain %q, got: %q", want, out)
		}
	}
	// Should NOT contain first lines
	for i := 1; i <= 7; i++ {
		notWant := strings.Repeat("x", i) + "\n"
		// Check exactly (avoid false match of longer lines containing shorter prefix)
		_ = notWant
	}
	// Simpler check: output should have exactly 3 lines
	outLines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(outLines) != 3 {
		t.Errorf("expected 3 lines with --tail 3, got %d: %q", len(outLines), out)
	}
}

func TestRunOutput_NonExistentProject(t *testing.T) {
	root := t.TempDir()

	err := runOutput("", root, "nonexistent-proj", "task1", "", "output", 0)
	if err == nil {
		t.Fatal("expected error for non-existent project/task")
	}
	if !strings.Contains(err.Error(), "nonexistent-proj") {
		t.Errorf("expected error to mention project name, got: %v", err)
	}
}

func TestRunOutput_RunDirFlag(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "output.md"), "direct rundir output\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput(runDir, "", "", "", "", "output", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "direct rundir output") {
		t.Errorf("expected 'direct rundir output', got: %q", out)
	}
}

func TestRunOutput_StdoutFile(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "agent-stdout.txt"), "raw stdout here\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "", "stdout", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "raw stdout here") {
		t.Errorf("expected 'raw stdout here', got: %q", out)
	}
}

func TestRunOutput_PromptFile(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "prompt.md"), "# My Prompt\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "", "prompt", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "My Prompt") {
		t.Errorf("expected 'My Prompt', got: %q", out)
	}
}

func TestRunOutput_MissingFile(t *testing.T) {
	root := t.TempDir()
	makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	// No output files written

	err := runOutput("", root, "proj", "task1", "", "output", 0)
	if err == nil {
		t.Fatal("expected error when no output files exist")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' in error, got: %v", err)
	}
}

func TestRunOutput_MostRecentRun(t *testing.T) {
	root := t.TempDir()
	// Create two runs; the second (run-002) should be chosen as most recent
	runDir1 := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-2*time.Hour), 0)
	runDir2 := makeRun(t, root, "proj", "task1", "run-002", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir1, "output.md"), "from run-001\n")
	writeFile(t, filepath.Join(runDir2, "output.md"), "from run-002\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "", "output", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "from run-002") {
		t.Errorf("expected most recent run (run-002) output, got: %q", out)
	}
	if strings.Contains(out, "from run-001") {
		t.Errorf("expected run-001 output NOT to be shown, but got: %q", out)
	}
}

func TestRunOutput_SpecificRunID(t *testing.T) {
	root := t.TempDir()
	runDir1 := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-2*time.Hour), 0)
	runDir2 := makeRun(t, root, "proj", "task1", "run-002", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir1, "output.md"), "from run-001\n")
	writeFile(t, filepath.Join(runDir2, "output.md"), "from run-002\n")

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = runOutput("", root, "proj", "task1", "run-001", "output", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "from run-001") {
		t.Errorf("expected run-001 output when --run=run-001, got: %q", out)
	}
}

func TestRunOutput_UnknownFileType(t *testing.T) {
	root := t.TempDir()
	makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)

	err := runOutput("", root, "proj", "task1", "", "invalid-type", 0)
	if err == nil {
		t.Fatal("expected error for unknown --file type")
	}
	if !strings.Contains(err.Error(), "invalid-type") {
		t.Errorf("expected error to mention the unknown type, got: %v", err)
	}
}

func TestOutputCmd_Integration(t *testing.T) {
	root := t.TempDir()
	runDir := makeRun(t, root, "proj", "task1", "run-001", storage.StatusCompleted, time.Now().Add(-time.Hour), 0)
	writeFile(t, filepath.Join(runDir, "output.md"), "integration test output\n")

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"output",
		"--root", root,
		"--project", "proj",
		"--task", "task1",
	})

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("output command failed: %v", runErr)
	}
	if !strings.Contains(out, "integration test output") {
		t.Errorf("expected 'integration test output', got: %q", out)
	}
}
