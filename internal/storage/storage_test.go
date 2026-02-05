package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteRunInfoValidation(t *testing.T) {
	if err := WriteRunInfo("", &RunInfo{}); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if err := WriteRunInfo("ignored", nil); err == nil {
		t.Fatalf("expected error for nil run-info")
	}
}

func TestReadRunInfoValidation(t *testing.T) {
	if _, err := ReadRunInfo(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestUpdateRunInfoValidation(t *testing.T) {
	if err := UpdateRunInfo("ignored", nil); err == nil {
		t.Fatalf("expected error for nil update")
	}
}

func TestRunInfoRoundtrip(t *testing.T) {
	info := &RunInfo{
		RunID:       "run-1",
		ParentRunID: "parent",
		ProjectID:   "project",
		TaskID:      "task",
		AgentType:   "codex",
		PID:         123,
		PGID:        456,
		StartTime:   time.Date(2026, 2, 4, 8, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 2, 4, 8, 5, 0, 0, time.UTC),
		ExitCode:    0,
		Status:      StatusCompleted,
		CWD:         "/tmp",
		PromptPath:  "/tmp/prompt.md",
		OutputPath:  "/tmp/output.md",
		StdoutPath:  "/tmp/stdout.txt",
		StderrPath:  "/tmp/stderr.txt",
		CommandLine: "echo hi",
	}
	path := filepath.Join(t.TempDir(), "run-info.yaml")
	if err := WriteRunInfo(path, info); err != nil {
		t.Fatalf("WriteRunInfo: %v", err)
	}
	got, err := ReadRunInfo(path)
	if err != nil {
		t.Fatalf("ReadRunInfo: %v", err)
	}
	if got.RunID != info.RunID || got.ProjectID != info.ProjectID || got.Status != info.Status {
		t.Fatalf("unexpected run-info: %+v", got)
	}
}

func TestNewStorageValidation(t *testing.T) {
	if _, err := NewStorage(""); err == nil {
		t.Fatalf("expected error for empty root")
	}
}

func TestFileStorageCreateAndGetRun(t *testing.T) {
	root := t.TempDir()
	st, err := NewStorage(root)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	st.now = func() time.Time { return time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC) }
	st.pid = func() int { return 42 }

	info, err := st.CreateRun("project", "task", "codex")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if info.RunID == "" || info.PID != 42 {
		t.Fatalf("unexpected run info: %+v", info)
	}

	loaded, err := st.GetRunInfo(info.RunID)
	if err != nil {
		t.Fatalf("GetRunInfo: %v", err)
	}
	if loaded.RunID != info.RunID || loaded.Status != StatusRunning {
		t.Fatalf("unexpected loaded info: %+v", loaded)
	}
}

func TestFileStorageUpdateRunStatus(t *testing.T) {
	root := t.TempDir()
	st, err := NewStorage(root)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	st.now = func() time.Time { return time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC) }
	st.pid = func() int { return 7 }

	info, err := st.CreateRun("project", "task", "claude")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := st.UpdateRunStatus(info.RunID, StatusCompleted, 0); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	loaded, err := st.GetRunInfo(info.RunID)
	if err != nil {
		t.Fatalf("GetRunInfo: %v", err)
	}
	if loaded.Status != StatusCompleted || loaded.ExitCode != 0 {
		t.Fatalf("unexpected status: %+v", loaded)
	}
}

func TestFileStorageListRuns(t *testing.T) {
	root := t.TempDir()
	st, err := NewStorage(root)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	st.now = func() time.Time { return time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC) }
	st.pid = func() int { return 1 }

	if _, err := st.CreateRun("project", "task", "codex"); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	st.pid = func() int { return 2 }
	if _, err := st.CreateRun("project", "task", "codex"); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	runs, err := st.ListRuns("project", "task")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if runs[0].RunID > runs[1].RunID {
		t.Fatalf("expected sorted run ids")
	}
}

func TestRunInfoPathErrors(t *testing.T) {
	root := t.TempDir()
	st, err := NewStorage(root)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	if _, err := st.runInfoPath("missing"); err == nil {
		t.Fatalf("expected error for missing run")
	}

	// create two runs with same id to force ambiguous
	runID := "dup-run"
	for _, project := range []string{"p1", "p2"} {
		path := filepath.Join(root, project, "task", "runs", runID)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		info := &RunInfo{RunID: runID, ProjectID: project, TaskID: "task", Status: StatusRunning}
		if err := WriteRunInfo(filepath.Join(path, "run-info.yaml"), info); err != nil {
			t.Fatalf("write run-info: %v", err)
		}
	}
	if _, err := st.runInfoPath(runID); err == nil {
		t.Fatalf("expected error for duplicate run ids")
	}
}

func TestFileStorageValidationErrors(t *testing.T) {
	st, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	if _, err := st.CreateRun("", "task", "agent"); err == nil {
		t.Fatalf("expected error for empty project")
	}
	if _, err := st.CreateRun("project", "", "agent"); err == nil {
		t.Fatalf("expected error for empty task")
	}
	if _, err := st.CreateRun("project", "task", ""); err == nil {
		t.Fatalf("expected error for empty agent")
	}
	if _, err := st.GetRunInfo(""); err == nil {
		t.Fatalf("expected error for empty run id")
	}
	if _, err := st.ListRuns("", "task"); err == nil {
		t.Fatalf("expected error for empty project id")
	}
	if _, err := st.ListRuns("project", ""); err == nil {
		t.Fatalf("expected error for empty task id")
	}
}

func TestWriteFileAtomicErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "run-info.yaml")
	if err := writeFileAtomic(path, []byte("data")); err == nil {
		t.Fatalf("expected error for missing directory")
	}
}

func TestFileStorageErrors(t *testing.T) {
	root := t.TempDir()
	st, err := NewStorage(root)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	if _, err := st.GetRunInfo("missing"); err == nil {
		t.Fatalf("expected error for missing run info")
	}
	if _, err := st.ListRuns("project", "task"); err == nil {
		t.Fatalf("expected error for missing runs dir")
	}
	if err := st.UpdateRunStatus("", StatusCompleted, 0); err == nil {
		t.Fatalf("expected error for empty run id")
	}
	if err := st.UpdateRunStatus("run", "", 0); err == nil {
		t.Fatalf("expected error for empty status")
	}
}

func TestReadRunInfoInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run-info.yaml")
	if err := os.WriteFile(path, []byte("invalid: ["), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ReadRunInfo(path); err == nil {
		t.Fatalf("expected error for invalid yaml")
	}
}

func TestUpdateRunInfoApplyError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run-info.yaml")
	info := &RunInfo{RunID: "run", ProjectID: "project", TaskID: "task", AgentType: "codex", StartTime: time.Now().UTC(), Status: StatusRunning}
	if err := WriteRunInfo(path, info); err != nil {
		t.Fatalf("WriteRunInfo: %v", err)
	}
	err := UpdateRunInfo(path, func(info *RunInfo) error {
		return os.ErrInvalid
	})
	if err == nil {
		t.Fatalf("expected error from update")
	}
}
