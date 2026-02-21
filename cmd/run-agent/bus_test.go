package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

// TestBusReadWithProject verifies that bus read resolves the project-level bus
// (PROJECT-MESSAGE-BUS.md) when --project is given without --task.
func TestBusReadWithProject(t *testing.T) {
	root := t.TempDir()

	projDir := filepath.Join(root, "my-project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	busPath := filepath.Join(projDir, "PROJECT-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("create bus: %v", err)
	}
	if _, err := bus.AppendMessage(&messagebus.Message{
		Type:      "INFO",
		ProjectID: "my-project",
		Body:      "project level message",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "read",
		"--root", root,
		"--project", "my-project",
	})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus read failed: %v", runErr)
	}
	if !strings.Contains(output, "project level message") {
		t.Errorf("expected 'project level message' in output, got: %q", output)
	}
}

// TestBusReadWithProjectAndTask verifies that bus read resolves the task-level bus
// (TASK-MESSAGE-BUS.md) when both --project and --task are given.
func TestBusReadWithProjectAndTask(t *testing.T) {
	root := t.TempDir()

	taskDir := filepath.Join(root, "my-project", "task-20260101-000001-aa")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("create bus: %v", err)
	}
	if _, err := bus.AppendMessage(&messagebus.Message{
		Type:      "INFO",
		ProjectID: "my-project",
		TaskID:    "task-20260101-000001-aa",
		Body:      "task level message",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "read",
		"--root", root,
		"--project", "my-project",
		"--task", "task-20260101-000001-aa",
	})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus read failed: %v", runErr)
	}
	if !strings.Contains(output, "task level message") {
		t.Errorf("expected 'task level message' in output, got: %q", output)
	}
}

// TestBusReadBusFlagAndProjectError verifies that bus read returns an error when
// both --bus and --project are specified.
func TestBusReadBusFlagAndProjectError(t *testing.T) {
	root := t.TempDir()
	busPath := filepath.Join(root, "bus.md")

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "read",
		"--bus", busPath,
		"--project", "my-project",
	})

	var runErr error
	captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr == nil {
		t.Fatal("expected error when both --bus and --project are specified")
	}
	if !strings.Contains(runErr.Error(), "cannot specify both") {
		t.Errorf("expected 'cannot specify both' in error, got: %v", runErr)
	}
}

// TestBusPostWithProject verifies that bus post auto-resolves the project-level bus
// (PROJECT-MESSAGE-BUS.md) when --project is given without --bus.
func TestBusPostWithProject(t *testing.T) {
	// Clear MESSAGE_BUS to ensure project/task hierarchy is used for path resolution.
	t.Setenv("MESSAGE_BUS", "")
	root := t.TempDir()

	projDir := filepath.Join(root, "my-project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "post",
		"--root", root,
		"--project", "my-project",
		"--type", "INFO",
		"--body", "posted to project bus",
	})

	var runErr error
	captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus post failed: %v", runErr)
	}

	busPath := filepath.Join(projDir, "PROJECT-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("open bus: %v", err)
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Body != "posted to project bus" {
		t.Errorf("expected body 'posted to project bus', got %q", messages[0].Body)
	}
	if messages[0].ProjectID != "my-project" {
		t.Errorf("expected project_id 'my-project', got %q", messages[0].ProjectID)
	}
}

// TestBusPostWithProjectAndTask verifies that bus post auto-resolves the task-level bus
// (TASK-MESSAGE-BUS.md) when both --project and --task are given without --bus.
func TestBusPostWithProjectAndTask(t *testing.T) {
	// Clear MESSAGE_BUS to ensure project/task hierarchy is used for path resolution.
	t.Setenv("MESSAGE_BUS", "")
	root := t.TempDir()

	taskDir := filepath.Join(root, "my-project", "task-20260101-000001-aa")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "post",
		"--root", root,
		"--project", "my-project",
		"--task", "task-20260101-000001-aa",
		"--type", "INFO",
		"--body", "posted to task bus",
	})

	var runErr error
	captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus post failed: %v", runErr)
	}

	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("open bus: %v", err)
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Body != "posted to task bus" {
		t.Errorf("expected body 'posted to task bus', got %q", messages[0].Body)
	}
	if messages[0].TaskID != "task-20260101-000001-aa" {
		t.Errorf("expected task_id 'task-20260101-000001-aa', got %q", messages[0].TaskID)
	}
}

