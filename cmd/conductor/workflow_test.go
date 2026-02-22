package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
)

func TestConductorWorkflowCommandRegistered(t *testing.T) {
	cmd := newRootCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "workflow" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected workflow subcommand on conductor root")
	}
}

func TestConductorWorkflowRunDryRun(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260222-140000-workflow"

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"workflow", "run",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
		"--dry-run",
		"--from-stage", "4",
		"--to-stage", "5",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "workflow state:") {
		t.Fatalf("expected workflow state output, got %q", out)
	}
	if !strings.Contains(out, "dry-run planned stages: [4 5]") {
		t.Fatalf("expected planned stages output, got %q", out)
	}
}

func TestConductorWorkflowRunDryRunJSON(t *testing.T) {
	root := t.TempDir()
	taskID := "task-20260222-150000-workflow"

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"workflow", "run",
		"--project", "my-project",
		"--task", taskID,
		"--root", root,
		"--dry-run",
		"--json",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var result runner.WorkflowResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode json: %v\n%s", err, stdout.String())
	}
	if result.StatePath == "" {
		t.Fatal("expected non-empty state_path")
	}
	if result.Template != runner.WorkflowTemplatePromptV5 {
		t.Fatalf("template = %q, want %q", result.Template, runner.WorkflowTemplatePromptV5)
	}
}
