// Package runner manages agent process execution for the orchestration subsystem.
package runner

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ProcessManager spawns agent processes within a run directory.
type ProcessManager struct {
	runDir string
}

// SpawnOptions controls how a process is started.
type SpawnOptions struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Process represents a spawned agent process and its metadata.
type Process struct {
	Cmd        *exec.Cmd
	PID        int
	PGID       int
	StdoutPath string
	StderrPath string

	capture *StdioCapture
}

// NewProcessManager creates a ProcessManager for the given run directory.
func NewProcessManager(runDir string) (*ProcessManager, error) {
	clean := filepath.Clean(strings.TrimSpace(runDir))
	if clean == "." || clean == "" {
		return nil, errors.New("run directory is empty")
	}
	return &ProcessManager{runDir: clean}, nil
}

// SpawnAgent starts a detached process with redirected stdio.
func (pm *ProcessManager) SpawnAgent(ctx context.Context, agentType string, opts SpawnOptions) (*Process, error) {
	if pm == nil {
		return nil, errors.New("process manager is nil")
	}
	if strings.TrimSpace(agentType) == "" {
		return nil, errors.New("agent type is empty")
	}
	command := strings.TrimSpace(opts.Command)
	if command == "" {
		return nil, errors.New("command is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, command, opts.Args...)
	if dir := strings.TrimSpace(opts.Dir); dir != "" {
		cmd.Dir = dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = opts.Env
	}

	capture, err := OpenStdio(pm.runDir, opts.Stdout, opts.Stderr)
	if err != nil {
		return nil, errors.Wrap(err, "open stdio")
	}
	cmd.Stdout = capture.Stdout
	cmd.Stderr = capture.Stderr

	stdin := opts.Stdin
	var stdinFile *os.File
	if stdin == nil {
		stdinFile, err = os.OpenFile(os.DevNull, os.O_RDONLY, 0o644)
		if err != nil {
			_ = capture.Close()
			return nil, errors.Wrap(err, "open stdin")
		}
		stdin = stdinFile
	}
	cmd.Stdin = stdin

	configureProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		if stdinFile != nil {
			_ = stdinFile.Close()
		}
		_ = capture.Close()
		return nil, errors.Wrap(err, "start process")
	}
	if stdinFile != nil {
		_ = stdinFile.Close()
	}

	pid := cmd.Process.Pid
	pgid, err := ProcessGroupID(pid)
	if err != nil {
		_ = capture.Close()
		return nil, errors.Wrap(err, "get process group id")
	}

	return &Process{
		Cmd:        cmd,
		PID:        pid,
		PGID:       pgid,
		StdoutPath: capture.StdoutPath,
		StderrPath: capture.StderrPath,
		capture:    capture,
	}, nil
}

// Wait waits for the process to exit and closes stdout/stderr files.
func (p *Process) Wait() error {
	if p == nil || p.Cmd == nil {
		return errors.New("process is nil")
	}
	if err := p.Cmd.Wait(); err != nil {
		_ = p.closeCapture()
		return errors.Wrap(err, "wait process")
	}
	return p.closeCapture()
}

func (p *Process) closeCapture() error {
	if p == nil || p.capture == nil {
		return nil
	}
	if err := p.capture.Close(); err != nil {
		return errors.Wrap(err, "close stdio")
	}
	return nil
}
