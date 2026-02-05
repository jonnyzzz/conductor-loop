package main

import "testing"

func TestRunAgentTaskValidation(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"task"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error for missing project/task")
	}
}

func TestRunAgentJobValidation(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error for missing project/task")
	}
}
