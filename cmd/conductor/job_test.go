package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestJobSubmitBatchSubcommandRegistered(t *testing.T) {
	cmd := newJobCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "submit-batch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected submit-batch subcommand on job command")
	}
}

func TestLoadBatchPromptsFromFlags(t *testing.T) {
	promptFile := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(promptFile, []byte("from file"), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	got, err := loadBatchPromptsFromFlags([]string{"inline"}, []string{promptFile})
	if err != nil {
		t.Fatalf("loadBatchPromptsFromFlags: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(got))
	}
	if got[0] != "inline" {
		t.Fatalf("unexpected inline prompt: %q", got[0])
	}
	if got[1] != "from file" {
		t.Fatalf("unexpected file prompt: %q", got[1])
	}
}

func TestJobSubmitBatchPostsAllPrompts(t *testing.T) {
	type createReq struct {
		ProjectID string `json:"project_id"`
		TaskID    string `json:"task_id"`
		AgentType string `json:"agent_type"`
		Prompt    string `json:"prompt"`
	}

	var (
		mu       sync.Mutex
		requests []createReq
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tasks" {
			http.NotFound(w, r)
			return
		}
		defer r.Body.Close()
		var req createReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		requests = append(requests, req)
		mu.Unlock()

		resp := jobCreateResponse{
			ProjectID: req.ProjectID,
			TaskID:    req.TaskID,
			RunID:     "run-" + req.TaskID,
			Status:    "pending",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"job", "submit-batch",
		"--server", srv.URL,
		"--project", "proj",
		"--agent", "claude",
		"--prompt", "first prompt",
		"--prompt", "second prompt",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requests))
	}
	if requests[0].Prompt != "first prompt" {
		t.Fatalf("unexpected first prompt: %q", requests[0].Prompt)
	}
	if requests[1].Prompt != "second prompt" {
		t.Fatalf("unexpected second prompt: %q", requests[1].Prompt)
	}
	if requests[0].TaskID == "" || requests[1].TaskID == "" {
		t.Fatalf("expected non-empty task IDs, got %q and %q", requests[0].TaskID, requests[1].TaskID)
	}
	if requests[0].TaskID == requests[1].TaskID {
		t.Fatalf("expected unique task IDs, got duplicate %q", requests[0].TaskID)
	}
	if !strings.Contains(stdout.String(), "batch item 1/2 submitted") {
		t.Fatalf("expected batch submit output, got %q", stdout.String())
	}
}

func TestJobSubmitBatchRejectsMismatchedTaskCount(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"job", "submit-batch",
		"--server", "http://localhost:14355",
		"--project", "proj",
		"--agent", "claude",
		"--prompt", "one",
		"--prompt", "two",
		"--task", "task-20260222-010101-one",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for mismatched task count")
	}
	if !strings.Contains(err.Error(), "must match prompt count") {
		t.Fatalf("unexpected error: %v", err)
	}
}
