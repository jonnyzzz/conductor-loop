//go:build !windows

package messagebus

import (
	"os"
	"syscall"
)

func tryFlockExclusive(file *os.File) (bool, error) {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return false, nil
	}
	return false, err
}

// tryFlockShared on Unix is a no-op: flock() is advisory, so readers always
// succeed without acquiring any lock. Returns (true, nil) immediately.
func tryFlockShared(_ *os.File) (bool, error) {
	return true, nil
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
