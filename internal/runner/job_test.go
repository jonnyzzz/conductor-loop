package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestResolvePrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	text, err := resolvePrompt(JobOptions{PromptPath: path})
	if err != nil || text != "hello" {
		t.Fatalf("unexpected prompt: %q err=%v", text, err)
	}
	text, err = resolvePrompt(JobOptions{Prompt: "inline"})
	if err != nil || text != "inline" {
		t.Fatalf("unexpected inline prompt: %q err=%v", text, err)
	}
	if _, err := resolvePrompt(JobOptions{}); err == nil {
		t.Fatalf("expected error for empty prompt")
	}
}

func TestCommandForAgent(t *testing.T) {
	cmd, args, err := commandForAgent("codex", "/tmp")
	if err != nil || cmd == "" || len(args) == 0 {
		t.Fatalf("codex command error: %v", err)
	}
	cmd, args, err = commandForAgent("claude", "/tmp")
	if err != nil || cmd != "claude" || len(args) == 0 {
		t.Fatalf("claude command error: %v", err)
	}
	cmd, args, err = commandForAgent("gemini", "")
	if err != nil || cmd != "gemini" || len(args) == 0 {
		t.Fatalf("gemini command error: %v", err)
	}
	if _, _, err := commandForAgent("unknown", ""); err == nil {
		t.Fatalf("expected error for unknown agent")
	}
}

func TestEnvMap(t *testing.T) {
	values := envMap([]string{"A=1", "B=2"})
	if values["A"] != "1" || values["B"] != "2" {
		t.Fatalf("unexpected env map: %+v", values)
	}
}

func TestIsRestAgent(t *testing.T) {
	if !isRestAgent("perplexity") || !isRestAgent("xai") {
		t.Fatalf("expected rest agents")
	}
	if isRestAgent("codex") {
		t.Fatalf("expected non-rest agent")
	}
}

func TestFinalizeRun(t *testing.T) {
	runDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runDir, "agent-stdout.txt"), []byte("output"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		StartTime: time.Now().UTC(),
		ExitCode:  -1,
		Status:    storage.StatusRunning,
	}
	path := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	if err := finalizeRun(runDir, "", info, nil); err != nil {
		t.Fatalf("finalizeRun: %v", err)
	}
	updated, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if updated.Status != storage.StatusCompleted || updated.ExitCode != 0 {
		t.Fatalf("unexpected status: %+v", updated)
	}
}

