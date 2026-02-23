package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
	"github.com/pkg/errors"
)

// RestartPrefix is prepended to the prompt when restarting a task (resume mode or Ralph loop restart).
const RestartPrefix = "Continue working on the following:\n\n"

const restartPrefix = RestartPrefix
const defaultDependencyPollInterval = 2 * time.Second

// TaskOptions controls execution for the run-agent task command.
type TaskOptions struct {
	RootDir        string
	ConfigPath     string
	Agent          string
	Prompt         string
	PromptPath     string // path to a file containing the prompt
	WorkingDir     string
	MessageBusPath string
	MaxRestarts    int
	MaxRestartsSet bool
	WaitTimeout    time.Duration
	PollInterval   time.Duration
	RestartDelay   time.Duration
	Timeout        time.Duration // idle output timeout per CLI job; 0 means no limit
	Environment    map[string]string
	FirstRunDir    string // optional: pre-allocated run directory used for the first run attempt
	ResumeMode     bool   // when true, prepend restart prefix even on the first run attempt
	ConductorURL   string // e.g. "http://127.0.0.1:14355"; passed to JobOptions
	ParentRunID    string // optional: parent run ID for threaded child task linkage
	DependsOn      []string
	// DependencyPollInterval controls how often dependency status is checked while blocked.
	// Zero means a default interval is used.
	DependencyPollInterval time.Duration
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

	// Resolve prompt from file if PromptPath is set
	if path := strings.TrimSpace(opts.PromptPath); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "read prompt file")
		}
		opts.Prompt = strings.TrimSpace(string(data))
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
	obslog.Log(log.Default(), "INFO", "runner", "task_run_started",
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
		obslog.F("agent", opts.Agent),
		obslog.F("message_bus_path", busPath),
		obslog.F("resume_mode", opts.ResumeMode),
	)

	dependsOn, err := resolveTaskDependencies(rootDir, projectID, taskID, taskDir, opts.DependsOn)
	if err != nil {
		return err
	}

	if err := waitForDependencies(taskDir, rootDir, projectID, taskID, dependsOn, opts.DependencyPollInterval, bus); err != nil {
		obslog.Log(log.Default(), "ERROR", "runner", "task_dependency_wait_failed",
			obslog.F("project_id", projectID),
			obslog.F("task_id", taskID),
			obslog.F("error", err),
		)
		return err
	}

	previousRunID := ""
	runnerFn := func(ctx context.Context, attempt int) error {
		jobPrompt := prompt
		if attempt > 0 || opts.ResumeMode {
			jobPrompt = restartPrefix + prompt
		}
		jobOpts := JobOptions{
			RootDir:        opts.RootDir,
			ConfigPath:     opts.ConfigPath,
			Agent:          opts.Agent,
			Prompt:         jobPrompt,
			WorkingDir:     opts.WorkingDir,
			MessageBusPath: busPath,
			ParentRunID:    opts.ParentRunID,
			PreviousRunID:  previousRunID,
			Environment:    opts.Environment,
			Timeout:        opts.Timeout,
			ConductorURL:   opts.ConductorURL,
		}
		if attempt == 0 && strings.TrimSpace(opts.FirstRunDir) != "" {
			jobOpts.PreallocatedRunDir = opts.FirstRunDir
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
	if opts.MaxRestartsSet {
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
	if err := loop.Run(context.Background()); err != nil {
		obslog.Log(log.Default(), "ERROR", "runner", "task_run_failed",
			obslog.F("project_id", projectID),
			obslog.F("task_id", taskID),
			obslog.F("error", err),
		)
		return err
	}
	obslog.Log(log.Default(), "INFO", "runner", "task_loop_completed",
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
	)

	propagationResult, err := propagateTaskCompletionToProject(rootDir, projectID, taskID, taskDir, busPath)
	if err != nil {
		appendDependencyMessage(bus, projectID, taskID, "ERROR", fmt.Sprintf("task completion fact propagation failed: %v", err))
		obslog.Log(log.Default(), "ERROR", "runner", "task_completion_propagation_failed",
			obslog.F("project_id", projectID),
			obslog.F("task_id", taskID),
			obslog.F("error", err),
		)
		return nil
	}
	if propagationResult.Posted {
		appendDependencyMessage(bus, projectID, taskID, "FACT",
			fmt.Sprintf("task completion facts propagated to project bus (msg_id=%s key=%s)", propagationResult.ProjectMessageID, propagationResult.PropagationKey))
	} else {
		appendDependencyMessage(bus, projectID, taskID, "INFO",
			fmt.Sprintf("task completion facts already propagated (key=%s)", propagationResult.PropagationKey))
	}
	obslog.Log(log.Default(), "INFO", "runner", "task_run_finished",
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
		obslog.F("completion_msg_posted", propagationResult.Posted),
		obslog.F("propagation_key", propagationResult.PropagationKey),
	)
	return nil
}

func resolveTaskDependencies(rootDir, projectID, taskID, taskDir string, requested []string) ([]string, error) {
	dependsOn, err := taskdeps.ReadDependsOn(taskDir)
	if err != nil {
		return nil, errors.Wrap(err, "read task dependencies")
	}

	if requested != nil {
		dependsOn, err = taskdeps.Normalize(taskID, requested)
		if err != nil {
			return nil, errors.Wrap(err, "normalize task dependencies")
		}
	}

	if err := taskdeps.ValidateNoCycle(rootDir, projectID, taskID, dependsOn); err != nil {
		return nil, errors.Wrap(err, "validate task dependencies")
	}

	if requested != nil {
		if err := taskdeps.WriteDependsOn(taskDir, dependsOn); err != nil {
			return nil, errors.Wrap(err, "write task dependencies")
		}
	}

	return dependsOn, nil
}

func waitForDependencies(taskDir, rootDir, projectID, taskID string, dependsOn []string, pollInterval time.Duration, bus *messagebus.MessageBus) error {
	if len(dependsOn) == 0 {
		return nil
	}
	if pollInterval <= 0 {
		pollInterval = defaultDependencyPollInterval
	}

	lastBlocked := ""
	for {
		if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
			return nil
		} else if err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "stat DONE while waiting for dependencies")
		}

		blockedBy, err := taskdeps.BlockedBy(rootDir, projectID, dependsOn)
		if err != nil {
			return errors.Wrap(err, "resolve blocked dependencies")
		}
		if len(blockedBy) == 0 {
			if lastBlocked != "" {
				appendDependencyMessage(bus, projectID, taskID, "FACT", "dependencies satisfied; starting task")
			}
			return nil
		}

		current := strings.Join(blockedBy, ",")
		if current != lastBlocked {
			body := fmt.Sprintf("task blocked by dependencies: %s", strings.Join(blockedBy, ", "))
			appendDependencyMessage(bus, projectID, taskID, "PROGRESS", body)
			lastBlocked = current
		}

		time.Sleep(pollInterval)
	}
}

func appendDependencyMessage(bus *messagebus.MessageBus, projectID, taskID, msgType, body string) {
	if bus == nil {
		return
	}
	_, _ = bus.AppendMessage(&messagebus.Message{
		Type:      msgType,
		ProjectID: projectID,
		TaskID:    taskID,
		Body:      body,
	})
}
