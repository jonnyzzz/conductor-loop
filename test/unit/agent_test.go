package unit_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

type testAgent struct {
	called bool
}

func (a *testAgent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
	_ = ctx
	_ = runCtx
	a.called = true
	return nil
}

func (a *testAgent) Type() string {
	return "test"
}

func TestAgentInterface(t *testing.T) {
	var _ agent.Agent = (*testAgent)(nil)
	agentImpl := &testAgent{}
	if err := agentImpl.Execute(context.Background(), &agent.RunContext{}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !agentImpl.called {
		t.Fatalf("expected Execute to set called flag")
	}
	if agentImpl.Type() != "test" {
		t.Fatalf("expected type %q, got %q", "test", agentImpl.Type())
	}
}

func TestRunContext(t *testing.T) {
	ctx := &agent.RunContext{
		RunID:      "run-1",
		ProjectID:  "project",
		TaskID:     "task",
		Prompt:     "prompt",
		WorkingDir: "/tmp",
		StdoutPath: "/tmp/stdout",
		StderrPath: "/tmp/stderr",
		Environment: map[string]string{
			"A": "1",
			"B": "2",
		},
	}

	if ctx.RunID != "run-1" {
		t.Fatalf("RunID mismatch: %q", ctx.RunID)
	}
	if ctx.ProjectID != "project" {
		t.Fatalf("ProjectID mismatch: %q", ctx.ProjectID)
	}
	if ctx.TaskID != "task" {
		t.Fatalf("TaskID mismatch: %q", ctx.TaskID)
	}
	if ctx.Prompt != "prompt" {
		t.Fatalf("Prompt mismatch: %q", ctx.Prompt)
	}
	if ctx.WorkingDir != "/tmp" {
		t.Fatalf("WorkingDir mismatch: %q", ctx.WorkingDir)
	}
	if ctx.StdoutPath != "/tmp/stdout" {
		t.Fatalf("StdoutPath mismatch: %q", ctx.StdoutPath)
	}
	if ctx.StderrPath != "/tmp/stderr" {
		t.Fatalf("StderrPath mismatch: %q", ctx.StderrPath)
	}
	if got := ctx.Environment["A"]; got != "1" {
		t.Fatalf("Environment mismatch: %q", got)
	}
}

func TestSpawnProcess(t *testing.T) {
	command, args := shellCommand("echo stdout && echo stderr 1>&2")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd, err := agent.SpawnProcess(command, args, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("SpawnProcess: %v", err)
	}
	if cmd.Process == nil {
		t.Fatalf("expected process to be started")
	}
	if cmd.Process.Pid <= 0 {
		t.Fatalf("expected pid to be set, got %d", cmd.Process.Pid)
	}
	assertProcessGroupConfigured(t, cmd)
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait process: %v", err)
	}

	if !strings.Contains(stdout.String(), "stdout") {
		t.Fatalf("stdout missing, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "stderr") {
		t.Fatalf("stderr missing, got %q", stderr.String())
	}
}

func TestCaptureOutput(t *testing.T) {
	root := t.TempDir()
	stdoutPath := filepath.Join(root, "logs", "agent-stdout.txt")
	stderrPath := filepath.Join(root, "logs", "agent-stderr.txt")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	capture, err := agent.CaptureOutput(&stdout, &stderr, agent.OutputFiles{
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
	})
	if err != nil {
		t.Fatalf("CaptureOutput: %v", err)
	}

	if capture.Stdout == nil {
		t.Fatalf("expected stdout writer")
	}
	if capture.Stderr == nil {
		t.Fatalf("expected stderr writer")
	}

	if _, err := capture.Stdout.Write([]byte("hello stdout\n")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if _, err := capture.Stderr.Write([]byte("hello stderr\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	if err := capture.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	stderrBytes, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("read stderr file: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "hello stdout") {
		t.Fatalf("stdout file missing content: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stderrBytes), "hello stderr") {
		t.Fatalf("stderr file missing content: %q", string(stderrBytes))
	}
	if !strings.Contains(stdout.String(), "hello stdout") {
		t.Fatalf("stdout buffer missing content: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "hello stderr") {
		t.Fatalf("stderr buffer missing content: %q", stderr.String())
	}
}

func TestCreateOutputMD(t *testing.T) {
	root := t.TempDir()
	fallbackPath := filepath.Join(root, "agent-stdout.txt")
	if err := os.WriteFile(fallbackPath, []byte("output content"), 0o644); err != nil {
		t.Fatalf("write fallback: %v", err)
	}

	outputPath, err := agent.CreateOutputMD(root, "")
	if err != nil {
		t.Fatalf("CreateOutputMD: %v", err)
	}
	if outputPath != filepath.Join(root, "output.md") {
		t.Fatalf("unexpected output path: %q", outputPath)
	}

	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if string(outputBytes) != "output content" {
		t.Fatalf("output.md content mismatch: %q", string(outputBytes))
	}
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-c", command}
}

func assertProcessGroupConfigured(t *testing.T, cmd interface{}) {
	t.Helper()

	v := reflect.ValueOf(cmd)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		t.Fatalf("expected command pointer")
	}
	field := v.Elem().FieldByName("SysProcAttr")
	if !field.IsValid() || field.IsNil() {
		t.Fatalf("SysProcAttr not set")
	}
	attr := field.Elem()
	if attr.Kind() != reflect.Struct {
		t.Fatalf("SysProcAttr not a struct")
	}
	if setsid := attr.FieldByName("Setsid"); setsid.IsValid() && setsid.Kind() == reflect.Bool {
		if !setsid.Bool() {
			t.Fatalf("expected Setsid true")
		}
		return
	}
	if flags := attr.FieldByName("CreationFlags"); flags.IsValid() {
		if flags.Kind() == reflect.Uint32 || flags.Kind() == reflect.Uint64 || flags.Kind() == reflect.Uint {
			if flags.Uint() == 0 {
				t.Fatalf("expected CreationFlags to be set")
			}
			return
		}
	}
	// Fallback: not a known platform-specific attribute.
	t.Fatalf("process group configuration not found")
}
