//go:build windows

package runner

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// TerminateProcessGroup terminates the process group on Windows.
// If a Job Object was registered for the given PID (via RegisterWindowsJobObject),
// TerminateJobObject is used to kill the entire process tree. Otherwise, falls
// back to single-PID termination.
func TerminateProcessGroup(pgid int) error {
	if pgid <= 0 {
		return errors.New("pgid is invalid")
	}
	// Try Job Object first (whole process tree).
	if ok, err := terminateByJob(pgid); ok {
		return err
	}
	// Fallback: single PID kill.
	proc, err := os.FindProcess(pgid)
	if err != nil {
		return errors.Wrap(err, "find process")
	}
	if err := proc.Kill(); err != nil {
		return errors.Wrap(err, "kill process")
	}
	return nil
}

// KillProcessGroup forcefully terminates the process group on Windows.
// On Windows there is no distinction between SIGTERM and SIGKILL;
// this is equivalent to TerminateProcessGroup.
func KillProcessGroup(pgid int) error {
	return TerminateProcessGroup(pgid)
}

// IsProcessAlive returns true if the process with the given PID is still running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	const processQueryLimitedInformation = 0x1000
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck
	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	const stillActive = 259 // STILL_ACTIVE
	return exitCode == stillActive
}
