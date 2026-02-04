package messagebus

import (
	stderrors "errors"
	"os"
	"time"

	"github.com/pkg/errors"
)

const lockPollInterval = 10 * time.Millisecond

// ErrLockTimeout indicates the lock was not acquired within the timeout.
var ErrLockTimeout = stderrors.New("lock timeout")

// LockExclusive acquires an exclusive lock with the specified timeout.
func LockExclusive(file *os.File, timeout time.Duration) error {
	return flockExclusive(file, timeout)
}

// Unlock releases a previously acquired lock.
func Unlock(file *os.File) error {
	if file == nil {
		return errors.New("lock file is nil")
	}
	return unlockFile(file)
}

func flockExclusive(file *os.File, timeout time.Duration) error {
	if file == nil {
		return errors.New("lock file is nil")
	}
	if timeout <= 0 {
		return errors.New("lock timeout must be positive")
	}
	deadline := time.Now().Add(timeout)
	for {
		locked, err := tryFlockExclusive(file)
		if err != nil {
			return errors.Wrap(err, "flock")
		}
		if locked {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrLockTimeout
		}
		time.Sleep(lockPollInterval)
	}
}
