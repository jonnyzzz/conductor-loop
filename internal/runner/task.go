package runner

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/pkg/errors"
)

// TaskOptions controls execution for the run-agent task command.
type TaskOptions struct {
	RootDir        string
	ConfigPath     string
	Agent          string
	Prompt         string
	WorkingDir     string
	MessageBusPath string
	MaxRestarts    int
	WaitTimeout    time.Duration
	PollInterval   time.Duration
	RestartDelay   time.Duration
	Environment    map[string]string
}

// RunTask starts the root agent and enforces the Ralph loop.
func RunTask(projectID, taskID string, opts TaskOptions) error {
	rootDir, err := resolveRootDir(opts.RootDir)
	if err != nil {
		return err
	}
	taskDir, err := resolveTaskDir(rootDir, projectID, taskID)
	if err != nil {
		return err
	}
	if err := ensureDir(taskDir); err != nil {
		return errors.Wrap(err, "ensure task dir")
	}

	taskPrompt, err := readFileTrimmed(filepath.Join(taskDir, "TASK.md"))
	if err != nil {
		return errors.Wrap(err, "read TASK.md")
	}
	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		prompt = taskPrompt
	}

	busPath := strings.TrimSpace(opts.MessageBusPath)
	if busPath == "" {
		busPath = filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return errors.Wrap(err, "new message bus")
	}

	previousRunID := ""
	runnerFn := func(ctx context.Context, attempt int) error {
		jobOpts := JobOptions{
			RootDir:        opts.RootDir,
			ConfigPath:     opts.ConfigPath,
			Agent:          opts.Agent,
			Prompt:         prompt,
			WorkingDir:     opts.WorkingDir,
			MessageBusPath: busPath,
			PreviousRunID:  previousRunID,
			Environment:    opts.Environment,
		}
		info, err := runJob(projectID, taskID, jobOpts)
		if info != nil {
			previousRunID = info.RunID
		}
		return err
	}

	options := []RalphOption{
		WithProjectTask(projectID, taskID),
		WithRootRunner(runnerFn),
	}
	if opts.MaxRestarts > 0 {
		options = append(options, WithMaxRestarts(opts.MaxRestarts))
	}
	if opts.WaitTimeout > 0 {
		options = append(options, WithWaitTimeout(opts.WaitTimeout))
	}
	if opts.PollInterval > 0 {
		options = append(options, WithPollInterval(opts.PollInterval))
	}
	if opts.RestartDelay > 0 {
		options = append(options, WithRestartDelay(opts.RestartDelay))
	}

	loop, err := NewRalphLoop(taskDir, bus, options...)
	if err != nil {
		return err
	}
	return loop.Run(context.Background())
}
