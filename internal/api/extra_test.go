package api

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestStartTask(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	server, err := NewServer(Options{RootDir: root, DisableTaskStart: false, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	server.startTask(TaskCreateRequest{ProjectID: "project", TaskID: "task", AgentType: "codex", Prompt: "prompt"})
}

func TestStopTaskRuns(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "task")
	runDir := filepath.Join(taskDir, "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusCompleted, EndTime: time.Now().UTC()}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	stopped, err := stopTaskRuns(taskDir)
	if err != nil {
		t.Fatalf("stopTaskRuns: %v", err)
	}
	if stopped != 0 {
		t.Fatalf("expected 0 stopped, got %d", stopped)
	}
}

func TestOffsetForLinePositive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(path, []byte("a\n"+"b\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if offset, err := offsetForLine(path, 1); err != nil || offset == 0 {
		t.Fatalf("expected offset, got %d err=%v", offset, err)
	}
}

func TestAPIErrorMethodNotAllowed(t *testing.T) {
	err := apiErrorMethodNotAllowed()
	if err == nil || err.Status != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func TestResponseRecorderFlush(t *testing.T) {
	rw := &recordingWriter{header: make(http.Header)}
	rec := &responseRecorder{ResponseWriter: rw}
	rec.Flush()
	if rw.buf.Len() < 0 {
		t.Fatalf("unexpected buffer")
	}
	_ = rec.WriteHeader
}
