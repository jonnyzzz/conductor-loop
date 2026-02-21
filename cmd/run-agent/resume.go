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

func newResumeCmd() *cobra.Command {
	var (
		projectID  string
		taskID     string
		root       string
		agent      string
		prompt     string
		promptFile string
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Reset an exhausted task's restart counter and optionally retry it",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			taskID = strings.TrimSpace(taskID)
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			if taskID == "" {
				return fmt.Errorf("--task is required")
			}
			if err := storage.ValidateTaskID(taskID); err != nil {
				return err
			}

			rootDir := strings.TrimSpace(root)
			if rootDir == "" {
				rootDir = "./runs"
			}

			taskDir := filepath.Join(rootDir, projectID, taskID)
			if _, err := os.Stat(taskDir); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("task directory not found: %s", taskDir)
				}
				return fmt.Errorf("stat task directory: %w", err)
			}

			// Delete the DONE file if it exists so the Ralph loop can run again.
			doneFile := filepath.Join(taskDir, "DONE")
			if err := os.Remove(doneFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove DONE file: %w", err)
			}

			// Note: the restart counter is tracked in-memory by the Ralph loop
			// and is not persisted to disk, so there is no counter file to delete.

			fmt.Fprintf(cmd.OutOrStdout(), "Resumed task %s (restart counter reset)\n", taskID)

			// If --agent is not specified, just reset and exit.
			agentType := strings.TrimSpace(agent)
			if agentType == "" {
				return nil
			}

			// Look for a default config if neither --config nor --agent is explicit.
			if strings.TrimSpace(configPath) == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				configPath = found
			}

			opts := runner.TaskOptions{
				RootDir:        rootDir,
				ConfigPath:     configPath,
				Agent:          agentType,
				Prompt:         strings.TrimSpace(prompt),
				PromptPath:     strings.TrimSpace(promptFile),
				ResumeMode:     true,
				MaxRestarts:    3,
				MaxRestartsSet: true,
				RestartDelay:   time.Second,
			}
			return runner.RunTask(projectID, taskID, opts)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (required)")
	cmd.Flags().StringVar(&root, "root", "./runs", "run-agent root directory")
	cmd.Flags().StringVar(&agent, "agent", "", "agent type; if set, launches a new run after reset")
	cmd.Flags().StringVar(&prompt, "prompt", "", "prompt text (used when --agent is set)")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "prompt file path (used when --agent is set)")
	cmd.Flags().StringVar(&configPath, "config", "", "config file path")

	return cmd
}
