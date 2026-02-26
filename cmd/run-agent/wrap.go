package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/spf13/cobra"
)

func newWrapCmd() *cobra.Command {
	var (
		taskID    string
		agentType string
		opts      runner.WrapOptions
	)

	cmd := &cobra.Command{
		Use:   "wrap --agent <agent> -- [args...]",
		Short: "Run an agent CLI command with tracked task/run metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			cleanAgent := strings.TrimSpace(agentType)
			if cleanAgent == "" {
				return fmt.Errorf("agent is required")
			}

			resolvedProjectID, err := resolveWrapProjectID(opts.RootDir)
			if err != nil {
				return err
			}
			originalTaskID := strings.TrimSpace(taskID)
			resolvedTaskID, err := resolveTaskID(originalTaskID)
			if err != nil {
				return err
			}
			if originalTaskID == "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "task: %s\n", resolvedTaskID)
			}

			if strings.TrimSpace(opts.ParentRunID) == "" {
				opts.ParentRunID = strings.TrimSpace(os.Getenv("JRUN_ID"))
			}

			return runner.RunWrap(resolvedProjectID, resolvedTaskID, cleanAgent, args, opts)
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "task id (auto-generated if omitted)")
	cmd.Flags().StringVar(&agentType, "agent", "", "agent command to execute (e.g. claude, codex, gemini)")
	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory (default: current working directory)")
	cmd.Flags().StringVar(&opts.ParentRunID, "parent-run-id", "", "parent run id (defaults to JRUN_ID if set)")
	cmd.Flags().StringVar(&opts.PreviousRunID, "previous-run-id", "", "previous run id")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "maximum wrapped command duration (e.g. 30m, 2h); 0 means no limit")
	cmd.Flags().StringVar(&opts.TaskPrompt, "task-prompt", "", "TASK.md content used only when creating a new task directory")

	return cmd
}

func resolveWrapProjectID(rootDir string) (string, error) {
	taskFolderProject, _ := inferMessageScopeFromTaskFolder(os.Getenv("JRUN_TASK_FOLDER"))
	runFolderProject, _, _ := inferMessageScopeFromRunFolder(os.Getenv("JRUN_RUN_FOLDER"))
	cwdProject, _, _, _ := inferScopeFromCWDRunInfo()

	projectID := firstNonEmpty(
		runFolderProject,
		taskFolderProject,
		strings.TrimSpace(os.Getenv("JRUN_PROJECT_ID")),
		cwdProject,
		inferProjectFromCWD(),
	)
	if projectID != "" {
		return projectID, nil
	}
	return resolveOrInitProject(rootDir)
}

func sanitizeProjectID(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
