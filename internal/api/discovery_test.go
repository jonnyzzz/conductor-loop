package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestNewRunDiscoveryValidation(t *testing.T) {
	if _, err := NewRunDiscovery("", 0); err == nil {
		t.Fatalf("expected error for empty root")
	}
}

func TestRunDiscoveryScan(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	discovery, err := NewRunDiscovery(root, time.Second)
	if err != nil {
		t.Fatalf("NewRunDiscovery: %v", err)
	}
	if err := discovery.scan(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	select {
	case runID := <-discovery.NewRuns():
		if runID != "run-1" {
			t.Fatalf("unexpected run id: %q", runID)
		}
	default:
		t.Fatalf("expected run id")
	}
}

func TestListRunIDs(t *testing.T) {
	root := t.TempDir()
	if _, err := listRunIDs(root); err != nil {
		t.Fatalf("listRunIDs: %v", err)
	}
}
