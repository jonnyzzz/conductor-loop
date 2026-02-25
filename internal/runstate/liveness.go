// Package runstate provides run metadata liveness reconciliation helpers.
package runstate

import (
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const staleRunExitCode = -1

// ReadRunInfo loads run-info.yaml and reconciles stale running states when
// the recorded PID/PGID is no longer alive.
func ReadRunInfo(path string) (*storage.RunInfo, error) {
	return ReadRunInfoWithClock(path, time.Now)
}

// ReadRunInfoWithClock behaves like ReadRunInfo, but allows injecting time in tests.
func ReadRunInfoWithClock(path string, now func() time.Time) (*storage.RunInfo, error) {
	if now == nil {
		now = time.Now
	}

	info, err := storage.ReadRunInfo(path)
	if err != nil {
		if stderrors.Is(err, os.ErrNotExist) {
			return synthesizeRunInfo(path), nil
		}
		return nil, err
	}
	if strings.TrimSpace(info.Status) == "" {
		info.Status = storage.StatusUnknown
	}

	if !shouldCheckLiveness(info) && !shouldPromoteDoneCompletion(info, path) {
		return info, nil
	}

	changed := false
	if err := storage.UpdateRunInfo(path, func(current *storage.RunInfo) error {
		if shouldCheckLiveness(current) {
			if runAlive(current) {
				return nil
			}
			if hasTaskDoneMarker(path) {
				markCompletedByDone(current, now)
			} else {
				current.Status = storage.StatusFailed
				if current.EndTime.IsZero() {
					current.EndTime = now().UTC()
				}
				if current.ExitCode == 0 {
					current.ExitCode = staleRunExitCode
				}
				if strings.TrimSpace(current.ErrorSummary) == "" {
					current.ErrorSummary = "reconciled stale running status: process is not alive"
				}
			}
			changed = true
			return nil
		}
		if shouldPromoteDoneCompletion(current, path) {
			markCompletedByDone(current, now)
			changed = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if !changed {
		return info, nil
	}
	return storage.ReadRunInfo(path)
}

func shouldPromoteDoneCompletion(info *storage.RunInfo, runInfoPath string) bool {
	if info == nil || !hasTaskDoneMarker(runInfoPath) {
		return false
	}
	status := strings.TrimSpace(info.Status)
	switch status {
	case storage.StatusCompleted:
		return false
	case storage.StatusRunning:
		return !runAlive(info)
	case storage.StatusFailed:
		// Heal previously reconciled stale failures when DONE exists.
		return info.ExitCode == staleRunExitCode
	default:
		return false
	}
}

func markCompletedByDone(info *storage.RunInfo, now func() time.Time) {
	if info == nil {
		return
	}
	info.Status = storage.StatusCompleted
	if info.EndTime.IsZero() {
		info.EndTime = now().UTC()
	}
	if info.ExitCode < 0 {
		info.ExitCode = 0
	}
	const doneSummary = "reconciled stale running status: task DONE marker is present"
	switch strings.TrimSpace(info.ErrorSummary) {
	case "", "reconciled stale running status: process is not alive":
		info.ErrorSummary = doneSummary
	}
}

func hasTaskDoneMarker(runInfoPath string) bool {
	taskDir, ok := taskDirFromRunInfoPath(runInfoPath)
	if !ok {
		return false
	}
	donePath := filepath.Join(taskDir, "DONE")
	info, err := os.Stat(donePath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func taskDirFromRunInfoPath(runInfoPath string) (string, bool) {
	runDir := filepath.Dir(strings.TrimSpace(runInfoPath))
	if runDir == "." || runDir == "" {
		return "", false
	}
	runsDir := filepath.Dir(runDir)
	if filepath.Base(runsDir) != "runs" {
		return "", false
	}
	taskDir := filepath.Dir(runsDir)
	if taskDir == "." || taskDir == "" || taskDir == string(filepath.Separator) {
		return "", false
	}
	return taskDir, true
}

func shouldCheckLiveness(info *storage.RunInfo) bool {
	if info == nil {
		return false
	}
	if strings.TrimSpace(info.Status) != storage.StatusRunning {
		return false
	}
	if info.PID == 0 && info.PGID == 0 {
		return false
	}
	// When PID/PGID is unknown, keep the persisted status unchanged.
	return info.PID > 0 || info.PGID > 0
}

func synthesizeRunInfo(path string) *storage.RunInfo {
	runID := strings.TrimSpace(filepath.Base(filepath.Dir(path)))
	if runID == "." || runID == string(filepath.Separator) {
		runID = ""
	}
	projectID, taskID := storage.RunScopeFromRunInfoPath(path)
	return &storage.RunInfo{
		RunID:     runID,
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    storage.StatusUnknown,
		Version:   0,
	}
}

func runAlive(info *storage.RunInfo) bool {
	if info == nil {
		return false
	}
	if info.PID > 0 && runner.IsProcessAlive(info.PID) {
		return true
	}
	if info.PGID > 0 {
		alive, err := runner.IsProcessGroupAlive(info.PGID)
		if err == nil && alive {
			return true
		}
	}
	return false
}
