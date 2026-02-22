package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

func newWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Workflow orchestration utilities",
	}
	cmd.AddCommand(newWorkflowRunCmd())
	return cmd
}

func newWorkflowRunCmd() *cobra.Command {
	var (
		projectID string
		taskID    string
		opts      runner.WorkflowOptions
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run staged workflow execution with persisted stage state",
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

			if !opts.DryRun && strings.TrimSpace(opts.ConfigPath) == "" && strings.TrimSpace(opts.Agent) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				opts.ConfigPath = found
			}

			result, err := runner.RunWorkflow(projectID, taskID, opts)
			if err != nil {
				if result != nil {
					if writeErr := printConductorWorkflowRunSummary(cmd.OutOrStdout(), result); writeErr != nil {
						return writeErr
					}
				}
				return err
			}
			if jsonOut {
				data, err := runner.WorkflowResultJSON(result)
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			return printConductorWorkflowRunSummary(cmd.OutOrStdout(), result)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory")
	cmd.Flags().StringVar(&opts.MessageBusPath, "message-bus", "", "message bus path")
	cmd.Flags().StringVar(&opts.Template, "template", runner.WorkflowTemplatePromptV5, "workflow template")
	cmd.Flags().IntVar(&opts.FromStage, "from-stage", 0, "first stage to execute")
	cmd.Flags().IntVar(&opts.ToStage, "to-stage", 12, "last stage to execute")
	cmd.Flags().BoolVar(&opts.Resume, "resume", false, "resume from persisted stage state (skip completed stages)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "plan stages and persist state without executing jobs")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "idle output timeout per stage (e.g. 30m, 2h); 0 means no limit")
	cmd.Flags().StringVar(&opts.StatePath, "state-file", "", "override stage state file path")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output workflow result as JSON")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("task")

	return cmd
}

func printConductorWorkflowRunSummary(out io.Writer, result *runner.WorkflowResult) error {
	if result == nil {
		return nil
	}
	if _, err := fmt.Fprintf(out, "workflow state: %s\n", result.StatePath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "template: %s stages: %d..%d\n", result.Template, result.FromStage, result.ToStage); err != nil {
		return err
	}
	if result.DryRun {
		_, err := fmt.Fprintf(out, "dry-run planned stages: %v\n", result.PlannedStages)
		return err
	}
	if _, err := fmt.Fprintf(out, "executed stages: %v\n", result.ExecutedStages); err != nil {
		return err
	}
	if len(result.SkippedStages) > 0 {
		if _, err := fmt.Fprintf(out, "skipped stages: %v\n", result.SkippedStages); err != nil {
			return err
		}
	}
	if !result.State.CompletedAt.IsZero() {
		if _, err := fmt.Fprintf(out, "completed at: %s\n", result.State.CompletedAt.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return nil
}
