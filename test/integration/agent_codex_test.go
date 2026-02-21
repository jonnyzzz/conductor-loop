package integration_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/codex"
)

const (
	envCodexHelperMode       = "CODEX_HELPER_MODE"
	envCodexExpectedPrompt   = "CODEX_EXPECT_PROMPT"
	envCodexExpectedCwd      = "CODEX_EXPECT_CWD"
	envCodexExpectedEnvKey   = "CODEX_EXPECT_ENV_KEY"
	envCodexExpectedEnvValue = "CODEX_EXPECT_ENV_VALUE"
	envCodexHelperExitCode   = "CODEX_HELPER_EXIT_CODE"
)

func TestCodexExecution(t *testing.T) {
	root := t.TempDir()
	workingDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	resolvedWorkingDir := workingDir
	if resolved, err := filepath.EvalSymlinks(workingDir); err == nil {
		resolvedWorkingDir = resolved
	}
	prompt := "prompt from test"

	stdoutPath := filepath.Join(root, "agent-stdout.txt")
	stderrPath := filepath.Join(root, "agent-stderr.txt")

	codexPath := filepath.Join(root, codexBinaryName())
	if err := copyFile(os.Args[0], codexPath, 0o755); err != nil {
		t.Fatalf("copy codex helper: %v", err)
	}
	pathEnv := root + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", pathEnv)

	envKey := "OPENAI_API_KEY"
	envValue := "runctx-openai-token"

	t.Setenv(envCodexHelperMode, "1")
	t.Setenv(envCodexExpectedPrompt, prompt)
	t.Setenv(envCodexExpectedCwd, resolvedWorkingDir)
	t.Setenv(envCodexExpectedEnvKey, envKey)
	t.Setenv(envCodexExpectedEnvValue, envValue)

	agentImpl := &codex.CodexAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runCtx := &agent.RunContext{
		Prompt:     prompt,
		WorkingDir: resolvedWorkingDir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Environment: map[string]string{
			envKey: envValue,
		},
	}

	if err := agentImpl.Execute(ctx, runCtx); err != nil {
		stdoutBytes, _ := os.ReadFile(stdoutPath)
		stderrBytes, _ := os.ReadFile(stderrPath)
		t.Fatalf("Execute: %v\nstdout:\n%s\nstderr:\n%s", err, stdoutBytes, stderrBytes)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	stdoutText := string(stdoutBytes)
	if !strings.Contains(stdoutText, prompt) {
		t.Fatalf("stdout missing prompt, got %q", stdoutText)
	}

	stderrBytes, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("read stderr file: %v", err)
	}
	stderrText := string(stderrBytes)
	if !strings.Contains(stderrText, resolvedWorkingDir) {
		t.Fatalf("stderr missing working dir, got %q", stderrText)
	}
	if !strings.Contains(stderrText, envKey+"="+envValue) {
		t.Fatalf("stderr missing env value, got %q", stderrText)
	}
}

func TestCodexExecutionPromptFromFile(t *testing.T) {
	root := t.TempDir()
	workingDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	resolvedWorkingDir := workingDir
	if resolved, err := filepath.EvalSymlinks(workingDir); err == nil {
		resolvedWorkingDir = resolved
	}

	promptText := "prompt loaded from file"
	promptPath := filepath.Join(root, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte(promptText), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	stdoutPath := filepath.Join(root, "agent-stdout.txt")
	stderrPath := filepath.Join(root, "agent-stderr.txt")

	codexPath := filepath.Join(root, codexBinaryName())
	if err := copyFile(os.Args[0], codexPath, 0o755); err != nil {
		t.Fatalf("copy codex helper: %v", err)
	}
	pathEnv := root + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", pathEnv)

	envKey := "OPENAI_API_KEY"
	envValue := "token-from-prompt-file-test"

	t.Setenv(envCodexHelperMode, "1")
	t.Setenv(envCodexExpectedPrompt, promptText)
	t.Setenv(envCodexExpectedCwd, resolvedWorkingDir)
	t.Setenv(envCodexExpectedEnvKey, envKey)
	t.Setenv(envCodexExpectedEnvValue, envValue)

	agentImpl := &codex.CodexAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runCtx := &agent.RunContext{
		Prompt:     promptPath,
		WorkingDir: resolvedWorkingDir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Environment: map[string]string{
			envKey: envValue,
		},
	}

	if err := agentImpl.Execute(ctx, runCtx); err != nil {
		stdoutBytes, _ := os.ReadFile(stdoutPath)
		stderrBytes, _ := os.ReadFile(stderrPath)
		t.Fatalf("Execute: %v\nstdout:\n%s\nstderr:\n%s", err, stdoutBytes, stderrBytes)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), promptText) {
		t.Fatalf("stdout missing file prompt, got %q", string(stdoutBytes))
	}
}

