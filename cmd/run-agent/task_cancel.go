package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const cancelledFile = "CANCELLED"

// newTaskCancelCmd returns the "run-agent task cancel" subcommand.
// It writes a CANCELLED marker to the task directory with a timestamp and reason,
// allowing the task to be marked as resolved without deleting its folder.
func newTaskCancelCmd() *cobra.Command {
	var (
		rootDir   string
		projectID string
		taskID    string
		reason    string
	)

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Mark a task as cancelled (writes CANCELLED marker with reason)",
		Long: `Cancel a task by writing a CANCELLED marker file to the task directory.
This does not delete the task folder or its run history.
A cancelled task will not be restarted by monitor loops.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir = strings.TrimSpace(rootDir)
			projectID = strings.TrimSpace(projectID)
			taskID = strings.TrimSpace(taskID)
			reason = strings.TrimSpace(reason)

			if rootDir == "" {
				return fmt.Errorf("--root is required")
			}
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			if taskID == "" {
				return fmt.Errorf("--task is required")
			}
			if reason == "" {
				return fmt.Errorf("--reason is required")
			}

			taskDir := filepath.Join(rootDir, projectID, taskID)
			if _, err := os.Stat(taskDir); os.IsNotExist(err) {
				return fmt.Errorf("task directory not found: %s", taskDir)
			}

			cancelPath := filepath.Join(taskDir, cancelledFile)
			content := fmt.Sprintf("cancelled_at: %s\nreason: %s\n",
				time.Now().UTC().Format(time.RFC3339), reason)

			if err := os.WriteFile(cancelPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write CANCELLED marker: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "task %s/%s marked as cancelled\nreason: %s\n",
				projectID, taskID, reason)
			return nil
		},
	}

	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&reason, "reason", "", "cancellation reason (required)")

	return cmd
}
