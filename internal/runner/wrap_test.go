package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRunWrapSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrap integration helper script is unix-only in this test")
	}

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createWrapCLI(t, binDir, "codex", 0)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	argsPath := filepath.Join(root, "args.txt")
	envPath := filepath.Join(root, "env.txt")

	projectID := "project"
	taskID := "task-20260221-120000-wrap-success"
	err := RunWrap(projectID, taskID, "codex", []string{"--foo", "bar baz"}, WrapOptions{
		RootDir:    root,
		WorkingDir: root,
		Environment: map[string]string{
			"WRAP_ARGS_FILE": argsPath,
			"WRAP_ENV_FILE":  envPath,
		},
	})
	if err != nil {
		t.Fatalf("RunWrap: %v", err)
	}

	runDir := singleRunDir(t, root, projectID, taskID)
	info, err := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if info.Status != storage.StatusCompleted {
		t.Fatalf("status=%q, want %q", info.Status, storage.StatusCompleted)
	}
	if info.ExitCode != 0 {
		t.Fatalf("exit_code=%d, want 0", info.ExitCode)
	}
	if info.CommandLine == "" || !strings.Contains(info.CommandLine, "bar baz") {
		t.Fatalf("unexpected commandline: %q", info.CommandLine)
	}
	if info.PID <= 0 {
		t.Fatalf("pid should be > 0, got %d", info.PID)
	}

	taskMD, err := os.ReadFile(filepath.Join(root, projectID, taskID, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if !strings.Contains(string(taskMD), "Wrapped CLI invocation.") {
		t.Fatalf("TASK.md missing wrap marker:\n%s", string(taskMD))
	}

	if _, err := os.Stat(filepath.Join(runDir, "prompt.md")); err != nil {
		t.Fatalf("prompt.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "output.md")); err != nil {
		t.Fatalf("output.md missing: %v", err)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	if strings.TrimSpace(string(argsData)) != "--foo|bar baz" {
		t.Fatalf("wrapped args mismatch: %q", strings.TrimSpace(string(argsData)))
	}

	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	parts := strings.Split(strings.TrimSpace(string(envData)), "|")
	if len(parts) != 5 {
		t.Fatalf("unexpected env payload: %q", strings.TrimSpace(string(envData)))
	}
	if parts[0] != projectID {
		t.Fatalf("JRUN_PROJECT_ID=%q, want %q", parts[0], projectID)
	}
	if parts[1] != taskID {
		t.Fatalf("JRUN_TASK_ID=%q, want %q", parts[1], taskID)
	}
	if parts[2] != info.RunID {
		t.Fatalf("JRUN_ID=%q, want %q", parts[2], info.RunID)
	}
	if parts[3] != filepath.Join(root, projectID, taskID) {
		t.Fatalf("TASK_FOLDER=%q", parts[3])
	}
	if parts[4] != runDir {
		t.Fatalf("RUN_FOLDER=%q, want %q", parts[4], runDir)
	}

	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
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

func TestRunWrapFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrap integration helper script is unix-only in this test")
	}

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createWrapCLI(t, binDir, "gemini", 3)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectID := "project"
	taskID := "task-20260221-120100-wrap-failure"
	err := RunWrap(projectID, taskID, "gemini", []string{"--failing"}, WrapOptions{
		RootDir:    root,
		WorkingDir: root,
	})
	if err == nil {
		t.Fatal("expected error for failing wrapped command")
	}

	runDir := singleRunDir(t, root, projectID, taskID)
	info, readErr := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if readErr != nil {
		t.Fatalf("read run-info: %v", readErr)
	}
	if info.Status != storage.StatusFailed {
		t.Fatalf("status=%q, want %q", info.Status, storage.StatusFailed)
	}
	if info.ExitCode != 3 {
		t.Fatalf("exit_code=%d, want 3", info.ExitCode)
	}
	if info.ErrorSummary == "" {
		t.Fatal("expected non-empty error_summary")
	}

	busPath := filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	bus, busErr := messagebus.NewMessageBus(busPath)
	if busErr != nil {
		t.Fatalf("open message bus: %v", busErr)
	}
	msgs, busReadErr := bus.ReadMessages("")
	if busReadErr != nil {
		t.Fatalf("read messages: %v", busReadErr)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Type != messagebus.EventTypeRunStart {
		t.Fatalf("msg[0].type=%q, want %q", msgs[0].Type, messagebus.EventTypeRunStart)
	}
	if msgs[1].Type != messagebus.EventTypeRunCrash {
		t.Fatalf("msg[1].type=%q, want %q", msgs[1].Type, messagebus.EventTypeRunCrash)
	}
}

func TestRunWrapDoesNotOverwriteTaskMD(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrap integration helper script is unix-only in this test")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-20260221-120200-wrap-taskmd"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	original := "existing task prompt\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createWrapCLI(t, binDir, "claude", 0)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunWrap(projectID, taskID, "claude", []string{"--ok"}, WrapOptions{
		RootDir:    root,
		WorkingDir: root,
		TaskPrompt: "new prompt should not replace existing TASK.md",
	}); err != nil {
		t.Fatalf("RunWrap: %v", err)
	}

	taskMD, err := os.ReadFile(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if string(taskMD) != original {
		t.Fatalf("TASK.md overwritten: got %q want %q", string(taskMD), original)
	}
}

func singleRunDir(t *testing.T, root, projectID, taskID string) string {
	t.Helper()
	runsDir := filepath.Join(root, projectID, taskID, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 run dir, got %d", len(entries))
	}
	return filepath.Join(runsDir, entries[0].Name())
}

func createWrapCLI(t *testing.T, dir, name string, exitCode int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Fatalf("createWrapCLI should not be used on windows")
	}
	path := filepath.Join(dir, name)
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "` + name + ` 1.0.0"
  exit 0
fi
if [ -n "$WRAP_ARGS_FILE" ]; then
  : > "$WRAP_ARGS_FILE"
  first=1
  for arg in "$@"; do
    if [ "$first" -eq 0 ]; then
      printf "|" >> "$WRAP_ARGS_FILE"
    fi
    printf "%s" "$arg" >> "$WRAP_ARGS_FILE"
    first=0
  done
fi
if [ -n "$WRAP_ENV_FILE" ]; then
  printf "%s|%s|%s|%s|%s" "$JRUN_PROJECT_ID" "$JRUN_TASK_ID" "$JRUN_ID" "$TASK_FOLDER" "$RUN_FOLDER" > "$WRAP_ENV_FILE"
fi
echo "wrapped stdout"
echo "wrapped stderr" >&2
exit ` + strconv.Itoa(exitCode) + `
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrap script: %v", err)
	}
}
