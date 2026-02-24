package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

const version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "run-agent",
		Short:        "Conductor Loop run-agent CLI",
		Version:      version,
		SilenceUsage: true,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.AddCommand(newTaskCmd())
	cmd.AddCommand(newGoalCmd())
	cmd.AddCommand(newWorkflowCmd())
	cmd.AddCommand(newJobCmd())
	cmd.AddCommand(newWrapCmd())
	cmd.AddCommand(newShellSetupCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newBusCmd())
	cmd.AddCommand(newGCCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newOutputCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newWatchCmd())
	cmd.AddCommand(newResumeCmd())
	cmd.AddCommand(newMonitorCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newIterateCmd())
	cmd.AddCommand(newReviewCmd())

	return cmd
}

func newTaskCmd() *cobra.Command {
	var (
		projectID string
		taskID    string
		opts      runner.TaskOptions
	)

	cmd := &cobra.Command{
		Use:   "task",
		Short: "Run a task with the Ralph loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			originalTaskID := strings.TrimSpace(taskID)
			if projectID == "" {
				return fmt.Errorf("project is required")
			}
			var err error
			taskID, err = resolveTaskID(originalTaskID)
			if err != nil {
				return err
			}
			if originalTaskID == "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "task: %s\n", taskID)
			}
			if strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}
			opts.MaxRestartsSet = cmd.Flags().Changed("max-restarts")
			return runner.RunTask(projectID, taskID, opts)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type")
	cmd.Flags().StringVar(&opts.Prompt, "prompt", "", "prompt override")
	cmd.Flags().StringVar(&opts.PromptPath, "prompt-file", "", "prompt file path")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory")
	cmd.Flags().StringVar(&opts.MessageBusPath, "message-bus", "", "message bus path")
	cmd.Flags().StringArrayVar(&opts.DependsOn, "depends-on", nil, "task dependencies (repeat or comma-separate)")
	cmd.Flags().DurationVar(&opts.DependencyPollInterval, "dependency-poll-interval", 0, "dependency check poll interval while blocked (default: 2s)")
	cmd.Flags().StringVar(&opts.ConductorURL, "conductor-url", "", "conductor server URL (e.g. http://127.0.0.1:14355)")
	cmd.Flags().IntVar(&opts.MaxRestarts, "max-restarts", 0, "max restarts")
	cmd.Flags().DurationVar(&opts.WaitTimeout, "child-wait-timeout", 0, "child wait timeout")
	cmd.Flags().DurationVar(&opts.PollInterval, "child-poll-interval", 0, "child poll interval")
	cmd.Flags().DurationVar(&opts.RestartDelay, "restart-delay", time.Second, "restart delay")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "idle output timeout per job (e.g. 30m, 2h); 0 means no limit")

	cmd.AddCommand(newTaskResumeCmd())
	cmd.AddCommand(newTaskDeleteCmd())
	cmd.AddCommand(newTaskCancelCmd())

	return cmd
}