// TestBusRootDefaultsToRunsDir verifies that resolveBusFilePath defaults root to
// "./runs" when neither RUNS_DIR env var nor an explicit root is provided.
func TestBusRootDefaultsToRunsDir(t *testing.T) {
	t.Setenv("RUNS_DIR", "")

	// Project-level bus path
	got := resolveBusFilePath("", "my-project", "")
	want := filepath.Join("./runs", "my-project", "PROJECT-MESSAGE-BUS.md")
	if got != want {
		t.Errorf("project bus: got %q, want %q", got, want)
	}

	// Task-level bus path
	got = resolveBusFilePath("", "my-project", "task-20260101-000001-aa")
	want = filepath.Join("./runs", "my-project", "task-20260101-000001-aa", "TASK-MESSAGE-BUS.md")
	if got != want {
		t.Errorf("task bus: got %q, want %q", got, want)
	}
}

// TestBusRootUsesRunsDirEnvVar verifies that resolveBusFilePath uses the RUNS_DIR
// env var when no explicit root is provided.
func TestBusRootUsesRunsDirEnvVar(t *testing.T) {
	envRoot := t.TempDir()
	t.Setenv("RUNS_DIR", envRoot)

	got := resolveBusFilePath("", "my-project", "")
	want := filepath.Join(envRoot, "my-project", "PROJECT-MESSAGE-BUS.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestBusReadProjectLevelVsTaskLevel verifies that bus read reads from the correct
// bus file depending on whether --task is specified.
func TestBusReadProjectLevelVsTaskLevel(t *testing.T) {
	root := t.TempDir()

	projDir := filepath.Join(root, "proj")
	taskDir := filepath.Join(root, "proj", "task-20260101-000001-aa")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	projBus, err := messagebus.NewMessageBus(filepath.Join(projDir, "PROJECT-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("create project bus: %v", err)
	}
	if _, err := projBus.AppendMessage(&messagebus.Message{
		Type:      "INFO",
		ProjectID: "proj",
		Body:      "project-only msg",
	}); err != nil {
		t.Fatalf("append project: %v", err)
	}

	taskBus, err := messagebus.NewMessageBus(filepath.Join(taskDir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("create task bus: %v", err)
	}
	if _, err := taskBus.AppendMessage(&messagebus.Message{
		Type:      "INFO",
		ProjectID: "proj",
		TaskID:    "task-20260101-000001-aa",
		Body:      "task-only msg",
	}); err != nil {
		t.Fatalf("append task: %v", err)
	}

	// Read project-level bus (no --task)
	cmd := newRootCmd()
	cmd.SetArgs([]string{"bus", "read", "--root", root, "--project", "proj"})
	var out string
	var runErr error
	out = captureStdout(t, func() { runErr = cmd.Execute() })
	if runErr != nil {
		t.Fatalf("bus read project failed: %v", runErr)
	}
	if !strings.Contains(out, "project-only msg") {
		t.Errorf("expected project-only msg in project bus output, got: %q", out)
	}
	if strings.Contains(out, "task-only msg") {
		t.Errorf("task-only msg should not appear in project bus output, got: %q", out)
	}

	// Read task-level bus (with --task)
	cmd2 := newRootCmd()
	cmd2.SetArgs([]string{"bus", "read", "--root", root, "--project", "proj", "--task", "task-20260101-000001-aa"})
	var out2 string
	var runErr2 error
	out2 = captureStdout(t, func() { runErr2 = cmd2.Execute() })
	if runErr2 != nil {
		t.Fatalf("bus read task failed: %v", runErr2)
	}
	if !strings.Contains(out2, "task-only msg") {
		t.Errorf("expected task-only msg in task bus output, got: %q", out2)
	}
	if strings.Contains(out2, "project-only msg") {
		t.Errorf("project-only msg should not appear in task bus output, got: %q", out2)
	}
}

func TestDiscoverBusFilePathPrefersTaskProjectLegacy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "MESSAGE-BUS.md"), []byte("legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy bus: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "PROJECT-MESSAGE-BUS.md"), []byte("project\n"), 0o644); err != nil {
		t.Fatalf("write project bus: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "TASK-MESSAGE-BUS.md"), []byte("task\n"), 0o644); err != nil {
		t.Fatalf("write task bus: %v", err)
	}

	got, err := discoverBusFilePath(root)
	if err != nil {
		t.Fatalf("discoverBusFilePath: %v", err)
	}
	want := filepath.Join(root, "TASK-MESSAGE-BUS.md")
	if got != want {
		t.Fatalf("discovered %q, want %q", got, want)
	}
}

func TestBusDiscoverCmdFindsNearestBus(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "runs", "proj", "task-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	taskBus := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	if err := os.WriteFile(taskBus, []byte(""), 0o644); err != nil {
		t.Fatalf("write task bus: %v", err)
	}

	nested := filepath.Join(taskDir, "runs", "2026")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}
	t.Chdir(nested)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"bus", "discover"})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus discover failed: %v", runErr)
	}
	if strings.TrimSpace(output) != taskBus {
		t.Fatalf("discover output=%q, want %q", strings.TrimSpace(output), taskBus)
	}
}

