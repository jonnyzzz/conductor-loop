//go:build !windows

package runner

import (
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

func applyProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}

func getProcessGroupID(pid int) (int, error) {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return 0, errors.Wrap(err, "get pgid")
	}
	return pgid, nil
}
