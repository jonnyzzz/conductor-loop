//go:build windows

package api

import "github.com/pkg/errors"

func defaultSelfUpdateReexec(_ string, _ []string, _ []string) error {
	return errors.New("in-place self-update handoff is not supported on windows")
}
