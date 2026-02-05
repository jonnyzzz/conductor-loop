package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const (
	envOrchStubStdout   = "ORCH_STUB_STDOUT"
	envOrchStubStderr   = "ORCH_STUB_STDERR"
	envOrchStubSleepMs  = "ORCH_STUB_SLEEP_MS"
	envOrchStubDoneFile = "ORCH_STUB_DONE_FILE"
)

func TestRunJob(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task-001"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("job task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	parentRunID := "parent-run"
	stdout := "job output"

	opts := runner.JobOptions{
		RootDir:     root,
		Agent:       "codex",
		Prompt:      "job prompt",
		WorkingDir:  taskDir,
		ParentRunID: parentRunID,
		Environment: map[string]string{
			envOrchStubStdout: stdout,
		},
	}

	if err := runner.RunJob(projectID, taskID, opts); err != nil {
		t.Fatalf("RunJob: %v", err)
	}

	runDir := singleRunDir(t, taskDir)
	info := readRunInfo(t, runDir)
	if info.ParentRunID != parentRunID {
		t.Fatalf("parent run id: want %q got %q", parentRunID, info.ParentRunID)
	}
	if info.Status != storage.StatusCompleted {
		t.Fatalf("status: want %q got %q", storage.StatusCompleted, info.Status)
	}
	outputPath := filepath.Join(runDir, "output.md")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if !strings.Contains(string(data), stdout) {
		t.Fatalf("output.md missing stdout, got %q", string(data))
	}
}

func TestRunTask(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task-002"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("root task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	donePath := filepath.Join(taskDir, "DONE")
	stdout := "root output"

	opts := runner.TaskOptions{
		RootDir:    root,
		Agent:      "codex",
		WorkingDir: taskDir,
		Environment: map[string]string{
			envOrchStubDoneFile: donePath,
			envOrchStubStdout:   stdout,
		},
	}

	if err := runner.RunTask(projectID, taskID, opts); err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if _, err := os.Stat(donePath); err != nil {
		t.Fatalf("DONE missing: %v", err)
	}

	runDir := singleRunDir(t, taskDir)
	outputPath := filepath.Join(runDir, "output.md")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if !strings.Contains(string(data), stdout) {
		t.Fatalf("output.md missing stdout, got %q", string(data))
	}
}

func TestParentChildRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not reliable on windows")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-003"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("parent child task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	childErr := make(chan error, 1)
	go func() {
		childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
			RootDir:     root,
			Agent:       "codex",
			Prompt:      "child prompt",
			WorkingDir:  taskDir,
			ParentRunID: "root-run",
			Environment: map[string]string{
				envOrchStubSleepMs: "300",
				envOrchStubStdout:  "child",
			},
		})
	}()

	time.Sleep(120 * time.Millisecond)

	start := time.Now()
	err := runner.RunTask(projectID, taskID, runner.TaskOptions{
		RootDir:      root,
		Agent:        "codex",
		WorkingDir:   taskDir,
		WaitTimeout:  2 * time.Second,
		PollInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 200*time.Millisecond {
		t.Fatalf("expected RunTask to wait, elapsed %v", elapsed)
	}

	if err := <-childErr; err != nil {
		t.Fatalf("child run failed: %v", err)
	}
}

func TestNestedRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not reliable on windows")
	}

	root := t.TempDir()
	projectID := "project"
	taskID := "task-004"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("nested task"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	childErr := make(chan error, 2)
	go func() {
		childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
			RootDir:     root,
			Agent:       "codex",
			Prompt:      "child 1",
			WorkingDir:  taskDir,
			ParentRunID: "root-run",
			Environment: map[string]string{
				envOrchStubSleepMs: "400",
				envOrchStubStdout:  "child-one",
			},
		})
	}()
	go func() {
		childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
			RootDir:     root,
			Agent:       "codex",
			Prompt:      "child 2",
			WorkingDir:  taskDir,
			ParentRunID: "child-run",
			Environment: map[string]string{
				envOrchStubSleepMs: "250",
				envOrchStubStdout:  "child-two",
			},
		})
	}()

	time.Sleep(150 * time.Millisecond)

	start := time.Now()
	err := runner.RunTask(projectID, taskID, runner.TaskOptions{
		RootDir:      root,
		Agent:        "codex",
		WorkingDir:   taskDir,
		WaitTimeout:  2 * time.Second,
		PollInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 300*time.Millisecond {
		t.Fatalf("expected RunTask to wait for nested children, elapsed %v", elapsed)
	}

	for i := 0; i < 2; i++ {
		if err := <-childErr; err != nil {
			t.Fatalf("child run failed: %v", err)
		}
	}
}

func buildCodexStub(t *testing.T, dir string) string {
	t.Helper()

	stubPath := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}

	src := `package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func main() {
	if sleep := os.Getenv("` + envOrchStubSleepMs + `"); sleep != "" {
		if ms, err := strconv.Atoi(sleep); err == nil {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}
	if path := os.Getenv("` + envOrchStubDoneFile + `"); path != "" {
		_ = os.WriteFile(path, []byte(""), 0o644)
	}
	if out := os.Getenv("` + envOrchStubStdout + `"); out != "" {
		_, _ = fmt.Fprint(os.Stdout, out)
	} else {
		_, _ = fmt.Fprint(os.Stdout, "stub output")
	}
	if errText := os.Getenv("` + envOrchStubStderr + `"); errText != "" {
		_, _ = fmt.Fprint(os.Stderr, errText)
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}
`

	srcPath := filepath.Join(dir, "codex_stub.go")
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

func singleRunDir(t *testing.T, taskDir string) string {
	t.Helper()
	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	var runDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			runDirs = append(runDirs, filepath.Join(runsDir, entry.Name()))
		}
	}
	if len(runDirs) != 1 {
		t.Fatalf("expected 1 run dir, got %d", len(runDirs))
	}
	return runDirs[0]
}

func readRunInfo(t *testing.T, runDir string) *storage.RunInfo {
	t.Helper()
	info, err := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	return info
}
