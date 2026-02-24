package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputSynthesize_MultipleRuns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two run directories with output.md
	run1Dir := filepath.Join(tmpDir, "run-001")
	run2Dir := filepath.Join(tmpDir, "run-002")
	for _, d := range []string{run1Dir, run2Dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(run1Dir, "output.md"), []byte("# Run 1\nContent from run 1."), 0o644); err != nil {
		t.Fatalf("write run1 output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(run2Dir, "output.md"), []byte("# Run 2\nContent from run 2."), 0o644); err != nil {
		t.Fatalf("write run2 output: %v", err)
	}

	var buf bytes.Buffer
	err := runOutputSynthesize(&buf, "", "", "", []string{run1Dir, run2Dir}, false)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "run-001") {
		t.Errorf("expected run-001 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "run-002") {
		t.Errorf("expected run-002 in output")
	}
	if !strings.Contains(out, "Content from run 1") {
		t.Errorf("expected run 1 content")
	}
	if !strings.Contains(out, "Content from run 2") {
		t.Errorf("expected run 2 content")
	}
}

func TestOutputSynthesize_FallbackToStdout(t *testing.T) {
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-001")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// No output.md, only agent-stdout.txt
	if err := os.WriteFile(filepath.Join(runDir, "agent-stdout.txt"), []byte("raw stdout content"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	var buf bytes.Buffer
	err := runOutputSynthesize(&buf, "", "", "", []string{runDir}, false)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if !strings.Contains(buf.String(), "raw stdout content") {
		t.Errorf("expected stdout fallback content, got: %s", buf.String())
	}
}

func TestOutputSynthesize_MissingNonStrict(t *testing.T) {
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-missing")
	// Don't create the directory at all.

	var buf bytes.Buffer
	err := runOutputSynthesize(&buf, "", "", "", []string{runDir}, false)
	if err != nil {
		t.Fatalf("non-strict should not fail: %v", err)
	}
	if !strings.Contains(buf.String(), "WARNING") {
		t.Errorf("expected WARNING for missing run, got: %s", buf.String())
	}
}

func TestOutputSynthesize_MissingStrict(t *testing.T) {
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-missing")

	var buf bytes.Buffer
	err := runOutputSynthesize(&buf, "", "", "", []string{runDir}, true)
	if err == nil {
		t.Fatal("strict mode should fail for missing run")
	}
}

func TestOutputSynthesize_ResolveRunDir(t *testing.T) {
	// Absolute path should be used as-is
	result := resolveRunDirForSynthesize("/root", "proj", "task", "/abs/path/run-001")
	if result != "/abs/path/run-001" {
		t.Errorf("expected absolute path unchanged, got %q", result)
	}

	// Relative run ID with root/project/task should resolve
	result = resolveRunDirForSynthesize("/root", "proj", "task-001", "run-002")
	expected := "/root/proj/task-001/runs/run-002"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