func newTaskResumeCmd() *cobra.Command {
	var (
		projectID string
		taskID    string
		opts      runner.TaskOptions
	)

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume a stopped or failed task",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			taskID = strings.TrimSpace(taskID)
			if projectID == "" {
				return fmt.Errorf("project is required")
			}
			if taskID == "" {
				return fmt.Errorf("task is required")
			}
			if err := storage.ValidateTaskID(taskID); err != nil {
				return err
			}
			if strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}
			// Resolve root dir to validate task directory existence
			rootDir := strings.TrimSpace(opts.RootDir)
			if rootDir == "" {
				var err error
				rootDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("resolve root dir: %w", err)
				}
			}
			taskDir := filepath.Join(rootDir, projectID, taskID)
			if _, err := os.Stat(taskDir); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("task directory not found: %s", taskDir)
				}
				return fmt.Errorf("stat task directory: %w", err)
			}
			taskMDPath := filepath.Join(taskDir, "TASK.md")
			if _, err := os.Stat(taskMDPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("TASK.md not found in task directory %s", taskDir)
				}
				return fmt.Errorf("stat TASK.md: %w", err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Resuming task: %s\n", taskID)
			opts.ResumeMode = true
			opts.MaxRestartsSet = true
			return runner.RunTask(projectID, taskID, opts)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory")
	cmd.Flags().StringVar(&opts.MessageBusPath, "message-bus", "", "message bus path")
	cmd.Flags().IntVar(&opts.MaxRestarts, "max-restarts", 3, "max restarts")
	cmd.Flags().DurationVar(&opts.WaitTimeout, "child-wait-timeout", 0, "child wait timeout")
	cmd.Flags().DurationVar(&opts.PollInterval, "child-poll-interval", 0, "child poll interval")
	cmd.Flags().DurationVar(&opts.RestartDelay, "restart-delay", time.Second, "restart delay")

	return cmd
}

func newJobCmd() *cobra.Command {
	var (
		projectID string
		taskID    string
		opts      runner.JobOptions
		follow    bool
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: "Run a single agent job",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			originalTaskID := strings.TrimSpace(taskID)
			if projectID == "" {
				return fmt.Errorf("project is required")
			}
			var err error
			taskID, err = resolveTaskID(originalTaskID)
			if err != nil {
				return err
			}
			if originalTaskID == "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "task: %s\n", taskID)
			}
			if strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}
			return runSingleJob(projectID, taskID, opts, follow)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type")
	cmd.Flags().StringVar(&opts.Prompt, "prompt", "", "prompt text")
	cmd.Flags().StringVar(&opts.PromptPath, "prompt-file", "", "prompt file path")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory")
	cmd.Flags().StringVar(&opts.MessageBusPath, "message-bus", "", "message bus path")
	cmd.Flags().StringVar(&opts.ConductorURL, "conductor-url", "", "conductor server URL (e.g. http://127.0.0.1:14355)")
	cmd.Flags().StringVar(&opts.ParentRunID, "parent-run-id", "", "parent run id")
	cmd.Flags().StringVar(&opts.PreviousRunID, "previous-run-id", "", "previous run id")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "idle output timeout (e.g. 30m, 2h); 0 means no limit")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream output in real-time while job runs")

	cmd.AddCommand(newJobBatchCmd())

	return cmd
}

func newJobBatchCmd() *cobra.Command {
	var (
		projectID      string
		taskIDs        []string
		prompts        []string
		promptFiles    []string
		opts           runner.JobOptions
		follow         bool
		continueOnFail bool
	)

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Run multiple agent jobs sequentially",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			if projectID == "" {
				return fmt.Errorf("project is required")
			}

			batchPrompts, err := loadBatchPrompts(prompts, promptFiles)
			if err != nil {
				return err
			}
			if len(batchPrompts) == 0 {
				return fmt.Errorf("at least one --prompt or --prompt-file is required")
			}

			if strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}

			if len(taskIDs) != 0 && len(taskIDs) != len(batchPrompts) {
				return fmt.Errorf("--task count (%d) must match prompt count (%d)", len(taskIDs), len(batchPrompts))
			}

			var firstErr error
			for i, prompt := range batchPrompts {
				currentTaskID := storage.GenerateTaskID("")
				if len(taskIDs) > 0 {
					currentTaskID = strings.TrimSpace(taskIDs[i])
					if currentTaskID == "" {
						return fmt.Errorf("task at index %d is empty", i)
					}
					if err := storage.ValidateTaskID(currentTaskID); err != nil {
						return err
					}
				}

				currentOpts := opts
				currentOpts.Prompt = prompt
				currentOpts.PromptPath = ""
				currentOpts.PreallocatedRunDir = ""
				currentOpts.PreviousRunID = ""

				err := runSingleJob(projectID, currentTaskID, currentOpts, follow)
				if err != nil {
					if !continueOnFail {
						return fmt.Errorf("batch item %d (task %s): %w", i+1, currentTaskID, err)
					}
					if firstErr == nil {
						firstErr = err
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "batch item %d failed (task %s): %v\n", i+1, currentTaskID, err)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "batch item %d/%d completed: %s\n", i+1, len(batchPrompts), currentTaskID)
			}

			if firstErr != nil {
				return fmt.Errorf("one or more batch items failed: %w", firstErr)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringArrayVar(&taskIDs, "task", nil, "task id per prompt (repeat; must match prompt count)")
	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type")
	cmd.Flags().StringArrayVar(&prompts, "prompt", nil, "prompt text (repeat)")
	cmd.Flags().StringArrayVar(&promptFiles, "prompt-file", nil, "prompt file path (repeat)")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory")
	cmd.Flags().StringVar(&opts.MessageBusPath, "message-bus", "", "message bus path")
	cmd.Flags().StringVar(&opts.ConductorURL, "conductor-url", "", "conductor server URL (e.g. http://127.0.0.1:14355)")
	cmd.Flags().StringVar(&opts.ParentRunID, "parent-run-id", "", "parent run id for all submitted jobs")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "idle output timeout (e.g. 30m, 2h); 0 means no limit")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream output in real-time while each job runs")
	cmd.Flags().BoolVar(&continueOnFail, "continue-on-fail", false, "continue submitting remaining jobs when one fails")

	return cmd
}

