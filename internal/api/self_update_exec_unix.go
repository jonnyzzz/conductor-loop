//go:build !windows

package api

import "syscall"

func defaultSelfUpdateReexec(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}