func TestBusReadAutoDiscoversLegacyBusFromCWD(t *testing.T) {
	t.Setenv("MESSAGE_BUS", "")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "MESSAGE-BUS.md"), []byte(`# legacy bus
[2026-02-01 10:00:00] FACT: first
[2026-02-01 10:00:01] DECISION: second
`), 0o644); err != nil {
		t.Fatalf("write legacy bus: %v", err)
	}

	workDir := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	t.Chdir(workDir)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"bus", "read", "--tail", "1"})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus read failed: %v", runErr)
	}
	if !strings.Contains(output, "(DECISION) second") {
		t.Fatalf("expected latest legacy message in output, got: %q", output)
	}
	if strings.Contains(output, "first") {
		t.Fatalf("expected only tail=1 output, got: %q", output)
	}
}

func TestBusPostAutoDiscoversLegacyBusAndInfersProject(t *testing.T) {
	t.Setenv("MESSAGE_BUS", "")
	t.Setenv("JRUN_PROJECT_ID", "")
	t.Setenv("JRUN_TASK_ID", "")
	t.Setenv("JRUN_ID", "")

	root := t.TempDir()
	busPath := filepath.Join(root, "MESSAGE-BUS.md")
	if err := os.WriteFile(busPath, []byte("# bus\n[2026-02-01 10:00:00] FACT: bootstrap\n"), 0o644); err != nil {
		t.Fatalf("write legacy bus: %v", err)
	}
	t.Chdir(root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"bus", "post", "--type", "FACT", "--body", "auto discovered post"})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus post failed: %v", runErr)
	}
	if !strings.Contains(output, "msg_id:") {
		t.Fatalf("expected msg_id output, got %q", output)
	}

	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("open bus: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected at least one parsed message")
	}

	last := msgs[len(msgs)-1]
	if last.Body != "auto discovered post" {
		t.Fatalf("last body=%q, want %q", last.Body, "auto discovered post")
	}
	if last.Type != "FACT" {
		t.Fatalf("last type=%q, want FACT", last.Type)
	}
	if last.ProjectID != filepath.Base(root) {
		t.Fatalf("last project_id=%q, want %q", last.ProjectID, filepath.Base(root))
	}
}
