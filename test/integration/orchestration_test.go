package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const (
	envOrchStubStdout   = "ORCH_STUB_STDOUT"
	envOrchStubStderr   = "ORCH_STUB_STDERR"
	envOrchStubSleepMs  = "ORCH_STUB_SLEEP_MS"
	envOrchStubDoneFile = "ORCH_STUB_DONE_FILE"

	// envOrchChainAgentBin holds the path to the real run-agent binary that
	// chain stubs use to spawn child jobs.
	envOrchChainAgentBin = "ORCH_CHAIN_AGENT_BIN"
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
				// 600ms gives RunTask a comfortable ~400ms margin after the 120ms pre-sleep.
				// Previously 300ms gave only ~60ms margin (too tight under load).
				envOrchStubSleepMs: "600",
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
		WaitTimeout:  3 * time.Second,
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
				envOrchStubSleepMs: "800",
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
				envOrchStubSleepMs: "500",
				envOrchStubStdout:  "child-two",
			},
		})
	}()

	// Poll until both children have registered their run-info.yaml with status=running.
	// This avoids a race where RunTask calls FindActiveChildren before the goroutines
	// have finished their setup (dir creation + agent version detection + process spawn).
	runsDir := filepath.Join(taskDir, "runs")
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		entries, _ := os.ReadDir(runsDir)
		runningCount := 0
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			info, err := storage.ReadRunInfo(filepath.Join(runsDir, e.Name(), "run-info.yaml"))
			if err == nil && info.Status == storage.StatusRunning {
				runningCount++
			}
		}
		if runningCount >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

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

// TestRunJobMessageBusEventOrdering verifies that RUN_START is written to the
// message bus before the agent completes and RUN_STOP is written after, in the
// correct order (ISSUE-020).
func TestRunJobMessageBusEventOrdering(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task-bus-order"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("bus ordering test"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	opts := runner.JobOptions{
		RootDir:    root,
		Agent:      "codex",
		Prompt:     "test ordering",
		WorkingDir: taskDir,
		Environment: map[string]string{
			envOrchStubStdout: "ordering output",
		},
	}

	if err := runner.RunJob(projectID, taskID, opts); err != nil {
		t.Fatalf("RunJob: %v", err)
	}

	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}

	var startIdx, stopIdx int
	startIdx, stopIdx = -1, -1
	for i, m := range msgs {
		switch m.Type {
		case messagebus.EventTypeRunStart:
			if startIdx == -1 {
				startIdx = i
			}
		case messagebus.EventTypeRunStop, messagebus.EventTypeRunCrash:
			if stopIdx == -1 {
				stopIdx = i
			}
		}
	}

	if startIdx == -1 {
		t.Fatalf("RUN_START event not found in message bus")
	}
	if stopIdx == -1 {
		t.Fatalf("RUN_STOP or RUN_CRASH event not found in message bus")
	}
	if startIdx >= stopIdx {
		t.Fatalf("RUN_START (index %d) must appear before RUN_STOP/RUN_CRASH (index %d)", startIdx, stopIdx)
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

// TestChainClaudeCodexGemini verifies a three-level agent chain within a
// single project/task:
//
//	claude run  →  spawns codex run (parent_run_id = claude's run ID)
//	codex run   →  spawns gemini run (parent_run_id = codex's run ID)
//	gemini run  →  exits successfully
//
// All three runs must complete with the correct parent_run_id linkage.
func TestChainClaudeCodexGemini(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-based chain stubs not supported on windows")
	}

	const (
		chainProjectID = "test"
		chainTaskID    = "task-20260101-120000-chain-test"
	)

	root := t.TempDir()
	taskDir := filepath.Join(root, chainProjectID, chainTaskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("chain integration test"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	// Build the real run-agent binary; chain stubs exec it to spawn child jobs.
	runAgentDir := t.TempDir()
	runAgentBin := buildRunAgentBinary(t, runAgentDir)
	t.Setenv(envOrchChainAgentBin, runAgentBin)

	// Build three chain stubs into the same stub directory.
	stubDir := t.TempDir()
	buildChainClaudeStub(t, stubDir) // spawns a codex child job
	buildChainCodexStub(t, stubDir)  // spawns a gemini child job
	buildChainGeminiStub(t, stubDir) // terminal: outputs text and exits
	t.Setenv("PATH", prependPath(stubDir))

	opts := runner.JobOptions{
		RootDir:    root,
		Agent:      "claude",
		Prompt:     "start chain",
		WorkingDir: taskDir,
	}
	if err := runner.RunJob(chainProjectID, chainTaskID, opts); err != nil {
		t.Fatalf("RunJob (claude): %v", err)
	}

	// Collect all run directories created by the chain.
	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	var infos []*storage.RunInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		infos = append(infos, readRunInfo(t, filepath.Join(runsDir, e.Name())))
	}

	if len(infos) != 3 {
		t.Fatalf("expected 3 run dirs, got %d", len(infos))
	}

	// All runs must be completed.
	for _, info := range infos {
		if info.Status != storage.StatusCompleted {
			t.Errorf("run %s (agent=%s): status want %q got %q",
				info.RunID, info.AgentType, storage.StatusCompleted, info.Status)
		}
	}

	// Locate the root run (claude): no parent_run_id.
	var claudeRun *storage.RunInfo
	for _, info := range infos {
		if info.ParentRunID == "" {
			if claudeRun != nil {
				t.Fatalf("multiple runs with no parent_run_id")
			}
			claudeRun = info
		}
	}
	if claudeRun == nil {
		t.Fatalf("no root run found (every run has parent_run_id set)")
	}
	if claudeRun.AgentType != "claude" {
		t.Errorf("root run agent_type: want claude, got %q", claudeRun.AgentType)
	}

	// Locate the child run (codex): parent_run_id = claude's run ID.
	var codexRun *storage.RunInfo
	for _, info := range infos {
		if info.ParentRunID == claudeRun.RunID {
			codexRun = info
			break
		}
	}
	if codexRun == nil {
		t.Fatalf("no codex run with parent_run_id=%q", claudeRun.RunID)
	}
	if codexRun.AgentType != "codex" {
		t.Errorf("child run agent_type: want codex, got %q", codexRun.AgentType)
	}

	// Locate the grandchild run (gemini): parent_run_id = codex's run ID.
	var geminiRun *storage.RunInfo
	for _, info := range infos {
		if info.ParentRunID == codexRun.RunID {
			geminiRun = info
			break
		}
	}
	if geminiRun == nil {
		t.Fatalf("no gemini run with parent_run_id=%q", codexRun.RunID)
	}
	if geminiRun.AgentType != "gemini" {
		t.Errorf("grandchild run agent_type: want gemini, got %q", geminiRun.AgentType)
	}
}

// buildRunAgentBinary compiles the run-agent binary from source and returns
// its path. Used by chain stubs to spawn child jobs via the real CLI.
func buildRunAgentBinary(t *testing.T, dir string) string {
	t.Helper()
	binPath := filepath.Join(dir, "run-agent")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	repoRoot := runAgentCmdRepoRoot(t)
	mainPkg := filepath.Join(repoRoot, "cmd", "run-agent")
	cmd := exec.Command("go", "build", "-o", binPath, mainPkg)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build run-agent: %v\n%s", err, out)
	}
	return binPath
}

