package runner

import (
	"os/exec"

	"github.com/pkg/errors"
)

// ProcessGroupID returns the process group id for the given pid.
func ProcessGroupID(pid int) (int, error) {
	if pid <= 0 {
		return 0, errors.New("pid is invalid")
	}
	return getProcessGroupID(pid)
}

func configureProcessGroup(cmd *exec.Cmd) {
	applyProcessGroup(cmd)
}
