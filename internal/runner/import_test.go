package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRunImportedProcessMirrorsLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based process fixture is unix-only")
	}

	root := t.TempDir()
	stdoutSource := filepath.Join(root, "external-stdout.log")
	stderrSource := filepath.Join(root, "external-stderr.log")
	scriptPath := filepath.Join(root, "external.sh")
	script := `#!/bin/sh
echo "stdout-1" >> "$SRC_STDOUT"
echo "stderr-1" >> "$SRC_STDERR"
sleep 0.2
echo "stdout-2" >> "$SRC_STDOUT"
echo "stderr-2" >> "$SRC_STDERR"
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

	err := RunImportedProcess("project", "task-import", ImportOptions{
		RootDir:    root,
		WorkingDir: root,
		Process: ImportedProcess{
			PID:         cmd.Process.Pid,
			AgentType:   "codex",
			CommandLine: scriptPath,
			StdoutPath:  stdoutSource,
			StderrPath:  stderrSource,
		},
	})
	if err != nil {
		t.Fatalf("RunImportedProcess: %v", err)
	}
	if waitErr := <-waitDone; waitErr != nil {
		t.Fatalf("fixture process wait: %v", waitErr)
	}

	runDir := singleRunDir(t, root, "project", "task-import")
	info, err := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if info.Status != storage.StatusCompleted {
		t.Fatalf("status=%q, want %q", info.Status, storage.StatusCompleted)
	}
	if storage.EffectiveProcessOwnership(info) != storage.ProcessOwnershipExternal {
		t.Fatalf("ownership=%q, want %q", info.ProcessOwnership, storage.ProcessOwnershipExternal)
	}
	if info.ExitCode != -1 {
		t.Fatalf("exit_code=%d, want -1 for adopted process", info.ExitCode)
	}
	if !strings.Contains(info.ErrorSummary, "exit code unavailable") {
		t.Fatalf("unexpected error_summary: %q", info.ErrorSummary)
	}

	stdoutData, err := os.ReadFile(filepath.Join(runDir, "agent-stdout.txt"))
	if err != nil {
		t.Fatalf("read mirrored stdout: %v", err)
	}
	if !strings.Contains(string(stdoutData), "stdout-1") || !strings.Contains(string(stdoutData), "stdout-2") {
		t.Fatalf("mirrored stdout missing expected lines: %q", string(stdoutData))
	}

	stderrData, err := os.ReadFile(filepath.Join(runDir, "agent-stderr.txt"))
	if err != nil {
		t.Fatalf("read mirrored stderr: %v", err)
	}
	if !strings.Contains(string(stderrData), "stderr-1") || !strings.Contains(string(stderrData), "stderr-2") {
		t.Fatalf("mirrored stderr missing expected lines: %q", string(stderrData))
	}

	if _, err := os.Stat(filepath.Join(runDir, "output.md")); err != nil {
		t.Fatalf("output.md missing: %v", err)
	}

	busPath := filepath.Join(root, "project", "task-import", "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("open message bus: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Type != messagebus.EventTypeRunStart {
		t.Fatalf("msg[0].type=%q, want %q", msgs[0].Type, messagebus.EventTypeRunStart)
	}
	if msgs[1].Type != messagebus.EventTypeRunStop {
		t.Fatalf("msg[1].type=%q, want %q", msgs[1].Type, messagebus.EventTypeRunStop)
	}
}

func TestRunImportedProcessPropagatesParentRunIDToRunInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based process fixture is unix-only")
	}

	root := t.TempDir()
	stdoutSource := filepath.Join(root, "external-parent-stdout.log")
	scriptPath := filepath.Join(root, "external-parent.sh")
	script := `#!/bin/sh
sleep 0.2
echo "stdout-parent" >> "$SRC_STDOUT"
sleep 0.2
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(scriptPath)
	cmd.Env = append(os.Environ(),
		"SRC_STDOUT="+stdoutSource,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fixture process: %v", err)
	}
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	const parentRunID = "run-parent-import"
	err := RunImportedProcess("project", "task-import-parent", ImportOptions{
		RootDir:     root,
		WorkingDir:  root,
		ParentRunID: parentRunID,
		Process: ImportedProcess{
			PID:         cmd.Process.Pid,
			AgentType:   "codex",
			CommandLine: scriptPath,
			StdoutPath:  stdoutSource,
		},
	})
	if err != nil {
		t.Fatalf("RunImportedProcess: %v", err)
	}
	if waitErr := <-waitDone; waitErr != nil {
		t.Fatalf("fixture process wait: %v", waitErr)
	}

	runDir := singleRunDir(t, root, "project", "task-import-parent")
	info, err := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if got := strings.TrimSpace(info.ParentRunID); got != parentRunID {
		t.Fatalf("parent_run_id=%q, want %q", got, parentRunID)
	}
}

func TestNormalizeImportedProcessDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdout.log")
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	normalized, err := normalizeImportedProcess(ImportedProcess{
		PID:        os.Getpid(),
		AgentType:  "Codex",
		StdoutPath: path,
	})
	if err != nil {
		t.Fatalf("normalizeImportedProcess: %v", err)
	}
	if normalized.Ownership != storage.ProcessOwnershipExternal {
		t.Fatalf("ownership=%q, want %q", normalized.Ownership, storage.ProcessOwnershipExternal)
	}
	if normalized.AgentType != "codex" {
		t.Fatalf("agent_type=%q, want codex", normalized.AgentType)
	}
	if !filepath.IsAbs(normalized.StdoutPath) {
		t.Fatalf("stdout path should be absolute: %q", normalized.StdoutPath)
	}
}
