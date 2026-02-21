package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/claude"
)

const (
	envClaudeArgs  = "CLAUDE_STUB_ARGS"
	envClaudeStdin = "CLAUDE_STUB_STDIN"
	envClaudeToken = "CLAUDE_STUB_TOKEN"
	envClaudeMode  = "CLAUDE_STUB_MODE"
)

func TestClaudeStdioCapture(t *testing.T) {
	stubDir := t.TempDir()
	stubPath := buildClaudeStub(t, stubDir)
	argsPath := filepath.Join(stubDir, "args.txt")
	stdinPath := filepath.Join(stubDir, "stdin.txt")
	tokenPath := filepath.Join(stubDir, "token.txt")
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))
	t.Setenv(envClaudeArgs, argsPath)
	t.Setenv(envClaudeStdin, stdinPath)
	t.Setenv(envClaudeToken, tokenPath)

	workingDir := t.TempDir()
	stdoutPath := filepath.Join(stubDir, "agent-stdout.txt")
	stderrPath := filepath.Join(stubDir, "agent-stderr.txt")
	prompt := "hello claude"

	runCtx := &agent.RunContext{
		Prompt:     prompt,
		WorkingDir: workingDir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Environment: map[string]string{
			"ANTHROPIC_API_KEY": "stub-token",
		},
	}

	agentImpl := &claude.ClaudeAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := agentImpl.Execute(ctx, runCtx); err != nil {
		stdoutBytes, _ := os.ReadFile(stdoutPath)
		stderrBytes, _ := os.ReadFile(stderrPath)
		t.Fatalf("Execute: %v\nstdout:\n%s\nstderr:\n%s", err, stdoutBytes, stderrBytes)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "stub stdout") {
		t.Fatalf("stdout missing stub output: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stderrBytes), "stub stderr") {
		t.Fatalf("stderr missing stub output: %q", string(stderrBytes))
	}

	stdinBytes, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	if string(stdinBytes) != prompt {
		t.Fatalf("stdin mismatch: %q", string(stdinBytes))
	}

	argsBytes, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	args := splitArgs(string(argsBytes))
	assertArg(t, args, "-p")
	assertArgPair(t, args, "--input-format", "text")
	assertArgPair(t, args, "--output-format", "stream-json")
	assertArg(t, args, "--tools")
	assertArg(t, args, "default")
	assertArg(t, args, "--verbose")
	assertArg(t, args, "--permission-mode")
	assertArg(t, args, "bypassPermissions")
	assertArgPair(t, args, "-C", workingDir)

	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	if string(tokenBytes) != "stub-token" {
		t.Fatalf("token mismatch: %q", string(tokenBytes))
	}
}

func TestClaudeWritesOutputMDFromStream(t *testing.T) {
	stubDir := t.TempDir()
	stubPath := buildClaudeStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))
	t.Setenv(envClaudeMode, "stream-json")

	workingDir := t.TempDir()
	runDir := t.TempDir()
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(runDir, "agent-stderr.txt")

	runCtx := &agent.RunContext{
		Prompt:     "hello claude",
		WorkingDir: workingDir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Environment: map[string]string{
			"ANTHROPIC_API_KEY": "stub-token",
		},
	}

	agentImpl := &claude.ClaudeAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := agentImpl.Execute(ctx, runCtx); err != nil {
		stdoutBytes, _ := os.ReadFile(stdoutPath)
		stderrBytes, _ := os.ReadFile(stderrPath)
		t.Fatalf("Execute: %v\nstdout:\n%s\nstderr:\n%s", err, stdoutBytes, stderrBytes)
	}

	outputBytes, err := os.ReadFile(filepath.Join(runDir, "output.md"))
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if string(outputBytes) != "final stream result" {
		t.Fatalf("unexpected output.md contents: %q", string(outputBytes))
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "\"type\":\"result\"") {
		t.Fatalf("stdout missing result event: %q", string(stdoutBytes))
	}
}

