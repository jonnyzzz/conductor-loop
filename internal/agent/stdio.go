package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// OutputFiles defines file destinations for captured stdout and stderr.
type OutputFiles struct {
	StdoutPath string
	StderrPath string
}

// OutputCapture holds writers and file handles for captured output.
type OutputCapture struct {
	Stdout io.Writer
	Stderr io.Writer

	stdoutFile *os.File
	stderrFile *os.File
}

// CaptureOutput opens stdout/stderr files and returns writers that tee output.
func CaptureOutput(stdout, stderr io.Writer, files OutputFiles) (*OutputCapture, error) {
	stdoutPath := strings.TrimSpace(files.StdoutPath)
	if stdoutPath == "" {
		return nil, errors.New("stdout path is empty")
	}
	stderrPath := strings.TrimSpace(files.StderrPath)
	if stderrPath == "" {
		return nil, errors.New("stderr path is empty")
	}

	stdoutFile, err := openOutputFile(stdoutPath)
	if err != nil {
		return nil, errors.Wrap(err, "open stdout file")
	}
	stderrFile, err := openOutputFile(stderrPath)
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

	return &OutputCapture{
		Stdout:     stdoutWriter,
		Stderr:     stderrWriter,
		stdoutFile: stdoutFile,
		stderrFile: stderrFile,
	}, nil
}

// Close closes the underlying stdout/stderr files.
func (c *OutputCapture) Close() error {
	if c == nil {
		return errors.New("output capture is nil")
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

func openOutputFile(path string) (*os.File, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("output path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrap(err, "create output directory")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, errors.Wrap(err, "open output file")
	}
	return file, nil
}
