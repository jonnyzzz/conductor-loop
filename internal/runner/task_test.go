package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
)

func TestRunTaskDone(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunTask("project", "task", TaskOptions{RootDir: root, Agent: "codex"}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}
}

func TestRunTaskMissingPrompt(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := RunTask("project", "task", TaskOptions{RootDir: root, Agent: "codex"}); err == nil {
		t.Fatalf("expected error for missing TASK.md")
	}
}

func TestRunTask_CreatesTaskMD(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	// Create DONE so the loop exits immediately without running the agent
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	prompt := "do the thing"
	if err := RunTask("project", "task", TaskOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  prompt,
	}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if strings.TrimSpace(string(data)) != prompt {
		t.Fatalf("TASK.md content = %q, want %q", strings.TrimSpace(string(data)), prompt)
	}
}

func TestRunTask_DoesNotOverwriteExistingTaskMD(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	original := "existing prompt\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	// Create DONE so the loop exits immediately without running the agent.
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunTask("project", "task", TaskOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  "new prompt",
	}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if string(data) != original {
		t.Fatalf("TASK.md was overwritten: got %q, want %q", string(data), original)
	}
}

func TestRunTask_WithPromptFile(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	// Create DONE so the loop exits immediately without running the agent
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	promptFile := filepath.Join(t.TempDir(), "prompt.md")
	promptContent := "prompt from file"
	if err := os.WriteFile(promptFile, []byte(promptContent), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	if err := RunTask("project", "task", TaskOptions{
		RootDir:    root,
		Agent:      "codex",
		PromptPath: promptFile,
	}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if strings.TrimSpace(string(data)) != promptContent {
		t.Fatalf("TASK.md content = %q, want %q", strings.TrimSpace(string(data)), promptContent)
	}
}

func TestRunTask_UsesExistingTaskMD(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("from file"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// No prompt provided — should use TASK.md content and succeed
	if err := RunTask("project", "task", TaskOptions{
		RootDir: root,
		Agent:   "codex",
	}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}
}

func TestRunTask_FailsWithoutTaskMDAndPrompt(t *testing.T) {
	root := t.TempDir()
	// No TASK.md and no prompt
	if err := RunTask("project", "task", TaskOptions{
		RootDir: root,
		Agent:   "codex",
	}); err == nil {
		t.Fatalf("expected error for missing TASK.md and no prompt")
	}
}

func TestRunTask_RestartPrefixOnSecondAttempt(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("base prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFailingCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := RunTask("project", "task", TaskOptions{
		RootDir:        root,
		Agent:          "codex",
		MaxRestarts:    2,
		MaxRestartsSet: true,
		RestartDelay:   time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected max restarts error")
	}

	// Collect prompt.md files from runs directory; os.ReadDir returns entries sorted by name
	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	var promptFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		p := filepath.Join(runsDir, entry.Name(), "prompt.md")
		if _, statErr := os.Stat(p); statErr == nil {
			promptFiles = append(promptFiles, p)
		}
	}
	if len(promptFiles) < 2 {
		t.Fatalf("expected at least 2 prompt.md files, got %d", len(promptFiles))
	}

	firstPrompt, err := os.ReadFile(promptFiles[0])
	if err != nil {
		t.Fatalf("read first prompt: %v", err)
	}
	secondPrompt, err := os.ReadFile(promptFiles[1])
	if err != nil {
		t.Fatalf("read second prompt: %v", err)
	}

	if strings.Contains(string(firstPrompt), restartPrefix) {
		t.Fatalf("first attempt should not have restart prefix, got:\n%s", firstPrompt)
	}
	if !strings.Contains(string(secondPrompt), restartPrefix) {
		t.Fatalf("second attempt should have restart prefix, got:\n%s", secondPrompt)
	}
}

func TestRunTaskMaxRestarts(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := RunTask("project", "task", TaskOptions{
		RootDir:        root,
		Agent:          "codex",
		MaxRestarts:    1,
		MaxRestartsSet: true,
		RestartDelay:   10 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected max restarts error")
	}
}

func TestRunTask_PersistsDependsOn(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "project", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunTask("project", "task", TaskOptions{
		RootDir:   root,
		Agent:     "codex",
		DependsOn: []string{"task-a", "task-b"},
	}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}

	dependsOn, err := taskdeps.ReadDependsOn(taskDir)
	if err != nil {
		t.Fatalf("ReadDependsOn: %v", err)
	}
	if len(dependsOn) != 2 || dependsOn[0] != "task-a" || dependsOn[1] != "task-b" {
		t.Fatalf("depends_on=%v, want [task-a task-b]", dependsOn)
	}
}

func TestRunTask_RejectsDependencyCycle(t *testing.T) {
	root := t.TempDir()

	taskADir := filepath.Join(root, "project", "task-a")
	if err := os.MkdirAll(taskADir, 0o755); err != nil {
		t.Fatalf("mkdir task-a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskADir, "TASK.md"), []byte("task a"), 0o644); err != nil {
		t.Fatalf("write task-a TASK.md: %v", err)
	}
	if err := taskdeps.WriteDependsOn(taskADir, []string{"task-b"}); err != nil {
		t.Fatalf("WriteDependsOn(task-a): %v", err)
	}

	taskBDir := filepath.Join(root, "project", "task-b")
	if err := os.MkdirAll(taskBDir, 0o755); err != nil {
		t.Fatalf("mkdir task-b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskBDir, "TASK.md"), []byte("task b"), 0o644); err != nil {
		t.Fatalf("write task-b TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskBDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := RunTask("project", "task-b", TaskOptions{
		RootDir:   root,
		Agent:     "codex",
		DependsOn: []string{"task-a"},
	})
	if err == nil {
		t.Fatalf("expected dependency cycle error")
	}
	if !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("expected dependency cycle error, got: %v", err)
	}
}

func TestRunTask_WaitsForDependencies(t *testing.T) {
	root := t.TempDir()

	taskMainDir := filepath.Join(root, "project", "task-main")
	taskDepDir := filepath.Join(root, "project", "task-dep")
	if err := os.MkdirAll(taskMainDir, 0o755); err != nil {
		t.Fatalf("mkdir task-main: %v", err)
	}
	if err := os.MkdirAll(taskDepDir, 0o755); err != nil {
		t.Fatalf("mkdir task-dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskMainDir, "TASK.md"), []byte("main"), 0o644); err != nil {
		t.Fatalf("write task-main TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDepDir, "TASK.md"), []byte("dep"), 0o644); err != nil {
		t.Fatalf("write task-dep TASK.md: %v", err)
	}

	binDir := t.TempDir()
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	done := make(chan error, 1)
	go func() {
		done <- RunTask("project", "task-main", TaskOptions{
			RootDir:                 root,
			Agent:                   "codex",
			DependsOn:               []string{"task-dep"},
			DependencyPollInterval:  20 * time.Millisecond,
		})
	}()

	select {
	case err := <-done:
		t.Fatalf("RunTask returned early while dependency unresolved: %v", err)
	case <-time.After(150 * time.Millisecond):
		// expected: task is blocked
	}

	if err := os.WriteFile(filepath.Join(taskDepDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write task-dep DONE: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskMainDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write task-main DONE: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunTask returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("RunTask did not finish after dependencies were resolved")
	}
}