func TestClaudeExecutionFailurePropagatesError(t *testing.T) {
	stubDir := t.TempDir()
	stubPath := buildClaudeStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))
	t.Setenv(envClaudeMode, "fail")

	workingDir := t.TempDir()
	stdoutPath := filepath.Join(stubDir, "agent-stdout.txt")
	stderrPath := filepath.Join(stubDir, "agent-stderr.txt")

	runCtx := &agent.RunContext{
		Prompt:     "hello claude",
		WorkingDir: workingDir,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Environment: map[string]string{
			"ANTHROPIC_API_KEY": "stub-token",
		},
	}

	agentImpl := &claude.ClaudeAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := agentImpl.Execute(ctx, runCtx)
	if err == nil {
		t.Fatalf("expected Execute to fail")
	}
	if !strings.Contains(err.Error(), "claude execution failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	stderrBytes, readErr := os.ReadFile(stderrPath)
	if readErr != nil {
		t.Fatalf("read stderr: %v", readErr)
	}
	if !strings.Contains(string(stderrBytes), "forced failure from stub") {
		t.Fatalf("stderr missing failure details: %q", string(stderrBytes))
	}
}

func TestClaudeExecution(t *testing.T) {
	if os.Getenv("RUN_CLAUDE_TESTS") != "1" {
		t.Skip("set RUN_CLAUDE_TESTS=1 to run claude CLI integration test")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found")
	}
	if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	root := t.TempDir()
	runCtx := &agent.RunContext{
		Prompt:     "Respond with the word OK.",
		WorkingDir: root,
		StdoutPath: filepath.Join(root, "agent-stdout.txt"),
		StderrPath: filepath.Join(root, "agent-stderr.txt"),
	}

	agentImpl := &claude.ClaudeAgent{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := agentImpl.Execute(ctx, runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	stdoutBytes, err := os.ReadFile(runCtx.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if strings.TrimSpace(string(stdoutBytes)) == "" {
		t.Fatalf("expected stdout output")
	}
}

func buildClaudeStub(t *testing.T, dir string) string {
	t.Helper()

	stubPath := filepath.Join(dir, "claude")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}

	src := `package main

import (
	"io"
	"os"
	"strings"
)

func main() {
	if path := os.Getenv("` + envClaudeArgs + `"); path != "" {
		_ = os.WriteFile(path, []byte(strings.Join(os.Args[1:], "\n")), 0o644)
	}
	if path := os.Getenv("` + envClaudeStdin + `"); path != "" {
		data, _ := io.ReadAll(os.Stdin)
		_ = os.WriteFile(path, data, 0o644)
	} else {
		_, _ = io.Copy(io.Discard, os.Stdin)
	}
	if path := os.Getenv("` + envClaudeToken + `"); path != "" {
		_ = os.WriteFile(path, []byte(os.Getenv("ANTHROPIC_API_KEY")), 0o644)
	}

	switch os.Getenv("` + envClaudeMode + `") {
	case "stream-json":
		_, _ = os.Stdout.WriteString("{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"intermediate\"}]}}\n")
		_, _ = os.Stdout.WriteString("{\"type\":\"result\",\"result\":\"final stream result\",\"is_error\":false}\n")
		_, _ = os.Stderr.WriteString("stub stderr\n")
	case "fail":
		_, _ = os.Stderr.WriteString("forced failure from stub\n")
		os.Exit(23)
	default:
		_, _ = os.Stdout.WriteString("stub stdout\n")
		_, _ = os.Stderr.WriteString("stub stderr\n")
	}
}
`

	srcPath := filepath.Join(dir, "claude_stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", stubPath, srcPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build stub: %v\n%s", err, out)
	}

	return stubPath
}

func prependPath(dir string) string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return dir
	}
	return dir + string(os.PathListSeparator) + pathEnv
}

func splitArgs(value string) []string {
	lines := strings.Split(value, "\n")
	args := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		args = append(args, trimmed)
	}
	return args
}

func assertArg(t *testing.T, args []string, expected string) {
	t.Helper()
	for _, arg := range args {
		if arg == expected {
			return
		}
	}
	t.Fatalf("missing arg %q in %v", expected, args)
}

func assertArgPair(t *testing.T, args []string, flag string, value string) {
	t.Helper()
	for i, arg := range args {
		if arg == flag {
			if i+1 >= len(args) {
				t.Fatalf("flag %q missing value in %v", flag, args)
			}
			if args[i+1] != value {
				t.Fatalf("flag %q value mismatch: %q", flag, args[i+1])
			}
			return
		}
	}
	t.Fatalf("missing flag %q in %v", flag, args)
}