func TestPostRunEvent(t *testing.T) {
	info := &storage.RunInfo{RunID: "run", ProjectID: "project", TaskID: "task"}
	busPath := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	if err := postRunEvent(busPath, info, "INFO", "hello"); err != nil {
		t.Fatalf("postRunEvent: %v", err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestRunJobCLI(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	info, err := runJob("project", "task", JobOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}
	if info == nil || info.RunID == "" {
		t.Fatalf("expected run info")
	}
	if _, err := os.Stat(filepath.Join(root, "project", "task", "runs", info.RunID, "output.md")); err != nil {
		t.Fatalf("expected output.md: %v", err)
	}
}

func TestRunJobExported(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunJob("project", "task", JobOptions{RootDir: root, Agent: "codex", Prompt: "hello"}); err != nil {
		t.Fatalf("RunJob: %v", err)
	}
}

func TestRunJobWithConfig(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	configPath := filepath.Join(root, "config.yaml")
	configContent := `agents:
  codex:
    type: codex
    token: token

defaults:
  agent: codex
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := runJob("project", "task", JobOptions{
		RootDir:    root,
		ConfigPath: configPath,
		Agent:      "codex",
		Prompt:     "hello",
	}); err != nil {
		t.Fatalf("runJob: %v", err)
	}
}

func TestRunJobRESTAgent(t *testing.T) {
	root := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	configPath := filepath.Join(root, "config.yaml")
	configContent := fmt.Sprintf(`agents:
  xai:
    type: xai
    token: token
    base_url: %s

defaults:
  agent: xai
  timeout: 10
`, srv.URL)
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	info, err := runJob("project", "task", JobOptions{
		RootDir:    root,
		ConfigPath: configPath,
		Agent:      "xai",
		Prompt:     "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}
	if info.Status != storage.StatusCompleted {
		t.Fatalf("expected completed status, got %q", info.Status)
	}
}

func TestCtxOrBackground(t *testing.T) {
	ctx := ctxOrBackground()
	if ctx == nil || ctx.Err() != nil {
		t.Fatalf("expected background context")
	}
}

func createFakeCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nmore >nul\r\necho stdout\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\ncat >/dev/null\necho stdout\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func createFailingCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nexit /b 1\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func TestRunJobErrors(t *testing.T) {
	if _, err := runJob("", "task", JobOptions{}); err == nil {
		t.Fatalf("expected error for missing project")
	}
	if _, err := runJob("project", "", JobOptions{}); err == nil {
		t.Fatalf("expected error for missing task")
	}
	if _, err := runJob("project", "task", JobOptions{Prompt: strings.Repeat(" ", 2)}); err == nil {
		t.Fatalf("expected error for empty prompt")
	}
}

func TestRunJobCLIExitFailure(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFailingCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	info, err := runJob("project", "task", JobOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  "hello",
	})
	if err == nil {
		t.Fatalf("expected error from failing agent")
	}
	if info == nil || info.Status != storage.StatusFailed {
		t.Fatalf("expected failed run info, got %+v", info)
	}
}

func TestExecuteCLICommandError(t *testing.T) {
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", AgentType: "unknown"}
	if err := executeCLI(context.Background(), "unknown", "prompt.md", t.TempDir(), nil, t.TempDir(), "", info); err == nil {
		t.Fatalf("expected error for unknown agent type")
	}
}

func TestExecuteCLISpawnError(t *testing.T) {
	runDir := t.TempDir()
	promptPath := filepath.Join(runDir, "prompt.md")
	if err := os.WriteFile(promptPath, []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
	}
	env := []string{"PATH=" + t.TempDir()}
	if err := executeCLI(context.Background(), "codex", promptPath, runDir, env, runDir, "", info); err == nil {
		t.Fatalf("expected spawn error")
	}
	updated, err := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if updated.Status != storage.StatusFailed {
		t.Fatalf("expected failed status, got %q", updated.Status)
	}
}

func TestExecuteCLIPostRunEventError(t *testing.T) {
	runDir := t.TempDir()
	binDir := filepath.Join(runDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")

	promptPath := filepath.Join(runDir, "prompt.md")
	if err := os.WriteFile(promptPath, []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	blocker := filepath.Join(runDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	busPath := filepath.Join(blocker, "TASK-MESSAGE-BUS.md")

	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
	}
	env := []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH")}
	if err := executeCLI(context.Background(), "codex", promptPath, runDir, env, runDir, busPath, info); err == nil {
		t.Fatalf("expected postRunEvent error")
	}
}

func TestExecuteRESTUnsupported(t *testing.T) {
	runDir := t.TempDir()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "unknown",
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
	}
	if err := executeREST(context.Background(), "unknown", agentSelection{Name: "unknown"}, "prompt", runDir, nil, runDir, "", info); err == nil {
		t.Fatalf("expected unsupported rest agent error")
	}
}

func TestFinalizeRunFailure(t *testing.T) {
	runDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runDir, "agent-stdout.txt"), []byte("output"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
	}
	path := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	if err := finalizeRun(runDir, "", info, errors.New("boom")); err == nil {
		t.Fatalf("expected error for failed run")
	}
	updated, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if updated.Status != storage.StatusFailed || updated.ExitCode != 1 {
		t.Fatalf("unexpected status: %+v", updated)
	}
}

func TestFinalizeRunNilInfo(t *testing.T) {
	if err := finalizeRun(t.TempDir(), "", nil, nil); err == nil {
		t.Fatalf("expected error for nil run info")
	}
}
