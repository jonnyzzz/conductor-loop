package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

// watchPollInterval is the interval between status polls. Package-level for testing.
var watchPollInterval = 2 * time.Second

func newWatchCmd() *cobra.Command {
	var (
		projectID  string
		taskIDs    []string
		rootDir    string
		timeout    time.Duration
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch tasks until completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootDir == "" {
				if v := os.Getenv("RUNS_DIR"); v != "" {
					rootDir = v
				} else {
					rootDir = "./runs"
				}
			}
			return runWatch(cmd.OutOrStdout(), rootDir, projectID, taskIDs, timeout, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringArrayVar(&taskIDs, "task", nil, "task id(s) to watch (can repeat)")
	cmd.Flags().StringVar(&rootDir, "root", "", "runs root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "max wait time (exit code 1 on timeout)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

// watchTaskStatus holds the current observed status of a watched task.
type watchTaskStatus struct {
	TaskID  string  `json:"task_id"`
	Status  string  `json:"status"`
	Elapsed float64 `json:"elapsed"` // seconds
	Done    bool    `json:"done"`
}

// getWatchTaskStatus reads the latest run-info.yaml for a task and returns its status.
func getWatchTaskStatus(root, projectID, taskID string) watchTaskStatus {
	ts := watchTaskStatus{TaskID: taskID, Status: "unknown"}

	runsDir := filepath.Join(root, projectID, taskID, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return ts
	}

	var runNames []string
	for _, e := range entries {
		if e.IsDir() {
			runNames = append(runNames, e.Name())
		}
	}
	if len(runNames) == 0 {
		return ts
	}
	sort.Strings(runNames)
	latest := runNames[len(runNames)-1]

	infoPath := filepath.Join(runsDir, latest, "run-info.yaml")
	info, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		return ts
	}

	ts.Status = info.Status
	if !info.StartTime.IsZero() {
		switch info.Status {
		case storage.StatusCompleted, storage.StatusFailed:
			if !info.EndTime.IsZero() {
				ts.Elapsed = info.EndTime.Sub(info.StartTime).Seconds()
			} else {
				ts.Elapsed = time.Since(info.StartTime).Seconds()
			}
		default:
			ts.Elapsed = time.Since(info.StartTime).Seconds()
		}
	}
	ts.Done = info.Status == storage.StatusCompleted || info.Status == storage.StatusFailed

	return ts
}

func runWatch(out io.Writer, root, projectID string, taskIDs []string, timeout time.Duration, jsonOutput bool) error {
	if len(taskIDs) == 0 {
		return fmt.Errorf("at least one --task is required")
	}

	deadline := time.Now().Add(timeout)

	if !jsonOutput {
		fmt.Fprintf(out, "Watching %d task(s) for project %q:\n", len(taskIDs), projectID)
	}

	for {
		statuses := make([]watchTaskStatus, len(taskIDs))
		allDone := true
		runningCount := 0

		for i, taskID := range taskIDs {
			statuses[i] = getWatchTaskStatus(root, projectID, taskID)
			if !statuses[i].Done {
				allDone = false
				if statuses[i].Status == storage.StatusRunning {
					runningCount++
				}
			}
		}

		remaining := time.Until(deadline)

		if jsonOutput {
			type jsonPayload struct {
				Tasks   []watchTaskStatus `json:"tasks"`
				AllDone bool              `json:"all_done"`
			}
			enc := json.NewEncoder(out)
			_ = enc.Encode(jsonPayload{Tasks: statuses, AllDone: allDone})
		} else {
			for _, ts := range statuses {
				elapsed := int(ts.Elapsed)
				minutes := elapsed / 60
				seconds := elapsed % 60
				label := "elapsed"
				if ts.Done {
					label = "duration"
				}
				fmt.Fprintf(out, "  %-45s [%-9s] %s: %dm%ds\n", ts.TaskID, ts.Status, label, minutes, seconds)
			}
			if allDone {
				fmt.Fprintf(out, "All tasks complete.\n")
			} else {
				fmt.Fprintf(out, "Waiting for %d running task(s)... (timeout in %s)\n", runningCount, remaining.Round(time.Second))
			}
		}

		if allDone {
			return nil
		}

		if remaining <= 0 {
			return fmt.Errorf("timeout after %s: not all tasks completed", timeout)
		}

		time.Sleep(watchPollInterval)
	}
}
