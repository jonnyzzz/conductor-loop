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
