//go:build !windows

package runner

import (
	stderrors "errors"
	"syscall"

	"github.com/pkg/errors"
)

func isProcessGroupAlive(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, errors.New("pgid is invalid")
	}
	err := syscall.Kill(-pgid, 0)
	if err == nil {
		return true, nil
	}
	if stderrors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	if stderrors.Is(err, syscall.EPERM) {
		return true, nil
	}
	return false, errors.Wrap(err, "check process group")
}
