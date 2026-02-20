package main

import (
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRunAgentTaskValidation(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error for missing project")
	}
}

func TestRunAgentJobValidation(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error for missing project")
	}
}

func TestResolveTaskIDAutoGenerates(t *testing.T) {
	id, err := resolveTaskID("")
	if err != nil {
		t.Fatalf("resolveTaskID empty: %v", err)
	}
	if err := storage.ValidateTaskID(id); err != nil {
		t.Errorf("auto-generated task ID %q failed validation: %v", id, err)
	}
	if !strings.HasPrefix(id, "task-") {
		t.Errorf("expected task ID to start with 'task-', got %q", id)
	}
}

func TestResolveTaskIDAcceptsValid(t *testing.T) {
	valid := "task-20260220-153045-my-feature"
	id, err := resolveTaskID(valid)
	if err != nil {
		t.Fatalf("resolveTaskID valid: %v", err)
	}
	if id != valid {
		t.Errorf("expected %q, got %q", valid, id)
	}
}

func TestResolveTaskIDRejectsInvalid(t *testing.T) {
	invalid := []string{
		"my-task",
		"task-foo",
		"random-string",
		"task-2026022-153045-slug",
		"task-20260220-15304-slug",
	}
	for _, id := range invalid {
		_, err := resolveTaskID(id)
		if err == nil {
			t.Errorf("expected error for invalid task ID %q", id)
		}
	}
}

func TestJobAutoGeneratesTaskID(t *testing.T) {
	// Providing --project but no --task should auto-generate a valid task ID.
	// The command will fail at runner.RunJob (no agent configured), but we can
	// verify that the error is NOT about task ID format.
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job", "--project", "my-project"})
	err := cmd.Execute()
	// We expect an error because no agent is configured, but NOT a task ID error.
	if err != nil && strings.Contains(err.Error(), "invalid task ID") {
		t.Errorf("unexpected task ID validation error: %v", err)
	}
}

func TestJobRejectsInvalidTaskID(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job", "--project", "my-project", "--task", "invalid-task-id"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for invalid task ID")
	}
	if !strings.Contains(err.Error(), "invalid task ID") {
		t.Errorf("expected 'invalid task ID' in error, got: %v", err)
	}
}
