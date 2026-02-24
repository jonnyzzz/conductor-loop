package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

// mkBus creates the parent directory and returns a new MessageBus at the given path.
func mkBus(t *testing.T, busPath string) *messagebus.MessageBus {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(busPath), err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus %s: %v", busPath, err)
	}
	return bus
}

// appendMsg appends a message to the bus, failing the test on error.
func appendMsg(t *testing.T, bus *messagebus.MessageBus, msg *messagebus.Message) {
	t.Helper()
	if _, err := bus.AppendMessage(msg); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
}

func TestRunReviewQuorum_QuorumMet(t *testing.T) {
	root := t.TempDir()
	projectID := "test-project"
	taskID := "test-task"
	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	bus := mkBus(t, busPath)

	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-1", Body: "APPROVED — looks great"})
	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-2", Body: "APPROVED — all checks pass"})

	var out bytes.Buffer
	err := runReviewQuorum(&out, root, projectID, taskID, []string{"run-1", "run-2"}, 2)
	if err != nil {
		t.Fatalf("expected quorum met, got error: %v", err)
	}
	if !strings.Contains(out.String(), "QUORUM MET") {
		t.Errorf("expected 'QUORUM MET' in output, got:\n%s", out.String())
	}
}

func TestRunReviewQuorum_InsufficientApprovals(t *testing.T) {
	root := t.TempDir()
	projectID := "test-project"
	taskID := "test-task"
	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	bus := mkBus(t, busPath)

	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-1", Body: "LGTM"})

	var out bytes.Buffer
	err := runReviewQuorum(&out, root, projectID, taskID, []string{"run-1"}, 2)
	if err == nil {
		t.Fatal("expected error for insufficient approvals, got nil")
	}
	if !strings.Contains(err.Error(), "quorum not met") {
		t.Errorf("expected 'quorum not met' in error, got: %v", err)
	}
	if !strings.Contains(out.String(), "QUORUM NOT MET") {
		t.Errorf("expected 'QUORUM NOT MET' in output, got:\n%s", out.String())
	}
}

func TestRunReviewQuorum_RejectionVeto(t *testing.T) {
	root := t.TempDir()
	projectID := "test-project"
	taskID := "test-task"
	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	bus := mkBus(t, busPath)

	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-1", Body: "APPROVED"})
	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-2", Body: "APPROVED"})
	appendMsg(t, bus, &messagebus.Message{Type: "DECISION", ProjectID: projectID, TaskID: taskID, RunID: "run-3", Body: "REJECTED — critical bug found"})

	var out bytes.Buffer
	err := runReviewQuorum(&out, root, projectID, taskID, []string{"run-1", "run-2", "run-3"}, 2)
	if err == nil {
		t.Fatal("expected error for rejection veto, got nil")
	}
	if !strings.Contains(err.Error(), "rejection veto") {
		t.Errorf("expected 'rejection veto' in error, got: %v", err)
	}
}

func TestRunReviewQuorum_FiltersNonReviewMessages(t *testing.T) {
	root := t.TempDir()
	projectID := "test-project"
	taskID := "test-task"
	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	bus := mkBus(t, busPath)

	// PROGRESS and FACT messages with approval tokens should NOT be counted.
	appendMsg(t, bus, &messagebus.Message{Type: "PROGRESS", ProjectID: projectID, TaskID: taskID, RunID: "run-1", Body: "APPROVED checkpoint passed"})
	appendMsg(t, bus, &messagebus.Message{Type: "FACT", ProjectID: projectID, TaskID: taskID, RunID: "run-2", Body: "LGTM confirmed"})

	var out bytes.Buffer
	err := runReviewQuorum(&out, root, projectID, taskID, []string{"run-1", "run-2"}, 2)
	if err == nil {
		t.Fatal("expected error for no qualifying approvals (PROGRESS/FACT not counted), got nil")
	}
}

func TestRunReviewQuorum_FiltersRunIDs(t *testing.T) {
	root := t.TempDir()
	projectID := "test-project"
	taskID := "test-task"
	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	bus := mkBus(t, busPath)

	// run-99 is approved but NOT in the requested run ID list.
	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-99", Body: "APPROVED"})
	// run-1 is the only one in our list, with only 1 approval (need 2).
	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, TaskID: taskID, RunID: "run-1", Body: "APPROVED"})

	var out bytes.Buffer
	err := runReviewQuorum(&out, root, projectID, taskID, []string{"run-1"}, 2)
	if err == nil {
		t.Fatal("expected error: run-99 approval should be filtered out, leaving only 1")
	}
}

func TestRunReviewQuorum_ProjectBus(t *testing.T) {
	root := t.TempDir()
	projectID := "test-project"
	busPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	bus := mkBus(t, busPath)

	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, RunID: "run-1", Body: "APPROVED"})
	appendMsg(t, bus, &messagebus.Message{Type: "REVIEW", ProjectID: projectID, RunID: "run-2", Body: "APPROVED"})

	var out bytes.Buffer
	// taskID="" → uses PROJECT-MESSAGE-BUS.md
	err := runReviewQuorum(&out, root, projectID, "", []string{"run-1", "run-2"}, 2)
	if err != nil {
		t.Fatalf("expected quorum met on project bus, got: %v", err)
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s      string
		tokens []string
		want   bool
	}{
		{"APPROVED — LOOKS GOOD", []string{"APPROVED", "LGTM"}, true},
		{"LGTM FROM ME", []string{"APPROVED", "LGTM"}, true},
		{"+1 SHIP IT", []string{"+1"}, true},
		{"REJECTED BUILD", []string{"REJECTED"}, true},
		{"NOTHING HERE", []string{"APPROVED", "LGTM"}, false},
		{"", []string{"APPROVED"}, false},
	}
	for _, tc := range tests {
		// containsAny expects upper-cased input.
		got := containsAny(tc.s, tc.tokens)
		if got != tc.want {
			t.Errorf("containsAny(%q, %v) = %v, want %v", tc.s, tc.tokens, got, tc.want)
		}
	}
}

func TestReviewCmdHelp(t *testing.T) {
	cmd := newReviewCmd()
	if cmd.Use != "review" {
		t.Errorf("expected Use=review, got %q", cmd.Use)
	}
	subs := cmd.Commands()
	if len(subs) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(subs))
	}
	if subs[0].Use != "quorum" {
		t.Errorf("expected subcommand Use=quorum, got %q", subs[0].Use)
	}
}

func TestReviewQuorumCmd_RequiredFlags(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"review", "quorum", "--runs", "run-1"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --project is missing")
	}
	if !strings.Contains(err.Error(), "--project") {
		t.Errorf("expected error about --project, got: %v", err)
	}
}
