package agent

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ProcessOptions controls process spawn behavior.
type ProcessOptions struct {
	Dir string
	Env []string
}

// SpawnProcess starts a detached process with stdio attached.
func SpawnProcess(command string, args []string, stdin io.Reader, stdout, stderr io.Writer) (*exec.Cmd, error) {
	return SpawnProcessWithOptions(command, args, stdin, stdout, stderr, ProcessOptions{})
}

// SpawnProcessWithOptions starts a detached process with stdio attached and optional settings.
func SpawnProcessWithOptions(command string, args []string, stdin io.Reader, stdout, stderr io.Writer, opts ProcessOptions) (*exec.Cmd, error) {
	clean := strings.TrimSpace(command)
	if clean == "" {
		return nil, errors.New("command is empty")
	}
	cmd := exec.Command(clean, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if dir := strings.TrimSpace(opts.Dir); dir != "" {
		cmd.Dir = dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = opts.Env
	}
	applyProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "start process")
	}
	return cmd, nil
}

// CreateOutputMD ensures output.md exists, copying from fallback when needed.
func CreateOutputMD(runDir, fallback string) (string, error) {
	cleanRunDir := filepath.Clean(strings.TrimSpace(runDir))
	if cleanRunDir == "." || cleanRunDir == "" {
		return "", errors.New("run directory is empty")
	}
	if info, err := os.Stat(cleanRunDir); err != nil {
		return "", errors.Wrap(err, "stat run directory")
	} else if !info.IsDir() {
		return "", errors.New("run directory is not a directory")
	}

	outputPath := filepath.Join(cleanRunDir, "output.md")
	if info, err := os.Stat(outputPath); err == nil {
		if info.IsDir() {
			return "", errors.New("output.md is a directory")
		}
		return outputPath, nil
	} else if !os.IsNotExist(err) {
		return "", errors.Wrap(err, "stat output.md")
	}

	fallbackPath := strings.TrimSpace(fallback)
	if fallbackPath == "" {
		fallbackPath = filepath.Join(cleanRunDir, "agent-stdout.txt")
	} else if !filepath.IsAbs(fallbackPath) {
		fallbackPath = filepath.Join(cleanRunDir, fallbackPath)
	}

	input, err := os.Open(fallbackPath)
	if err != nil {
		return "", errors.Wrap(err, "open fallback output")
	}

	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		_ = input.Close()
		return "", errors.Wrap(err, "create output.md")
	}
	if _, err := io.Copy(outputFile, input); err != nil {
		_ = outputFile.Close()
		_ = input.Close()
		return "", errors.Wrap(err, "copy output.md")
	}
	if err := outputFile.Sync(); err != nil {
		_ = outputFile.Close()
		_ = input.Close()
		return "", errors.Wrap(err, "sync output.md")
	}
	if err := outputFile.Close(); err != nil {
		_ = input.Close()
		return "", errors.Wrap(err, "close output.md")
	}
	if err := input.Close(); err != nil {
		return "", errors.Wrap(err, "close fallback output")
	}
	return outputPath, nil
}

var _ = CreateOutputMD
