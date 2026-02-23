package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// ---- helpers ----

// writeTODO writes a TODOs.md file with the given content to the given directory.
func writeTODO(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "TODOs.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write TODOs.md: %v", err)
	}
	return path
}

// makeMonitorRun creates a run directory with a run-info.yaml for monitor tests.
func makeMonitorRun(t *testing.T, root, project, task, runID, status string, exitCode int) string {
	t.Helper()
	runDir := filepath.Join(root, project, task, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     runID,
		ProjectID: project,
		TaskID:    task,
		AgentType: "claude",
		Status:    status,
		StartTime: time.Now().Add(-10 * time.Minute).UTC(),
		ExitCode:  exitCode,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return runDir
}

// writeMonitorOutput writes a non-empty output.md to a run directory.
func writeMonitorOutput(t *testing.T, runDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(runDir, "output.md"), []byte("## Summary\nDone.\n"), 0o644); err != nil {
		t.Fatalf("write output.md: %v", err)
	}
}

// touchMonitorDONE creates a DONE marker in the task directory.
func touchMonitorDONE(t *testing.T, root, project, task string) {
	t.Helper()
	taskDir := filepath.Join(root, project, task)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}
}

// ---- parseTodoEntries ----

func TestParseTodoEntries_Empty(t *testing.T) {
	dir := t.TempDir()
	path := writeTODO(t, dir, "# TODOs\n\nNothing here.\n")
	entries, err := parseTodoEntries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseTodoEntries_CheckedItemsSkipped(t *testing.T) {
	dir := t.TempDir()
	content := "- [x] task-20260101-000001-done\n- [ ] task-20260101-000002-pending\n"
	path := writeTODO(t, dir, content)
	entries, err := parseTodoEntries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TaskID != "task-20260101-000002-pending" {
		t.Errorf("unexpected task ID: %q", entries[0].TaskID)
	}
}

func TestParseTodoEntries_TaskIDEmbeddedInText(t *testing.T) {
	dir := t.TempDir()
	content := "- [ ] Continue work on `task-20260201-120000-my-feature`: description\n"
	path := writeTODO(t, dir, content)
	entries, err := parseTodoEntries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(entries), entries)
	}
	if entries[0].TaskID != "task-20260201-120000-my-feature" {
		t.Errorf("unexpected task ID: %q", entries[0].TaskID)
	}
}

func TestParseTodoEntries_ItemWithoutTaskIDIgnored(t *testing.T) {
	dir := t.TempDir()
	content := "- [ ] No task ID here — just plain text\n- [ ] task-20260101-000001-valid\n"
	path := writeTODO(t, dir, content)
	entries, err := parseTodoEntries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestParseTodoEntries_Indented(t *testing.T) {
	dir := t.TempDir()
	content := "  - [ ] task-20260101-000001-indented\n"
	path := writeTODO(t, dir, content)
	entries, err := parseTodoEntries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for indented item, got %d", len(entries))
	}
}

