//go:build !windows

package runner

import (
	stderrors "errors"
	"syscall"

	"github.com/pkg/errors"
)

// TerminateProcessGroup sends a SIGTERM to the provided process group id.
func TerminateProcessGroup(pgid int) error {
	if pgid <= 0 {
		return errors.New("pgid is invalid")
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		if stderrors.Is(err, syscall.ESRCH) {
			return errors.Wrap(err, "process group not found")
		}
		return errors.Wrap(err, "terminate process group")
	}
	return nil
}

// KillProcessGroup sends a SIGKILL to the provided process group id.
func KillProcessGroup(pgid int) error {
	if pgid <= 0 {
		return errors.New("pgid is invalid")
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		if stderrors.Is(err, syscall.ESRCH) {
			return errors.Wrap(err, "process group not found")
		}
		return errors.Wrap(err, "kill process group")
	}
	return nil
}

// IsProcessAlive returns true if the process with the given PID is still running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