func TestCodexExecutionFailurePropagatesError(t *testing.T) {
	root := t.TempDir()
	workingDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	resolvedWorkingDir := workingDir
	if resolved, err := filepath.EvalSymlinks(workingDir); err == nil {
		resolvedWorkingDir = resolved
	}

	prompt := "prompt from failure test"
	stdoutPath := filepath.Join(root, "agent-stdout.txt")
	stderrPath := filepath.Join(root, "agent-stderr.txt")

	codexPath := filepath.Join(root, codexBinaryName())
	if err := copyFile(os.Args[0], codexPath, 0o755); err != nil {
		t.Fatalf("copy codex helper: %v", err)
	}
	pathEnv := root + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", pathEnv)

	envKey := "OPENAI_API_KEY"
	envValue := "token-for-failure-test"

	t.Setenv(envCodexHelperMode, "1")
	t.Setenv(envCodexExpectedPrompt, prompt)
	t.Setenv(envCodexExpectedCwd, resolvedWorkingDir)
	t.Setenv(envCodexExpectedEnvKey, envKey)
	t.Setenv(envCodexExpectedEnvValue, envValue)
	t.Setenv(envCodexHelperExitCode, "17")

	agentImpl := &codex.CodexAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runCtx := &agent.RunContext{
		Prompt:     prompt,
		WorkingDir: resolvedWorkingDir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Environment: map[string]string{
			envKey: envValue,
		},
	}

	err := agentImpl.Execute(ctx, runCtx)
	if err == nil {
		t.Fatalf("expected Execute to fail")
	}
	if !strings.Contains(err.Error(), "codex execution failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	stderrBytes, readErr := os.ReadFile(stderrPath)
	if readErr != nil {
		t.Fatalf("read stderr file: %v", readErr)
	}
	if !strings.Contains(string(stderrBytes), "forced codex failure") {
		t.Fatalf("stderr missing forced failure marker, got %q", string(stderrBytes))
	}
}

func runCodexHelper() error {
	expectedPrompt := os.Getenv(envCodexExpectedPrompt)
	expectedCwd := os.Getenv(envCodexExpectedCwd)
	expectedEnvKey := os.Getenv(envCodexExpectedEnvKey)
	expectedEnvValue := os.Getenv(envCodexExpectedEnvValue)
	if expectedPrompt == "" {
		return fmt.Errorf("expected prompt env %s is empty", envCodexExpectedPrompt)
	}
	if expectedCwd == "" {
		return fmt.Errorf("expected cwd env %s is empty", envCodexExpectedCwd)
	}
	if expectedEnvKey == "" {
		return fmt.Errorf("expected env key %s is empty", envCodexExpectedEnvKey)
	}
	if expectedEnvValue == "" {
		return fmt.Errorf("expected env value %s is empty", envCodexExpectedEnvValue)
	}

	args := os.Args[1:]
	expectedArgs := []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"-C",
		expectedCwd,
		"-",
	}
	if len(args) != len(expectedArgs) {
		return fmt.Errorf("unexpected args: got %v want %v", args, expectedArgs)
	}
	for i, arg := range expectedArgs {
		if args[i] != arg {
			return fmt.Errorf("unexpected arg %d: got %q want %q", i, args[i], arg)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	if cwd != expectedCwd {
		return fmt.Errorf("cwd mismatch: got %q want %q", cwd, expectedCwd)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if string(data) != expectedPrompt {
		return fmt.Errorf("prompt mismatch: got %q want %q", string(data), expectedPrompt)
	}

	if got := os.Getenv(expectedEnvKey); got != expectedEnvValue {
		return fmt.Errorf("env mismatch: %s=%q want %q", expectedEnvKey, got, expectedEnvValue)
	}

	if exitCodeText := strings.TrimSpace(os.Getenv(envCodexHelperExitCode)); exitCodeText != "" {
		code, err := strconv.Atoi(exitCodeText)
		if err != nil {
			return fmt.Errorf("invalid %s value %q: %w", envCodexHelperExitCode, exitCodeText, err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "forced codex failure\n")
		os.Exit(code)
	}

	_, _ = fmt.Fprintf(os.Stdout, "prompt:%s\n", string(data))
	_, _ = fmt.Fprintf(os.Stderr, "cwd:%s\n", cwd)
	_, _ = fmt.Fprintf(os.Stderr, "env:%s=%s\n", expectedEnvKey, expectedEnvValue)
	return nil
}

func codexBinaryName() string {
	if runtime.GOOS == "windows" {
		return "codex.exe"
	}
	return "codex"
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