func runSingleJob(projectID, taskID string, opts runner.JobOptions, follow bool) error {
	if !follow {
		return runner.RunJob(projectID, taskID, opts)
	}
	// Pre-allocate run directory so we can follow output immediately.
	rootDir := opts.RootDir
	if rootDir == "" {
		if v := os.Getenv("RUNS_DIR"); v != "" {
			rootDir = v
		} else {
			rootDir = "./runs"
		}
	}
	runsDir := filepath.Join(rootDir, projectID, taskID, "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return fmt.Errorf("create runs dir: %w", err)
	}
	_, runDir, err := runner.AllocateRunDir(runsDir)
	if err != nil {
		return fmt.Errorf("allocate run dir: %w", err)
	}
	opts.PreallocatedRunDir = runDir
	jobDone := make(chan error, 1)
	go func() {
		jobDone <- runner.RunJob(projectID, taskID, opts)
	}()
	_ = followOutput(runDir, "")
	return <-jobDone
}

func loadBatchPrompts(prompts []string, promptFiles []string) ([]string, error) {
	result := make([]string, 0, len(prompts)+len(promptFiles))
	for i, prompt := range prompts {
		trimmed := strings.TrimSpace(prompt)
		if trimmed == "" {
			return nil, fmt.Errorf("prompt at index %d is empty", i)
		}
		result = append(result, prompt)
	}
	for _, promptFile := range promptFiles {
		trimmedPath := strings.TrimSpace(promptFile)
		if trimmedPath == "" {
			return nil, fmt.Errorf("prompt-file cannot be empty")
		}
		content, err := os.ReadFile(trimmedPath)
		if err != nil {
			return nil, fmt.Errorf("read prompt file %q: %w", trimmedPath, err)
		}
		trimmed := strings.TrimSpace(string(content))
		if trimmed == "" {
			return nil, fmt.Errorf("prompt file %q is empty", trimmedPath)
		}
		result = append(result, string(content))
	}
	return result, nil
}

// resolveTaskID returns a valid task ID. If taskID is empty, a new ID is
// auto-generated. If taskID is provided, it is validated against the required
// format; an error is returned if validation fails.
func resolveTaskID(taskID string) (string, error) {
	if taskID == "" {
		return storage.GenerateTaskID(""), nil
	}
	if err := storage.ValidateTaskID(taskID); err != nil {
		return "", err
	}
	return taskID, nil
}
