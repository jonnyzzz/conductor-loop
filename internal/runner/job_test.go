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
	tests := []struct {
		name      string
		agentType string
		wantCmd   string
		wantErr   bool
	}{
		{name: "codex", agentType: "codex", wantCmd: "codex"},
		{name: "claude", agentType: "claude", wantCmd: "claude"},
		{name: "gemini", agentType: "gemini", wantCmd: "gemini"},
		{name: "unknown", agentType: "unknown", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, args, err := commandForAgent(tc.agentType)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for agent type %q", tc.agentType)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != tc.wantCmd {
				t.Fatalf("expected command %q, got %q", tc.wantCmd, cmd)
			}
			if len(args) == 0 {
				t.Fatalf("expected non-empty args")
			}
			// working directory is handled by SpawnOptions.Dir, not CLI flags
			for _, arg := range args {
				if arg == "-C" {
					t.Fatalf("args should not contain -C flag, got %v", args)
				}
			}
		})
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
		content := "@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo " + name + " 1.0.0\r\n  exit /b 0\r\n)\r\nmore >nul\r\necho stdout\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '" + name + " 1.0.0'; exit 0; fi\ncat >/dev/null\necho stdout\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func createFailingCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo " + name + " 1.0.0\r\n  exit /b 0\r\n)\r\nexit /b 1\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '" + name + " 1.0.0'; exit 0; fi\nexit 1\n"
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

func TestRemoveEnvKeys(t *testing.T) {
	env := []string{"A=1", "CLAUDECODE=1", "B=2", "PATH=/usr/bin"}
	filtered := removeEnvKeys(env, "CLAUDECODE")
	for _, entry := range filtered {
		if strings.HasPrefix(entry, "CLAUDECODE=") {
			t.Fatalf("CLAUDECODE should have been removed, got %q", entry)
		}
	}
	if len(filtered) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(filtered), filtered)
	}
}

func TestRemoveEnvKeysMultiple(t *testing.T) {
	env := []string{"A=1", "CLAUDECODE=1", "B=2", "SECRET=x"}
	filtered := removeEnvKeys(env, "CLAUDECODE", "SECRET")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(filtered), filtered)
	}
}

func TestRemoveEnvKeysEmpty(t *testing.T) {
	env := []string{"A=1", "B=2"}
	filtered := removeEnvKeys(env, "CLAUDECODE")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(filtered), filtered)
	}
}

func TestFinalizeRunNilInfo(t *testing.T) {
	if err := finalizeRun(t.TempDir(), "", nil, nil); err == nil {
		t.Fatalf("expected error for nil run info")
	}
}

func TestTailFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		result := tailFile(filepath.Join(t.TempDir(), "nope.txt"), 50)
		if result != "" {
			t.Fatalf("expected empty string for missing file, got %q", result)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.txt")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		result := tailFile(path, 50)
		if result != "" {
			t.Fatalf("expected empty string for empty file, got %q", result)
		}
	})

	t.Run("short file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "short.txt")
		if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		result := tailFile(path, 50)
		if result != "line1\nline2\nline3" {
			t.Fatalf("unexpected result: %q", result)
		}
	})

	t.Run("long file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "long.txt")
		var lines []string
		for i := 1; i <= 100; i++ {
			lines = append(lines, fmt.Sprintf("line %d", i))
		}
		content := strings.Join(lines, "\n") + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		result := tailFile(path, 5)
		expected := "line 96\nline 97\nline 98\nline 99\nline 100"
		if result != expected {
			t.Fatalf("expected %q, got %q", expected, result)
		}
	})

	t.Run("zero maxLines", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(path, []byte("data\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		result := tailFile(path, 0)
		if result != "" {
			t.Fatalf("expected empty for zero maxLines, got %q", result)
		}
	})
}

func TestErrorSummaryClassification(t *testing.T) {
	tests := []struct {
		exitCode int
		expected string
	}{
		{1, "agent reported failure"},
		{2, "agent usage error"},
		{137, "agent killed (OOM or signal)"},
		{143, "agent terminated (SIGTERM)"},
		{42, "agent exited with code 42"},
		{255, "agent exited with code 255"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("exit_%d", tc.exitCode), func(t *testing.T) {
			got := classifyExitCode(tc.exitCode)
			if got != tc.expected {
				t.Fatalf("classifyExitCode(%d) = %q, want %q", tc.exitCode, got, tc.expected)
			}
		})
	}
}

// TestRunJobCLIEmitsRunStop verifies that a job exiting with code 0 emits RUN_STOP (not RUN_CRASH).
func TestRunJobCLIEmitsRunStop(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	busPath := filepath.Join(root, "TASK-MESSAGE-BUS.md")
	_, err := runJob("project", "task", JobOptions{
		RootDir:        root,
		Agent:          "codex",
		Prompt:         "hello",
		MessageBusPath: busPath,
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	msgs := readBusMessages(t, busPath)
	assertBusEvent(t, msgs, messagebus.EventTypeRunStart, true)
	assertBusEvent(t, msgs, messagebus.EventTypeRunStop, true)
	assertBusEvent(t, msgs, messagebus.EventTypeRunCrash, false)
}

// TestRunJobCLIEmitsRunCrash verifies that a job exiting with non-zero code emits RUN_CRASH (not RUN_STOP).
func TestRunJobCLIEmitsRunCrash(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFailingCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	busPath := filepath.Join(root, "TASK-MESSAGE-BUS.md")
	_, err := runJob("project", "task", JobOptions{
		RootDir:        root,
		Agent:          "codex",
		Prompt:         "hello",
		MessageBusPath: busPath,
	})
	// Non-zero exit is expected to return an error.
	if err == nil {
		t.Fatalf("expected error from failing agent")
	}

	msgs := readBusMessages(t, busPath)
	assertBusEvent(t, msgs, messagebus.EventTypeRunStart, true)
	assertBusEvent(t, msgs, messagebus.EventTypeRunCrash, true)
	assertBusEvent(t, msgs, messagebus.EventTypeRunStop, false)
}

// readBusMessages reads all messages from a message bus file for testing.
func readBusMessages(t *testing.T, busPath string) []*messagebus.Message {
	t.Helper()
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	return msgs
}

// assertBusEvent checks whether a message type is present (wantPresent=true) or absent (wantPresent=false).
func assertBusEvent(t *testing.T, msgs []*messagebus.Message, msgType string, wantPresent bool) {
	t.Helper()
	for _, m := range msgs {
		if m.Type == msgType {
			if !wantPresent {
				t.Fatalf("unexpected event %q found in message bus", msgType)
			}
			return
		}
	}
	if wantPresent {
		t.Fatalf("expected event %q not found in message bus", msgType)
	}
}
