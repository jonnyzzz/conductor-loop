//go:build windows

package runner

import (
	"os"

	"github.com/pkg/errors"
)

// TerminateProcessGroup terminates the process group on Windows.
func TerminateProcessGroup(pgid int) error {
	if pgid <= 0 {
		return errors.New("pgid is invalid")
	}
	proc, err := os.FindProcess(pgid)
	if err != nil {
		return errors.Wrap(err, "find process")
	}
	if err := proc.Kill(); err != nil {
		return errors.Wrap(err, "kill process")
	}
	return nil
}
