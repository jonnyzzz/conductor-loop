//go:build windows

// Windows file locking limitation:
//
// On Unix/macOS, flock() is advisory — readers can access files without
// acquiring locks, allowing lockless reads while writers hold exclusive locks.
// On Windows, LockFileEx uses mandatory locks that block ALL concurrent
// access to locked byte ranges, including reads. This means message bus
// polling may block when any agent holds a write lock.
//
// For full compatibility, use WSL2 on Windows. See ISSUE-002 in ISSUES.md
// and docs/user/troubleshooting.md for details.

package messagebus

import (
	"os"
	"syscall"
)

func tryFlockExclusive(file *os.File) (bool, error) {
	handle := syscall.Handle(file.Fd())
	var overlapped syscall.Overlapped
	err := syscall.LockFileEx(handle, syscall.LOCKFILE_EXCLUSIVE_LOCK|syscall.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
	if err == nil {
		return true, nil
	}
	if err == syscall.ERROR_LOCK_VIOLATION {
		return false, nil
	}
	return false, err
}

func unlockFile(file *os.File) error {
	handle := syscall.Handle(file.Fd())
	var overlapped syscall.Overlapped
	return syscall.UnlockFileEx(handle, 0, 1, 0, &overlapped)
}
