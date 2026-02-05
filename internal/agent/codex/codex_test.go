package codex

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

func TestOpenPromptInline(t *testing.T) {
	reader, closer, err := openPrompt("hello")
	if err != nil {
		t.Fatalf("openPrompt: %v", err)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "hello" {
		t.Fatalf("unexpected prompt: %q", string(data))
	}
	if err := closer(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestOpenPromptEmpty(t *testing.T) {
	if _, _, err := openPrompt(" "); err == nil {
		t.Fatalf("expected error for empty prompt")
	}
}

func TestOpenPromptFileAndDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(path, []byte("file prompt"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	reader, closer, err := openPrompt(path)
	if err != nil {
		t.Fatalf("openPrompt: %v", err)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "file prompt" {
		t.Fatalf("unexpected prompt: %q", string(data))
	}
	if err := closer(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, _, err := openPrompt(filepath.Dir(path)); err == nil {
		t.Fatalf("expected error for directory prompt")
	}
}

func TestCodexArgs(t *testing.T) {
	args := codexArgs("")
	if len(args) == 0 || args[0] != "exec" {
		t.Fatalf("unexpected args: %v", args)
	}
	args = codexArgs("/tmp")
	if len(args) < 4 || args[2] != "-C" {
		t.Fatalf("expected -C in args, got %v", args)
	}
}

func TestType(t *testing.T) {
	if (&CodexAgent{}).Type() != "codex" {
		t.Fatalf("unexpected type")
	}
}

func TestBuildEnvironmentAddsToken(t *testing.T) {
	env := buildEnvironment(map[string]string{"FOO": "1"}, "token")
	foundToken := false
	for _, entry := range env {
		if strings.HasPrefix(entry, tokenEnvVar+"=") {
			foundToken = true
			break
		}
	}
	if !foundToken {
		t.Fatalf("expected token env var")
	}
}

func TestWaitForProcessCanceled(t *testing.T) {
	cmd, args := shellCommand("sleep 1")
	process, err := agent.SpawnProcess(cmd, args, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("SpawnProcess: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = waitForProcess(ctx, process)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
}

func TestExecuteWithFakeCommand(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdoutPath := filepath.Join(dir, "stdout.txt")
	stderrPath := filepath.Join(dir, "stderr.txt")

	agentImpl := &CodexAgent{token: "token"}
	runCtx := &agent.RunContext{
		Prompt:     "hello",
		WorkingDir: dir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
	}
	if err := agentImpl.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	stdout, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(stdout), "stdout") {
		t.Fatalf("expected stdout content, got %q", string(stdout))
	}
}

func TestExecuteValidationErrors(t *testing.T) {
	agentImpl := &CodexAgent{}
	if err := agentImpl.Execute(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil run context")
	}
	ctx := &agent.RunContext{Prompt: "hi"}
	if err := agentImpl.Execute(context.Background(), ctx); err == nil {
		t.Fatalf("expected error for empty working dir")
	}
	ctx = &agent.RunContext{Prompt: " ", WorkingDir: t.TempDir(), StdoutPath: filepath.Join(t.TempDir(), "out"), StderrPath: filepath.Join(t.TempDir(), "err")}
	if err := agentImpl.Execute(context.Background(), ctx); err == nil {
		t.Fatalf("expected error for empty prompt")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	ctx = &agent.RunContext{Prompt: "hi", WorkingDir: t.TempDir(), StdoutPath: filepath.Join(t.TempDir(), "out2"), StderrPath: filepath.Join(t.TempDir(), "err2")}
	if err := agentImpl.Execute(cancelled, ctx); err == nil {
		t.Fatalf("expected error for canceled context")
	}
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-c", command}
}

func createFakeCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nmore >nul\r\necho stdout\r\necho stderr 1>&2\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\ncat >/dev/null\necho stdout\necho stderr 1>&2\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}
