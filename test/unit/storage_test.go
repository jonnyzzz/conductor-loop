package unit_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRunInfoSerialization(t *testing.T) {
	t.Parallel()

	info := &storage.RunInfo{
		RunID:     "run-123",
		ProjectID: "project",
		TaskID:    "task-001",
		AgentType: "codex",
		PID:       100,
		PGID:      200,
		StartTime: time.Date(2026, 2, 4, 10, 11, 12, 0, time.UTC),
		EndTime:   time.Date(2026, 2, 4, 10, 12, 13, 0, time.UTC),
		ExitCode:  2,
		Status:    storage.StatusFailed,
	}

	path := filepath.Join(t.TempDir(), "run-info.yaml")
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	got, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	assertRunInfoEqual(t, info, got)
}

func TestAtomicWrite(t *testing.T) {
	infoA := &storage.RunInfo{
		RunID:     "run-a",
		ProjectID: "project",
		TaskID:    "task-001",
		AgentType: "codex",
		PID:       10,
		PGID:      20,
		StartTime: time.Date(2026, 2, 4, 11, 0, 0, 0, time.UTC),
		ExitCode:  -1,
		Status:    storage.StatusRunning,
	}
	infoB := &storage.RunInfo{
		RunID:     "run-b",
		ProjectID: "project",
		TaskID:    "task-001",
		AgentType: "codex",
		PID:       11,
		PGID:      21,
		StartTime: time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 2, 4, 12, 1, 0, 0, time.UTC),
		ExitCode:  0,
		Status:    storage.StatusCompleted,
	}

	path := filepath.Join(t.TempDir(), "run-info.yaml")
	if err := storage.WriteRunInfo(path, infoA); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	if err := storage.WriteRunInfo(path, infoB); err != nil {
		t.Fatalf("rewrite run-info: %v", err)
	}
	got, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if got.RunID != infoB.RunID {
		t.Fatalf("expected run id %q, got %q", infoB.RunID, got.RunID)
	}
}

func TestConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run-info.yaml")

	const (
		writers    = 10
		iterations = 100
	)

	errCh := make(chan error, writers*iterations)
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				info := &storage.RunInfo{
					RunID:     fmt.Sprintf("run-%d-%d", w, i),
					ProjectID: "project",
					TaskID:    "task-001",
					AgentType: "codex",
					PID:       100 + w,
					PGID:      200 + w,
					StartTime: time.Date(2026, 2, 4, 13, 0, 0, i, time.UTC),
					ExitCode:  -1,
					Status:    storage.StatusRunning,
				}
				if err := storage.WriteRunInfo(path, info); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent write error: %v", err)
		}
	}

	got, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read run-info after concurrent writes: %v", err)
	}
	if !strings.HasPrefix(got.RunID, "run-") {
		t.Fatalf("unexpected run id after concurrent writes: %q", got.RunID)
	}
}

func TestUpdateRunInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run-info.yaml")
	info := &storage.RunInfo{
		RunID:     "run-update",
		ProjectID: "project",
		TaskID:    "task-001",
		AgentType: "codex",
		PID:       10,
		PGID:      20,
		StartTime: time.Date(2026, 2, 4, 14, 0, 0, 0, time.UTC),
		ExitCode:  -1,
		Status:    storage.StatusRunning,
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	endTime := time.Date(2026, 2, 4, 14, 5, 0, 0, time.UTC)
	if err := storage.UpdateRunInfo(path, func(info *storage.RunInfo) error {
		info.Status = storage.StatusCompleted
		info.ExitCode = 0
		info.EndTime = endTime
		return nil
	}); err != nil {
		t.Fatalf("update run-info: %v", err)
	}

	got, err := storage.ReadRunInfo(path)
	if err != nil {
		t.Fatalf("read updated run-info: %v", err)
	}
	if got.Status != storage.StatusCompleted {
		t.Fatalf("expected status %q, got %q", storage.StatusCompleted, got.Status)
	}
	if got.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", got.ExitCode)
	}
	if !got.EndTime.Equal(endTime) {
		t.Fatalf("expected end time %v, got %v", endTime, got.EndTime)
	}
}

func assertRunInfoEqual(t *testing.T, want, got *storage.RunInfo) {
	t.Helper()
	if want.RunID != got.RunID {
		t.Fatalf("run id: want %q, got %q", want.RunID, got.RunID)
	}
	if want.ParentRunID != got.ParentRunID {
		t.Fatalf("parent run id: want %q, got %q", want.ParentRunID, got.ParentRunID)
	}
	if want.ProjectID != got.ProjectID {
		t.Fatalf("project id: want %q, got %q", want.ProjectID, got.ProjectID)
	}
	if want.TaskID != got.TaskID {
		t.Fatalf("task id: want %q, got %q", want.TaskID, got.TaskID)
	}
	if want.AgentType != got.AgentType {
		t.Fatalf("agent type: want %q, got %q", want.AgentType, got.AgentType)
	}
	if want.PID != got.PID {
		t.Fatalf("pid: want %d, got %d", want.PID, got.PID)
	}
	if want.PGID != got.PGID {
		t.Fatalf("pgid: want %d, got %d", want.PGID, got.PGID)
	}
	if !want.StartTime.Equal(got.StartTime) {
		t.Fatalf("start time: want %v, got %v", want.StartTime, got.StartTime)
	}
	if !want.EndTime.Equal(got.EndTime) {
		t.Fatalf("end time: want %v, got %v", want.EndTime, got.EndTime)
	}
	if want.ExitCode != got.ExitCode {
		t.Fatalf("exit code: want %d, got %d", want.ExitCode, got.ExitCode)
	}
	if want.Status != got.Status {
		t.Fatalf("status: want %q, got %q", want.Status, got.Status)
	}
}
