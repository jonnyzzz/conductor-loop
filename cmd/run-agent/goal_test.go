package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/goaldecompose"
	"gopkg.in/yaml.v3"
)

func TestGoalDecomposeRequiresGoalInput(t *testing.T) {
	t.Setenv("JRUN_PROJECT_ID", "my-project")
	cmd := newRootCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"goal", "decompose"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when goal input is missing")
	}
	if !strings.Contains(err.Error(), "one of --goal or --goal-file is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoalDecomposeRejectsMixedGoalInputs(t *testing.T) {
	t.Setenv("JRUN_PROJECT_ID", "my-project")
	goalFile := filepath.Join(t.TempDir(), "goal.md")
	if err := os.WriteFile(goalFile, []byte("goal from file"), 0o644); err != nil {
		t.Fatalf("write goal file: %v", err)
	}

	cmd := newRootCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"goal", "decompose",
		"--goal", "inline goal",
		"--goal-file", goalFile,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for mixed --goal and --goal-file")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoalDecomposeJSONAndOutYAML(t *testing.T) {
	t.Setenv("JRUN_PROJECT_ID", "my-project")
	goalFile := filepath.Join(t.TempDir(), "GOAL.md")
	if err := os.WriteFile(goalFile, []byte("Ship deterministic goal decomposition"), 0o644); err != nil {
		t.Fatalf("write goal file: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "workflow.yaml")

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"goal", "decompose",
		"--goal-file", goalFile,
		"--json",
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var stdoutSpec goaldecompose.WorkflowSpec
	if err := json.Unmarshal(stdout.Bytes(), &stdoutSpec); err != nil {
		t.Fatalf("stdout is not valid json: %v\n%s", err, stdout.String())
	}
	if stdoutSpec.WorkflowID == "" {
		t.Fatal("stdout workflow_id is empty")
	}

	fileData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var fileSpec goaldecompose.WorkflowSpec
	if err := yaml.Unmarshal(fileData, &fileSpec); err != nil {
		t.Fatalf("file is not valid yaml: %v\n%s", err, string(fileData))
	}
	if fileSpec.WorkflowID != stdoutSpec.WorkflowID {
		t.Fatalf("workflow_id mismatch: file=%q stdout=%q", fileSpec.WorkflowID, stdoutSpec.WorkflowID)
	}
	if len(fileSpec.Tasks) == 0 {
		t.Fatal("expected generated tasks in output file")
	}
}

func TestGoalDecomposeOutFormatInferenceJSON(t *testing.T) {
	t.Setenv("JRUN_PROJECT_ID", "my-project")
	outPath := filepath.Join(t.TempDir(), "workflow.json")

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"goal", "decompose",
		"--goal", "Deliver deterministic CLI output",
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "schema_version:") {
		t.Fatalf("expected yaml stdout, got: %s", stdout.String())
	}

	fileData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var fileSpec goaldecompose.WorkflowSpec
	if err := json.Unmarshal(fileData, &fileSpec); err != nil {
		t.Fatalf("file is not valid json: %v\n%s", err, string(fileData))
	}
	if fileSpec.SchemaVersion != goaldecompose.SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", fileSpec.SchemaVersion, goaldecompose.SchemaVersion)
	}
}
