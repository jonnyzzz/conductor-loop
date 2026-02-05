package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestLoadConfigAndCreateRun(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.yaml")
	cfgBody := `
agents:
  codex:
    type: codex
    token: test-token
defaults:
  agent: codex
  timeout: 30
api:
  host: "127.0.0.1"
  port: 0
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Defaults.Agent != "codex" {
		t.Fatalf("unexpected default agent: %q", cfg.Defaults.Agent)
	}

	store, err := storage.NewStorage(root)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	info, err := store.CreateRun("project", "task-001", cfg.Defaults.Agent)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	runInfoPath := filepath.Join(root, "project", "task-001", "runs", info.RunID, "run-info.yaml")
	loaded, err := storage.ReadRunInfo(runInfoPath)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if loaded.ProjectID != "project" || loaded.TaskID != "task-001" {
		t.Fatalf("unexpected run info ids: %+v", loaded)
	}
	if loaded.AgentType != "codex" {
		t.Fatalf("unexpected agent type: %q", loaded.AgentType)
	}
	if loaded.Status != storage.StatusRunning {
		t.Fatalf("unexpected status: %q", loaded.Status)
	}
	if loaded.StartTime.IsZero() {
		t.Fatalf("expected start time to be set")
	}
}

func TestRunInfoPersistenceAcrossRestarts(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewStorage(root)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	info, err := store.CreateRun("project", "task-002", "codex")
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := store.UpdateRunStatus(info.RunID, storage.StatusCompleted, 0); err != nil {
		t.Fatalf("update status: %v", err)
	}

	restarted, err := storage.NewStorage(root)
	if err != nil {
		t.Fatalf("new storage after restart: %v", err)
	}
	loaded, err := restarted.GetRunInfo(info.RunID)
	if err != nil {
		t.Fatalf("get run info: %v", err)
	}
	if loaded.Status != storage.StatusCompleted {
		t.Fatalf("unexpected status: %q", loaded.Status)
	}
	if loaded.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", loaded.ExitCode)
	}
	if loaded.EndTime.IsZero() {
		t.Fatalf("expected end time to be set")
	}
}
