package main

import (
	"fmt"
	"os"
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
	cmd.AddCommand(newJobCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newBusCmd())
	cmd.AddCommand(newGCCmd())
	cmd.AddCommand(newValidateCmd())

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
			taskID = strings.TrimSpace(taskID)
			if projectID == "" {
				return fmt.Errorf("project is required")
			}
			var err error
			taskID, err = resolveTaskID(taskID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}
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
	cmd.Flags().IntVar(&opts.MaxRestarts, "max-restarts", 0, "max restarts")
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
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: "Run a single agent job",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			taskID = strings.TrimSpace(taskID)
			if projectID == "" {
				return fmt.Errorf("project is required")
			}
			var err error
			taskID, err = resolveTaskID(taskID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}
			return runner.RunJob(projectID, taskID, opts)
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
	cmd.Flags().StringVar(&opts.ParentRunID, "parent-run-id", "", "parent run id")
	cmd.Flags().StringVar(&opts.PreviousRunID, "previous-run-id", "", "previous run id")

	return cmd
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
