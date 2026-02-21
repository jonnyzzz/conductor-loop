package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// watchServerPollInterval is the default poll interval; package-level for testing override.
var watchServerPollInterval = 5 * time.Second

func newWatchCmd() *cobra.Command {
	var (
		server     string
		project    string
		taskIDs    []string
		timeout    time.Duration
		interval   time.Duration
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch tasks until completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			effectiveInterval := watchServerPollInterval
			if cmd.Flags().Changed("interval") {
				effectiveInterval = interval
			}
			return runConductorWatch(cmd.OutOrStdout(), server, project, taskIDs, timeout, effectiveInterval, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:14355", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringArrayVar(&taskIDs, "task", nil, "task ID(s) to watch (can repeat; default: all tasks in project)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "max wait time (exit code 1 on timeout)")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "poll interval")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output final status as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// conductorWatchStatus holds the observed status of a watched task.
type conductorWatchStatus struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	RunCount int    `json:"run_count"`
	Done     bool   `json:"done"`
}

// isConductorTerminalStatus returns true if status is a terminal state.
func isConductorTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "done", "error":
		return true
	}
	return false
}

// fetchAllTaskStatuses fetches all tasks in a project from the conductor server.
func fetchAllTaskStatuses(server, project string) ([]conductorWatchStatus, error) {
	url := server + "/api/projects/" + project + "/tasks"
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("fetch tasks: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result taskListAPIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	statuses := make([]conductorWatchStatus, 0, len(result.Items))
	for _, item := range result.Items {
		statuses = append(statuses, conductorWatchStatus{
			TaskID:   item.ID,
			Status:   item.Status,
			RunCount: item.RunCount,
			Done:     isConductorTerminalStatus(item.Status),
		})
	}
	return statuses, nil
}

// fetchSingleTaskStatus fetches one task's status via the project-scoped task detail endpoint.
func fetchSingleTaskStatus(server, project, taskID string) (conductorWatchStatus, error) {
	url := server + "/api/projects/" + project + "/tasks/" + taskID
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return conductorWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("fetch task %s: %w", taskID, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return conductorWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return conductorWatchStatus{TaskID: taskID, Status: "not_found", Done: false}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return conductorWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	// projectTask shape: id, status, done, runs (array)
	var detail struct {
		ID     string            `json:"id"`
		Status string            `json:"status"`
		Done   bool              `json:"done"`
		Runs   []json.RawMessage `json:"runs"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return conductorWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("decode response: %w", err)
	}

	return conductorWatchStatus{
		TaskID:   taskID,
		Status:   detail.Status,
		RunCount: len(detail.Runs),
		Done:     detail.Done || isConductorTerminalStatus(detail.Status),
	}, nil
}

func runConductorWatch(out io.Writer, server, project string, taskIDs []string, timeout, interval time.Duration, jsonOutput bool) error {
	deadline := time.Now().Add(timeout)
	pollNum := 0
	watchingAll := len(taskIDs) == 0

	if !jsonOutput {
		if watchingAll {
			fmt.Fprintf(out, "Watching all tasks in project %q...\n", project)
		} else {
			fmt.Fprintf(out, "Watching %d task(s) in project %q...\n", len(taskIDs), project)
		}
	}

	for {
		pollNum++

		var (
			statuses []conductorWatchStatus
			err      error
		)

		if watchingAll {
			statuses, err = fetchAllTaskStatuses(server, project)
			if err != nil {
				return err
			}
			if len(statuses) == 0 {
				if !jsonOutput {
					fmt.Fprintf(out, "No tasks found in project %q.\n", project)
				}
				return fmt.Errorf("no tasks found in project %q", project)
			}
		} else {
			statuses = make([]conductorWatchStatus, 0, len(taskIDs))
			for _, taskID := range taskIDs {
				s, fetchErr := fetchSingleTaskStatus(server, project, taskID)
				if fetchErr != nil {
					return fetchErr
				}
				statuses = append(statuses, s)
			}
		}

		allDone := len(statuses) > 0
		for _, s := range statuses {
			if !s.Done {
				allDone = false
				break
			}
		}

		remaining := time.Until(deadline)

		if !jsonOutput {
			fmt.Fprintf(out, "[%s] Poll #%d\n", time.Now().Format("2006-01-02 15:04:05"), pollNum)
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "TASK ID\tSTATUS\tRUNS")
			for _, s := range statuses {
				fmt.Fprintf(w, "%s\t%s\t%d\n", s.TaskID, s.Status, s.RunCount)
			}
			w.Flush()
			fmt.Fprintln(out)
		}

		if allDone {
			if jsonOutput {
				type jsonResult struct {
					Tasks   []conductorWatchStatus `json:"tasks"`
					AllDone bool                   `json:"all_done"`
				}
				enc := json.NewEncoder(out)
				_ = enc.Encode(jsonResult{Tasks: statuses, AllDone: true})
			} else {
				fmt.Fprintln(out, "All tasks completed.")
			}
			return nil
		}

		if remaining <= 0 {
			return fmt.Errorf("timeout after %s: not all tasks completed", timeout)
		}

		time.Sleep(interval)
	}
}
