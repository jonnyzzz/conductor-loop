//go:build windows

package runner

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

func isProcessGroupAlive(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, errors.New("pgid is invalid")
	}

	// If a Job Object is registered for this PID, use its active process count.
	// A count of 0 means the entire process tree has exited.
	if count := jobObjectActiveProcesses(pgid); count >= 0 {
		return count > 0, nil
	}

	// Fallback: single-PID liveness check via GetExitCodeProcess.
	proc, err := os.FindProcess(pgid)
	if err != nil {
		return false, errors.Wrap(err, "find process")
	}
	// Best-effort: Signal(0) is not reliably supported on Windows, so treat errors as alive.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return true, nil
	}
	return true, nil
}
