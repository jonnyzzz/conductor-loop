//go:build windows

package runner

import (
	"os/exec"
	"syscall"
)

func applyProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
}

func getProcessGroupID(pid int) (int, error) {
	return pid, nil
}
