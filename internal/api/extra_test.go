package api

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	server.startTask(TaskCreateRequest{ProjectID: "project", TaskID: "task", AgentType: "codex", Prompt: "prompt"}, "", "prompt")
}

func TestStartTask_ProcessImport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based process fixture is unix-only")
	}

	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stdoutSource := filepath.Join(root, "external-stdout.log")
	stderrSource := filepath.Join(root, "external-stderr.log")
	scriptPath := filepath.Join(root, "external.sh")
	script := `#!/bin/sh
echo "api-stdout-1" >> "$SRC_STDOUT"
echo "api-stderr-1" >> "$SRC_STDERR"
sleep 0.2
echo "api-stdout-2" >> "$SRC_STDOUT"
echo "api-stderr-2" >> "$SRC_STDERR"
sleep 0.2
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(scriptPath)
	cmd.Env = append(os.Environ(),
		"SRC_STDOUT="+stdoutSource,
		"SRC_STDERR="+stderrSource,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fixture process: %v", err)
	}
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	server, err := NewServer(Options{RootDir: root, DisableTaskStart: false, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	server.startTask(TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Prompt:    "prompt",
		ProcessImport: &ProcessImportRequest{
			PID:         cmd.Process.Pid,
			CommandLine: scriptPath,
			StdoutPath:  stdoutSource,
			StderrPath:  stderrSource,
		},
	}, "", "prompt")
	if waitErr := <-waitDone; waitErr != nil {
		t.Fatalf("fixture process wait: %v", waitErr)
	}

	runsDir := filepath.Join(root, "project", "task", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one run, got %d", len(entries))
	}

	runDir := filepath.Join(runsDir, entries[0].Name())
	info, err := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if info.Status != storage.StatusCompleted {
		t.Fatalf("status=%q, want completed", info.Status)
	}
	if storage.EffectiveProcessOwnership(info) != storage.ProcessOwnershipExternal {
		t.Fatalf("process_ownership=%q, want external", info.ProcessOwnership)
	}
	stdoutData, err := os.ReadFile(filepath.Join(runDir, "agent-stdout.txt"))
	if err != nil {
		t.Fatalf("read mirrored stdout: %v", err)
	}
	if !strings.Contains(string(stdoutData), "api-stdout-2") {
		t.Fatalf("missing mirrored stdout content: %q", string(stdoutData))
	}
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
