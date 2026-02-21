package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

func newTaskDeleteCmd() *cobra.Command {
	var (
		projectID string
		taskID    string
		root      string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a task and all its runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskDelete(projectID, taskID, root, force)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (required)")
	cmd.Flags().StringVar(&root, "root", "", "run-agent root directory (default: $RUNS_DIR or ./runs)")
	cmd.Flags().BoolVar(&force, "force", false, "delete even if task has running runs")

	return cmd
}

func runTaskDelete(projectID, taskID, root string, force bool) error {
	if projectID == "" {
		return fmt.Errorf("--project is required")
	}
	if taskID == "" {
		return fmt.Errorf("--task is required")
	}

	rootDir := resolveRootDir(root)

	taskDir := filepath.Join(rootDir, projectID, taskID)
	if _, err := os.Stat(taskDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("task not found: %s/%s", projectID, taskID)
		}
		return fmt.Errorf("stat task directory: %w", err)
	}

	if !force {
		// Scan for running runs.
		runsDir := filepath.Join(taskDir, "runs")
		entries, err := os.ReadDir(runsDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read runs directory: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			infoPath := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
			info, err := storage.ReadRunInfo(infoPath)
			if err != nil {
				continue
			}
			if info.Status == storage.StatusRunning {
				return fmt.Errorf("task %s has a running run (%s); use --force to delete anyway", taskID, entry.Name())
			}
		}
	}

	if err := os.RemoveAll(taskDir); err != nil {
		return fmt.Errorf("delete task directory: %w", err)
	}

	fmt.Printf("Deleted task: %s\n", taskID)
	return nil
}

// resolveRootDir returns the root directory from the flag, $RUNS_DIR env var, or "./runs".
func resolveRootDir(root string) string {
	if root != "" {
		return root
	}
	if env := os.Getenv("RUNS_DIR"); env != "" {
		return env
	}
	return "runs"
}
