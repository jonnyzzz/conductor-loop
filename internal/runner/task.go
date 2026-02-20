package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/pkg/errors"
)

const restartPrefix = "Continue working on the following:\n\n"

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
// It validates the agent CLI binary at startup before entering the loop.
func RunTask(projectID, taskID string, opts TaskOptions) error {
	agentType := strings.ToLower(strings.TrimSpace(opts.Agent))
	if agentType != "" {
		if err := ValidateAgent(context.Background(), agentType); err != nil {
			return errors.Wrap(err, "validate agent")
		}
	}

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

	taskMDPath := filepath.Join(taskDir, "TASK.md")
	prompt := strings.TrimSpace(opts.Prompt)

	taskPrompt, err := readFileTrimmed(taskMDPath)
	if err != nil {
		if !os.IsNotExist(errors.Cause(err)) && !os.IsNotExist(err) {
			return errors.Wrap(err, "read TASK.md")
		}
		// TASK.md doesn't exist
		if prompt == "" {
			return errors.New("TASK.md not found and no prompt provided")
		}
		// Write the provided prompt to TASK.md for future restarts
		if writeErr := os.WriteFile(taskMDPath, []byte(prompt+"\n"), 0o644); writeErr != nil {
			return errors.Wrap(writeErr, "write TASK.md")
		}
	} else {
		// TASK.md exists — use it if no explicit prompt given
		if prompt == "" {
			prompt = taskPrompt
		}
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
		jobPrompt := prompt
		if attempt > 0 {
			jobPrompt = restartPrefix + prompt
		}
		jobOpts := JobOptions{
			RootDir:        opts.RootDir,
			ConfigPath:     opts.ConfigPath,
			Agent:          opts.Agent,
			Prompt:         jobPrompt,
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
