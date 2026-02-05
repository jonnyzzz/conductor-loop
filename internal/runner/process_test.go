package runner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewProcessManagerValidation(t *testing.T) {
	if _, err := NewProcessManager(""); err == nil {
		t.Fatalf("expected error for empty run dir")
	}
}

func TestSpawnAgentValidation(t *testing.T) {
	runDir := t.TempDir()
	pm, err := NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}
	if _, err := pm.SpawnAgent(context.Background(), "", SpawnOptions{Command: "echo"}); err == nil {
		t.Fatalf("expected error for empty agent type")
	}
	if _, err := pm.SpawnAgent(context.Background(), "test", SpawnOptions{}); err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestSpawnAgent(t *testing.T) {
	runDir := t.TempDir()
	pm, err := NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}
	cmd, args := outputCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proc, err := pm.SpawnAgent(ctx, "test", SpawnOptions{Command: cmd, Args: args, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	stdoutBytes, err := os.ReadFile(proc.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := os.ReadFile(proc.StderrPath)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "stdout") {
		t.Fatalf("stdout missing: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stderrBytes), "stderr") {
		t.Fatalf("stderr missing: %q", string(stderrBytes))
	}
	if !strings.Contains(stdout.String(), "stdout") {
		t.Fatalf("stdout buffer missing")
	}
	if !strings.Contains(stderr.String(), "stderr") {
		t.Fatalf("stderr buffer missing")
	}
}

func TestProcessWaitNil(t *testing.T) {
	var proc *Process
	if err := proc.Wait(); err == nil {
		t.Fatalf("expected error for nil process")
	}
}

func TestStdioRedirection(t *testing.T) {
	runDir := t.TempDir()
	pm, err := NewProcessManager(runDir)
	if err != nil {
		t.Fatalf("NewProcessManager: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(runDir, "agent-stderr.txt")
	if err := os.WriteFile(stdoutPath, []byte("existing stdout\n"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := os.WriteFile(stderrPath, []byte("existing stderr\n"), 0o644); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	cmd, args := outputCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proc, err := pm.SpawnAgent(ctx, "test", SpawnOptions{Command: cmd, Args: args})
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(stdoutBytes), "existing stdout") || !strings.Contains(string(stdoutBytes), "stdout") {
		t.Fatalf("stdout content missing: %q", string(stdoutBytes))
	}
	if !strings.Contains(string(stderrBytes), "existing stderr") || !strings.Contains(string(stderrBytes), "stderr") {
		t.Fatalf("stderr content missing: %q", string(stderrBytes))
	}
}

func TestTerminateProcessGroupValidation(t *testing.T) {
	if err := TerminateProcessGroup(0); err == nil {
		t.Fatalf("expected error for invalid pgid")
	}
	if runtime.GOOS != "windows" {
		if err := TerminateProcessGroup(999999); err == nil {
			t.Fatalf("expected error for missing process group")
		}
	}
}

func TestTerminateProcessGroupSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group handling differs on windows")
	}
	runDir := t.TempDir()
	proc, cancel := spawnSleepProcess(t, runDir, 2*time.Second)
	defer cancel()
	if err := TerminateProcessGroup(proc.PGID); err != nil {
		t.Fatalf("TerminateProcessGroup: %v", err)
	}
	_ = proc.Wait()
}

func outputCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo stdout & echo stderr 1>&2 & ping -n 2 127.0.0.1 >nul"}
	}
	return "sh", []string{"-c", "echo stdout; echo stderr 1>&2; sleep 0.2"}
}
