// Package runstate provides run metadata liveness reconciliation helpers.
package runstate

import (
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
		return nil, err
	}

	if !shouldCheckLiveness(info) {
		return info, nil
	}

	changed := false
	if err := storage.UpdateRunInfo(path, func(current *storage.RunInfo) error {
		if !shouldCheckLiveness(current) {
			return nil
		}
		if runAlive(current) {
			return nil
		}
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
		changed = true
		return nil
	}); err != nil {
		return nil, err
	}

	if !changed {
		return info, nil
	}
	return storage.ReadRunInfo(path)
}

func shouldCheckLiveness(info *storage.RunInfo) bool {
	if info == nil {
		return false
	}
	if strings.TrimSpace(info.Status) != storage.StatusRunning {
		return false
	}
	// When PID/PGID is unknown, keep the persisted status unchanged.
	return info.PID > 0 || info.PGID > 0
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
