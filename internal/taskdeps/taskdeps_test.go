package taskdeps

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestNormalize(t *testing.T) {
	got, err := Normalize("task-main", []string{
		" task-a ",
		"task-b, task-c",
		"task-a",
		"task-c",
		"",
	})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	want := []string{"task-a", "task-b", "task-c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("depends_on=%v, want %v", got, want)
	}
}

func TestNormalizeRejectsSelfDependency(t *testing.T) {
	_, err := Normalize("task-self", []string{"task-self"})
	if err == nil {
		t.Fatalf("expected self dependency error")
	}
	if !strings.Contains(err.Error(), "cannot depend on itself") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeRejectsPathTraversal(t *testing.T) {
	_, err := Normalize("task-main", []string{"../evil"})
	if err == nil {
		t.Fatalf("expected invalid dependency error")
	}
}

func TestReadWriteDependsOn(t *testing.T) {
	taskDir := t.TempDir()
	dependsOn := []string{"task-a", "task-b"}

	if err := WriteDependsOn(taskDir, dependsOn); err != nil {
		t.Fatalf("WriteDependsOn: %v", err)
	}
	got, err := ReadDependsOn(taskDir)
	if err != nil {
		t.Fatalf("ReadDependsOn: %v", err)
	}
	if !reflect.DeepEqual(got, dependsOn) {
		t.Fatalf("depends_on=%v, want %v", got, dependsOn)
	}

	if err := WriteDependsOn(taskDir, nil); err != nil {
		t.Fatalf("WriteDependsOn(clear): %v", err)
	}
	got, err = ReadDependsOn(taskDir)
	if err != nil {
		t.Fatalf("ReadDependsOn after clear: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("depends_on=%v, want empty", got)
	}
}

func TestValidateNoCycle(t *testing.T) {
	root := t.TempDir()
	projectID := "proj"

	mustWriteTask(t, root, projectID, "task-a", []string{"task-b"})
	mustWriteTask(t, root, projectID, "task-b", []string{})

	if err := ValidateNoCycle(root, projectID, "task-c", []string{"task-a"}); err != nil {
		t.Fatalf("expected no cycle: %v", err)
	}
}

func TestValidateNoCycleDetectsLoop(t *testing.T) {
	root := t.TempDir()
	projectID := "proj"

	mustWriteTask(t, root, projectID, "task-a", []string{"task-b"})
	mustWriteTask(t, root, projectID, "task-b", []string{"task-c"})

	err := ValidateNoCycle(root, projectID, "task-c", []string{"task-a"})
	if err == nil {
		t.Fatalf("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlockedBy(t *testing.T) {
	root := t.TempDir()
	projectID := "proj"

	mustWriteTask(t, root, projectID, "task-done", nil)
	mustWriteFile(t, filepath.Join(root, projectID, "task-done", "DONE"), "")

	mustWriteTask(t, root, projectID, "task-completed", nil)
	mustWriteRun(t, root, projectID, "task-completed", "run-1", storage.StatusCompleted)

	mustWriteTask(t, root, projectID, "task-running", nil)
	mustWriteRun(t, root, projectID, "task-running", "run-2", storage.StatusRunning)

	mustWriteTask(t, root, projectID, "task-failed", nil)
	mustWriteRun(t, root, projectID, "task-failed", "run-3", storage.StatusFailed)

	blocked, err := BlockedBy(root, projectID, []string{
		"task-done",
		"task-completed",
		"task-running",
		"task-failed",
		"task-missing",
	})
	if err != nil {
		t.Fatalf("BlockedBy: %v", err)
	}
	want := []string{"task-running", "task-failed", "task-missing"}
	if !reflect.DeepEqual(blocked, want) {
		t.Fatalf("blocked_by=%v, want %v", blocked, want)
	}
}

func mustWriteTask(t *testing.T, root, projectID, taskID string, dependsOn []string) {
	t.Helper()
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	mustWriteFile(t, filepath.Join(taskDir, "TASK.md"), "prompt\n")
	if err := WriteDependsOn(taskDir, dependsOn); err != nil {
		t.Fatalf("WriteDependsOn: %v", err)
	}
}

func mustWriteRun(t *testing.T, root, projectID, taskID, runID, status string) {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	info := &storage.RunInfo{
		Version:   1,
		RunID:     runID,
		ProjectID: projectID,
		TaskID:    taskID,
		AgentType: "codex",
		StartTime: time.Now().Add(-time.Minute).UTC(),
		ExitCode:  -1,
		Status:    status,
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
		info.ExitCode = 0
		if status == storage.StatusFailed {
			info.ExitCode = 1
		}
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