func TestParseTodoEntries_FileNotFound(t *testing.T) {
	_, err := parseTodoEntries("/nonexistent/path/TODOs.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// ---- extractTaskIDFromText ----

func TestExtractTaskIDFromText_Found(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"task-20260101-120000-my-task", "task-20260101-120000-my-task"},
		{"`task-20260101-120000-backtick`", "task-20260101-120000-backtick"},
		{"prefix task-20260101-120000-embedded suffix", "task-20260101-120000-embedded"},
	}
	for _, tt := range tests {
		got := extractTaskIDFromText(tt.input)
		if got != tt.want {
			t.Errorf("extractTaskIDFromText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractTaskIDFromText_NotFound(t *testing.T) {
	inputs := []string{
		"no task id here",
		"task-2026-short",
		"task-20260101-my-feature", // missing time component
	}
	for _, input := range inputs {
		got := extractTaskIDFromText(input)
		if got != "" {
			t.Errorf("extractTaskIDFromText(%q) = %q, want empty", input, got)
		}
	}
}

// ---- assessMonitorTask ----

func TestAssessMonitorTask_TaskNotFound(t *testing.T) {
	root := t.TempDir()
	state := assessMonitorTask(root, "proj", "task-20260101-000001-aaa", 20*time.Minute, time.Now())
	if state.Exists {
		t.Error("expected Exists=false for non-existent task")
	}
	if state.Status != "-" {
		t.Errorf("expected Status='-', got %q", state.Status)
	}
}

func TestAssessMonitorTask_TaskExistsDONE(t *testing.T) {
	root := t.TempDir()
	const task = "task-20260101-000001-aaa"
	touchMonitorDONE(t, root, "proj", task)
	state := assessMonitorTask(root, "proj", task, 20*time.Minute, time.Now())
	if !state.Exists {
		t.Error("expected Exists=true")
	}
	if !state.Done {
		t.Error("expected Done=true")
	}
}

func TestAssessMonitorTask_FailedRun(t *testing.T) {
	root := t.TempDir()
	const task = "task-20260101-000002-bbb"
	makeMonitorRun(t, root, "proj", task, "run-001", storage.StatusFailed, 1)
	state := assessMonitorTask(root, "proj", task, 20*time.Minute, time.Now())
	if !state.Exists {
		t.Error("expected Exists=true")
	}
	if !state.HasRuns {
		t.Error("expected HasRuns=true")
	}
	if state.Status != storage.StatusFailed {
		t.Errorf("expected Status=%q, got %q", storage.StatusFailed, state.Status)
	}
}

func TestAssessMonitorTask_CompletedRun(t *testing.T) {
	root := t.TempDir()
	const task = "task-20260101-000003-ccc"
	makeMonitorRun(t, root, "proj", task, "run-001", storage.StatusCompleted, 0)
	state := assessMonitorTask(root, "proj", task, 20*time.Minute, time.Now())
	if state.Status != storage.StatusCompleted {
		t.Errorf("expected Status=%q, got %q", storage.StatusCompleted, state.Status)
	}
	if state.Done {
		t.Error("expected Done=false (no DONE file)")
	}
}

// ---- decideMonitorAction ----

func TestDecideMonitorAction_Done(t *testing.T) {
	state := monitorTaskState{TaskID: "task-20260101-000001-aaa", Done: true}
	d := decideMonitorAction(state, "", "")
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip for Done task, got %q", d.Action)
	}
}

func TestDecideMonitorAction_NoRuns(t *testing.T) {
	state := monitorTaskState{TaskID: "task-20260101-000001-aaa", Exists: false}
	d := decideMonitorAction(state, "", "")
	if d.Action != monitorActionStart {
		t.Errorf("expected start for task with no runs, got %q", d.Action)
	}
}

func TestDecideMonitorAction_Failed(t *testing.T) {
	state := monitorTaskState{
		TaskID:  "task-20260101-000001-aaa",
		Exists:  true,
		HasRuns: true,
		Status:  storage.StatusFailed,
	}
	d := decideMonitorAction(state, "", "")
	if d.Action != monitorActionResume {
		t.Errorf("expected resume for failed task, got %q", d.Action)
	}
}

func TestDecideMonitorAction_RunningAlive(t *testing.T) {
	state := monitorTaskState{
		TaskID:   "task-20260101-000001-aaa",
		Exists:   true,
		HasRuns:  true,
		Status:   storage.StatusRunning,
		PIDAlive: true,
		IsStale:  false,
	}
	d := decideMonitorAction(state, "", "")
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip for actively running task, got %q", d.Action)
	}
}

func TestDecideMonitorAction_RunningStale(t *testing.T) {
	state := monitorTaskState{
		TaskID:   "task-20260101-000001-aaa",
		Exists:   true,
		HasRuns:  true,
		Status:   storage.StatusRunning,
		PIDAlive: true,
		IsStale:  true,
	}
	d := decideMonitorAction(state, "", "")
	if d.Action != monitorActionRecover {
		t.Errorf("expected recover for stale running task, got %q", d.Action)
	}
}

func TestDecideMonitorAction_RunningDeadPID(t *testing.T) {
	state := monitorTaskState{
		TaskID:   "task-20260101-000001-aaa",
		Exists:   true,
		HasRuns:  true,
		Status:   storage.StatusRunning,
		PIDAlive: false,
	}
	d := decideMonitorAction(state, "", "")
	if d.Action != monitorActionResume {
		t.Errorf("expected resume for running-with-dead-PID task, got %q", d.Action)
	}
}

func TestDecideMonitorAction_CompletedWithOutput(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000001-aaa"
	runDir := makeMonitorRun(t, root, proj, task, "run-001", storage.StatusCompleted, 0)
	writeMonitorOutput(t, runDir)
	info, _ := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	state := monitorTaskState{
		TaskID:    task,
		Exists:    true,
		HasRuns:   true,
		Status:    storage.StatusCompleted,
		LatestRun: "run-001",
		Info:      info,
	}
	d := decideMonitorAction(state, root, proj)
	if d.Action != monitorActionFinalize {
		t.Errorf("expected finalize for completed task with output, got %q", d.Action)
	}
}

func TestDecideMonitorAction_CompletedNoOutput(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000001-aaa"
	runDir := makeMonitorRun(t, root, proj, task, "run-001", storage.StatusCompleted, 0)
	info, _ := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	state := monitorTaskState{
		TaskID:    task,
		Exists:    true,
		HasRuns:   true,
		Status:    storage.StatusCompleted,
		LatestRun: "run-001",
		Info:      info,
	}
	d := decideMonitorAction(state, root, proj)
	if d.Action != monitorActionSkip {
		t.Errorf("expected skip for completed task without output, got %q", d.Action)
	}
}

// ---- monitorOutputNonEmpty ----

func TestMonitorOutputNonEmpty_OutputExists(t *testing.T) {
	root := t.TempDir()
	runDir := makeMonitorRun(t, root, "proj", "task-20260101-000001-aaa", "run-001", storage.StatusCompleted, 0)
	writeMonitorOutput(t, runDir)
	info, _ := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	got := monitorOutputNonEmpty(root, "proj", "task-20260101-000001-aaa", "run-001", info)
	if !got {
		t.Error("expected non-empty output to be detected")
	}
}

func TestMonitorOutputNonEmpty_NoFile(t *testing.T) {
	root := t.TempDir()
	runDir := makeMonitorRun(t, root, "proj", "task-20260101-000001-aaa", "run-001", storage.StatusCompleted, 0)
	info, _ := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	got := monitorOutputNonEmpty(root, "proj", "task-20260101-000001-aaa", "run-001", info)
	if got {
		t.Error("expected false when output.md does not exist")
	}
}

func TestMonitorOutputNonEmpty_EmptyFile(t *testing.T) {
	root := t.TempDir()
	runDir := makeMonitorRun(t, root, "proj", "task-20260101-000001-aaa", "run-001", storage.StatusCompleted, 0)
	if err := os.WriteFile(filepath.Join(runDir, "output.md"), []byte("   \n"), 0o644); err != nil {
		t.Fatalf("write output.md: %v", err)
	}
	info, _ := storage.ReadRunInfo(filepath.Join(runDir, "run-info.yaml"))
	got := monitorOutputNonEmpty(root, "proj", "task-20260101-000001-aaa", "run-001", info)
	if got {
		t.Error("expected false for whitespace-only output.md")
	}
}

// ---- monitorPass (dry-run) ----

func TestMonitorPass_DryRunReportsActionWithoutExecuting(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000001-aaa"
	dir := t.TempDir()
	todoPath := writeTODO(t, dir, fmt.Sprintf("- [ ] %s\n", task))

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		Agent:      "claude",
		StaleAfter: 20 * time.Minute,
		RateLimit:  0,
		DryRun:     true,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()

	output := buf.String()
	if !strings.Contains(output, task) {
		t.Errorf("expected task ID in output, got:\n%s", output)
	}
	if !strings.Contains(output, monitorActionStart) {
		t.Errorf("expected 'start' action in output, got:\n%s", output)
	}
	// DONE file must NOT be created in dry-run mode.
	doneFile := filepath.Join(root, proj, task, "DONE")
	if _, err := os.Stat(doneFile); !os.IsNotExist(err) {
		t.Error("DONE file must not be created in dry-run mode")
	}
}

func TestMonitorPass_FinalizeCreatesDoNE(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000001-aaa"
	runDir := makeMonitorRun(t, root, proj, task, "run-001", storage.StatusCompleted, 0)
	writeMonitorOutput(t, runDir)

	dir := t.TempDir()
	todoPath := writeTODO(t, dir, fmt.Sprintf("- [ ] %s\n", task))

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		StaleAfter: 20 * time.Minute,
		RateLimit:  0,
		DryRun:     false,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()

	doneFile := filepath.Join(root, proj, task, "DONE")
	if _, err := os.Stat(doneFile); err != nil {
		t.Errorf("expected DONE file to be created: %v", err)
	}
	if !strings.Contains(buf.String(), monitorActionFinalize) {
		t.Errorf("expected 'finalize' in output, got:\n%s", buf.String())
	}
	// TODO item should be checked off after finalize.
	content, _ := os.ReadFile(todoPath)
	if strings.Contains(string(content), "- [ ] "+task) {
		t.Errorf("expected TODO item to be checked off in %s, got:\n%s", todoPath, string(content))
	}
	if !strings.Contains(string(content), "- [x] "+task) {
		t.Errorf("expected [x] marker in TODO file, got:\n%s", string(content))
	}
}

// ---- updateTodoFile ----

func TestUpdateTodoFile_MarksUncheckedItem(t *testing.T) {
	dir := t.TempDir()
	const task = "task-20260101-000001-aaa"
	path := writeTODO(t, dir, fmt.Sprintf("- [ ] %s\n- [ ] task-20260101-000002-bbb\n", task))
	if err := updateTodoFile(path, task); err != nil {
		t.Fatalf("updateTodoFile: %v", err)
	}
	content, _ := os.ReadFile(path)
	lines := strings.Split(string(content), "\n")
	if !strings.HasPrefix(lines[0], "- [x]") {
		t.Errorf("expected first line checked, got: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "- [ ]") {
		t.Errorf("expected second line unchanged, got: %q", lines[1])
	}
}

func TestUpdateTodoFile_AlreadyCheckedNotModified(t *testing.T) {
	dir := t.TempDir()
	const task = "task-20260101-000001-aaa"
	path := writeTODO(t, dir, fmt.Sprintf("- [x] %s\n", task))
	original, _ := os.ReadFile(path)
	if err := updateTodoFile(path, task); err != nil {
		t.Fatalf("updateTodoFile: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(original) != string(after) {
		t.Errorf("expected already-checked item to be unchanged, got:\n%s", string(after))
	}
}

func TestUpdateTodoFile_TaskNotFoundIsNoop(t *testing.T) {
	dir := t.TempDir()
	path := writeTODO(t, dir, "- [ ] task-20260101-000001-aaa\n")
	original, _ := os.ReadFile(path)
	if err := updateTodoFile(path, "task-20260101-000099-zzz"); err != nil {
		t.Fatalf("updateTodoFile: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(original) != string(after) {
		t.Errorf("expected no change for non-matching task, got: %q", string(after))
	}
}

func TestUpdateTodoFile_MissingFileReturnsError(t *testing.T) {
	err := updateTodoFile("/nonexistent/TODOs.md", "task-20260101-000001-aaa")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMonitorPass_SkipsAlreadyDone(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000001-aaa"
	makeMonitorRun(t, root, proj, task, "run-001", storage.StatusCompleted, 0)
	touchMonitorDONE(t, root, proj, task)

	dir := t.TempDir()
	todoPath := writeTODO(t, dir, fmt.Sprintf("- [ ] %s\n", task))

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		StaleAfter: 20 * time.Minute,
		RateLimit:  0,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()

	if !strings.Contains(buf.String(), monitorActionSkip) {
		t.Errorf("expected 'skip' action for DONE task, got:\n%s", buf.String())
	}
}

func TestMonitorPass_EmptyTODOs(t *testing.T) {
	root := t.TempDir()
	dir := t.TempDir()
	todoPath := writeTODO(t, dir, "# No tasks\n")

	opts := monitorOpts{
		RootDir:   root,
		ProjectID: "proj",
		TODOFile:  todoPath,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no pending task IDs") {
		t.Errorf("expected 'no pending task IDs' message, got:\n%s", buf.String())
	}
}

func TestMonitorPass_MissingTODOFile(t *testing.T) {
	opts := monitorOpts{
		RootDir:   t.TempDir(),
		ProjectID: "proj",
		TODOFile:  "/nonexistent/TODOs.md",
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	err := monitorPass(&buf, opts, &wg, time.Now())
	if err == nil {
		t.Fatal("expected error for missing TODO file")
	}
}

// ---- runMonitor --once ----

func TestRunMonitor_OnceExitsAfterPass(t *testing.T) {
	root := t.TempDir()
	dir := t.TempDir()
	todoPath := writeTODO(t, dir, "# No tasks\n")

	opts := monitorOpts{
		RootDir:   root,
		ProjectID: "proj",
		TODOFile:  todoPath,
		Interval:  time.Hour, // would block in daemon mode
		Once:      true,
	}
	var buf bytes.Buffer
	err := runMonitor(&buf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- command registration ----

func TestMonitorCmd_HelpText(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"monitor", "--help"})
	_ = cmd.Execute()
	output := buf.String()
	for _, want := range []string{"monitor", "project", "todo", "stale-after", "rate-limit", "dry-run", "once"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in monitor help, got:\n%s", want, output)
		}
	}
}

func TestMonitorCmd_RequiresProject(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"monitor", "--todo", "/tmp/todos.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --project is missing")
	}
}

// ---- rate limiting ----

func TestMonitorPass_RateLimitApplied(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	dir := t.TempDir()

	// Two failed tasks both needing resume; rate limit = 100ms.
	tasks := []string{
		"task-20260101-000001-aaa",
		"task-20260101-000002-bbb",
	}
	todoContent := ""
	for _, task := range tasks {
		makeMonitorRun(t, root, proj, task, "run-001", storage.StatusFailed, 1)
		todoContent += fmt.Sprintf("- [ ] %s\n", task)
	}
	todoPath := writeTODO(t, dir, todoContent)

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		Agent:      "claude",
		StaleAfter: 20 * time.Minute,
		RateLimit:  100 * time.Millisecond,
		DryRun:     true, // don't actually start jobs
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	start := time.Now()
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// Expect at least ~100ms for the single sleep between two actions.
	if elapsed < 80*time.Millisecond {
		t.Errorf("expected ≥100ms for rate limit, got %s", elapsed)
	}
	// Should not be unreasonably slow.
	if elapsed > 800*time.Millisecond {
		t.Errorf("rate limit unexpectedly slow: %s", elapsed)
	}
}

// ---- multiple tasks in one pass ----

func TestMonitorPass_MultipleTasks(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	dir := t.TempDir()

	taskDone := "task-20260101-000001-done"
	taskFailed := "task-20260101-000002-fail"
	taskCompleted := "task-20260101-000003-comp"

	// taskDone: has DONE marker
	touchMonitorDONE(t, root, proj, taskDone)
	// taskFailed: failed run
	makeMonitorRun(t, root, proj, taskFailed, "run-001", storage.StatusFailed, 1)
	// taskCompleted: completed with output
	runDir := makeMonitorRun(t, root, proj, taskCompleted, "run-001", storage.StatusCompleted, 0)
	writeMonitorOutput(t, runDir)

	todoPath := writeTODO(t, dir,
		fmt.Sprintf("- [ ] %s\n- [ ] %s\n- [ ] %s\n", taskDone, taskFailed, taskCompleted),
	)

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		Agent:      "claude",
		StaleAfter: 20 * time.Minute,
		RateLimit:  0,
		DryRun:     true,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()

	output := buf.String()
	if !strings.Contains(output, taskDone) {
		t.Errorf("expected taskDone in output")
	}
	if !strings.Contains(output, taskFailed) {
		t.Errorf("expected taskFailed in output")
	}
	if !strings.Contains(output, taskCompleted) {
		t.Errorf("expected taskCompleted in output")
	}
	// Verify correct actions are planned.
	if !strings.Contains(output, monitorActionSkip) {
		t.Errorf("expected 'skip' for DONE task in output:\n%s", output)
	}
	if !strings.Contains(output, monitorActionResume) {
		t.Errorf("expected 'resume' for failed task in output:\n%s", output)
	}
	if !strings.Contains(output, monitorActionFinalize) {
		t.Errorf("expected 'finalize' for completed task in output:\n%s", output)
	}
}

func TestMonitorPass_FinalizeUpdatesTODOFile(t *testing.T) {
	root := t.TempDir()
	const proj = "proj"
	const task = "task-20260101-000001-abc"
	runDir := makeMonitorRun(t, root, proj, task, "run-001", storage.StatusCompleted, 0)
	writeMonitorOutput(t, runDir)

	dir := t.TempDir()
	todoPath := writeTODO(t, dir, fmt.Sprintf("- [ ] %s\n", task))

	opts := monitorOpts{
		RootDir:    root,
		ProjectID:  proj,
		TODOFile:   todoPath,
		StaleAfter: 20 * time.Minute,
		RateLimit:  0,
		DryRun:     false,
	}
	var buf bytes.Buffer
	var wg sync.WaitGroup
	if err := monitorPass(&buf, opts, &wg, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()

	// Verify TODOs.md was updated
	content, err := os.ReadFile(todoPath)
	if err != nil {
		t.Fatalf("read TODOs.md: %v", err)
	}
	if !strings.Contains(string(content), fmt.Sprintf("- [x] %s", task)) {
		t.Errorf("expected task to be marked checked in TODOs.md, got:\n%s\nMonitor output:\n%s", string(content), buf.String())
	}
}
