package runner

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// StdioCapture holds writers and file handles for captured output.
type StdioCapture struct {
	Stdout io.Writer
	Stderr io.Writer

	StdoutPath string
	StderrPath string

	stdoutFile *os.File
	stderrFile *os.File
}

// OpenStdio opens stdout/stderr files in the run directory.
func OpenStdio(runDir string, stdout, stderr io.Writer) (*StdioCapture, error) {
	cleanRunDir := filepath.Clean(strings.TrimSpace(runDir))
	if cleanRunDir == "." || cleanRunDir == "" {
		return nil, errors.New("run directory is empty")
	}
	if err := os.MkdirAll(cleanRunDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "create run directory")
	}

	stdoutPath := filepath.Join(cleanRunDir, "agent-stdout.txt")
	stderrPath := filepath.Join(cleanRunDir, "agent-stderr.txt")

	stdoutFile, err := openAppendFile(stdoutPath)
	if err != nil {
		return nil, errors.Wrap(err, "open stdout file")
	}
	stderrFile, err := openAppendFile(stderrPath)
	if err != nil {
		_ = stdoutFile.Close()
		return nil, errors.Wrap(err, "open stderr file")
	}

	stdoutWriter := io.Writer(stdoutFile)
	if stdout != nil {
		stdoutWriter = io.MultiWriter(stdoutFile, stdout)
	}
	stderrWriter := io.Writer(stderrFile)
	if stderr != nil {
		stderrWriter = io.MultiWriter(stderrFile, stderr)
	}

	return &StdioCapture{
		Stdout:     stdoutWriter,
		Stderr:     stderrWriter,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		stdoutFile: stdoutFile,
		stderrFile: stderrFile,
	}, nil
}

// Close closes the underlying stdout/stderr files.
func (c *StdioCapture) Close() error {
	if c == nil {
		return errors.New("stdio capture is nil")
	}
	var firstErr error
	if c.stdoutFile != nil {
		if err := c.stdoutFile.Close(); err != nil {
			firstErr = errors.Wrap(err, "close stdout file")
		}
	}
	if c.stderrFile != nil {
		if err := c.stderrFile.Close(); err != nil && firstErr == nil {
			firstErr = errors.Wrap(err, "close stderr file")
		}
	}
	return firstErr
}

func openAppendFile(path string) (*os.File, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("output path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrap(err, "create output directory")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, errors.Wrap(err, "open output file")
	}
	return file, nil
}