// buildChainClaudeStub compiles a "claude" stub into dir. When run by the
// runner it reads JRUN_ID/JRUN_PROJECT_ID/JRUN_TASK_ID/RUNS_DIR from env and
// spawns a codex child job via run-agent, then exits.
func buildChainClaudeStub(t *testing.T, dir string) {
	t.Helper()
	stubPath := filepath.Join(dir, "claude")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}
	src := `package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Respond to version-detection probes without spawning child jobs.
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Fprintln(os.Stdout, "claude chain stub v0.0.0")
			return
		}
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	runAgentBin := os.Getenv("` + envOrchChainAgentBin + `")
	// TASK_FOLDER = <root>/<project>/<task>; go up two levels to get root.
	rootDir := filepath.Dir(filepath.Dir(os.Getenv("TASK_FOLDER")))
	cmd := exec.Command(runAgentBin, "job",
		"--agent", "codex",
		"--project", os.Getenv("JRUN_PROJECT_ID"),
		"--task", os.Getenv("JRUN_TASK_ID"),
		"--root", rootDir,
		"--parent-run-id", os.Getenv("JRUN_ID"),
		"--prompt", "codex agent step")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "claude chain stub: spawn codex job: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "claude chain complete")
}
`
	srcPath := filepath.Join(dir, "chain_claude_stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write claude chain stub source: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", stubPath, srcPath)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build claude chain stub: %v\n%s", err, out)
	}
}

// buildChainCodexStub compiles a "codex" stub into dir. When run by the
// runner it reads JRUN_ID/JRUN_PROJECT_ID/JRUN_TASK_ID/RUNS_DIR from env and
// spawns a gemini child job via run-agent, then exits.
func buildChainCodexStub(t *testing.T, dir string) {
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
	"os/exec"
	"path/filepath"
)

func main() {
	// Respond to version-detection probes without spawning child jobs.
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Fprintln(os.Stdout, "codex chain stub v0.0.0")
			return
		}
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	runAgentBin := os.Getenv("` + envOrchChainAgentBin + `")
	// TASK_FOLDER = <root>/<project>/<task>; go up two levels to get root.
	rootDir := filepath.Dir(filepath.Dir(os.Getenv("TASK_FOLDER")))
	cmd := exec.Command(runAgentBin, "job",
		"--agent", "gemini",
		"--project", os.Getenv("JRUN_PROJECT_ID"),
		"--task", os.Getenv("JRUN_TASK_ID"),
		"--root", rootDir,
		"--parent-run-id", os.Getenv("JRUN_ID"),
		"--prompt", "gemini agent step")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "codex chain stub: spawn gemini job: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "codex chain complete")
}
`
	srcPath := filepath.Join(dir, "chain_codex_stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write codex chain stub source: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", stubPath, srcPath)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build codex chain stub: %v\n%s", err, out)
	}
}

// buildChainGeminiStub compiles a terminal "gemini" stub into dir. It
// responds to --help with an "output-format" line (satisfying the
// checkGeminiStreamJSONSupport version probe), and exits successfully on
// regular invocations.
func buildChainGeminiStub(t *testing.T, dir string) {
	t.Helper()
	stubPath := filepath.Join(dir, "gemini")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}
	src := `package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" {
			// Satisfy the checkGeminiStreamJSONSupport version probe.
			fmt.Fprintln(os.Stdout, "--output-format stream-json")
			return
		}
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	fmt.Fprintln(os.Stdout, "gemini chain complete")
}
`
	srcPath := filepath.Join(dir, "chain_gemini_stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write gemini chain stub source: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", stubPath, srcPath)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build gemini chain stub: %v\n%s", err, out)
	}
}
