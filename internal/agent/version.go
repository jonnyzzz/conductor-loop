package agent

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// DetectCLIVersion runs "<command> --version" and returns the trimmed output.
func DetectCLIVersion(ctx context.Context, command string) (string, error) {
	clean := strings.TrimSpace(command)
	if clean == "" {
		return "", errors.New("command is empty")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, clean, "--version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "run %s --version", clean)
	}

	version := strings.TrimSpace(stdout.String())
	if version == "" {
		version = strings.TrimSpace(stderr.String())
	}
	return version, nil
}
