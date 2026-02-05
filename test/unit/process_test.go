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
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
)

func TestRunnerSpawnProcess(t *testing.T) {
	runDir := t.TempDir()
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}

	command, args := outputCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := pm.SpawnAgent(ctx, "test", runner.SpawnOptions{
		Command: command,
		Args:    args,
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if proc.PID <= 0 {
		t.Fatalf("expected pid to be set, got %d", proc.PID)
	}
	if proc.PGID <= 0 {
		t.Fatalf("expected pgid to be set, got %d", proc.PGID)
	}
	assertRunnerProcessGroupConfigured(t, proc.Cmd)

	if err := proc.Wait(); err != nil {
		t.Fatalf("wait process: %v", err)
	}

	stdoutBytes, err := os.ReadFile(proc.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	stderrBytes, err := os.ReadFile(proc.StderrPath)
	if err != nil {
		t.Fatalf("read stderr file: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "stdout") {
		t.Fatalf("stdout file missing content: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stderrBytes), "stderr") {
		t.Fatalf("stderr file missing content: %q", string(stderrBytes))
	}
	if !strings.Contains(stdout.String(), "stdout") {
		t.Fatalf("stdout buffer missing content: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "stderr") {
		t.Fatalf("stderr buffer missing content: %q", stderr.String())
	}
}

func TestProcessSetsid(t *testing.T) {
	runDir := t.TempDir()
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}

	command, args := outputCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := pm.SpawnAgent(ctx, "test", runner.SpawnOptions{
		Command: command,
		Args:    args,
	})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	assertRunnerProcessGroupConfigured(t, proc.Cmd)
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait process: %v", err)
	}
}

func TestStdioRedirection(t *testing.T) {
	runDir := t.TempDir()
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}

	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(runDir, "agent-stderr.txt")
	if err := os.WriteFile(stdoutPath, []byte("existing stdout\n"), 0o644); err != nil {
		t.Fatalf("write stdout prefix: %v", err)
	}
	if err := os.WriteFile(stderrPath, []byte("existing stderr\n"), 0o644); err != nil {
		t.Fatalf("write stderr prefix: %v", err)
	}

	command, args := outputCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := pm.SpawnAgent(ctx, "test", runner.SpawnOptions{
		Command: command,
		Args:    args,
	})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait process: %v", err)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	stderrBytes, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("read stderr file: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "existing stdout") {
		t.Fatalf("stdout prefix missing: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stdoutBytes), "stdout") {
		t.Fatalf("stdout content missing: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stderrBytes), "existing stderr") {
		t.Fatalf("stderr prefix missing: %q", string(stderrBytes))
	}
	if !strings.Contains(string(stderrBytes), "stderr") {
		t.Fatalf("stderr content missing: %q", string(stderrBytes))
	}
}

func TestProcessGroupManagement(t *testing.T) {
	runDir := t.TempDir()
	pm, err := runner.NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}

	command, args := outputCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := pm.SpawnAgent(ctx, "test", runner.SpawnOptions{
		Command: command,
		Args:    args,
	})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}

	pgid, err := runner.ProcessGroupID(proc.PID)
	if err != nil {
		_ = proc.Wait()
		t.Fatalf("ProcessGroupID: %v", err)
	}
	if pgid != proc.PGID {
		_ = proc.Wait()
		t.Fatalf("pgid mismatch: want %d, got %d", proc.PGID, pgid)
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait process: %v", err)
	}
}

func outputCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo stdout & echo stderr 1>&2 & ping -n 2 127.0.0.1 >nul"}
	}
	return "sh", []string{"-c", "echo stdout; echo stderr 1>&2; sleep 0.2"}
}

func assertRunnerProcessGroupConfigured(t *testing.T, cmd interface{}) {
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
	t.Fatalf("process group configuration not found")
}
