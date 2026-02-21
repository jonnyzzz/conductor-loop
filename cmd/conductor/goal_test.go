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

func TestConductorGoalDecomposeJSON(t *testing.T) {
	goalFile := filepath.Join(t.TempDir(), "GOAL.md")
	if err := os.WriteFile(goalFile, []byte("Build deterministic workflow specs"), 0o644); err != nil {
		t.Fatalf("write goal file: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "goal.yaml")

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"goal", "decompose",
		"--project", "my-project",
		"--goal-file", goalFile,
		"--json",
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var stdoutSpec goaldecompose.WorkflowSpec
	if err := json.Unmarshal(stdout.Bytes(), &stdoutSpec); err != nil {
		t.Fatalf("stdout json decode: %v", err)
	}
	if stdoutSpec.WorkflowID == "" {
		t.Fatal("expected non-empty workflow_id")
	}

	fileData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	var fileSpec goaldecompose.WorkflowSpec
	if err := yaml.Unmarshal(fileData, &fileSpec); err != nil {
		t.Fatalf("yaml decode: %v", err)
	}
	if fileSpec.WorkflowID != stdoutSpec.WorkflowID {
		t.Fatalf("workflow id mismatch: %q != %q", fileSpec.WorkflowID, stdoutSpec.WorkflowID)
	}
}

func TestConductorGoalDecomposeRequiresGoal(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"goal", "decompose", "--project", "my-project"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing goal input")
	}
	if !strings.Contains(err.Error(), "one of --goal or --goal-file is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
