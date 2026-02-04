//go:build windows

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
