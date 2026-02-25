package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
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
			var err error
			rootDir, err = config.ResolveRunsDir(rootDir)
			if err != nil {
				return fmt.Errorf("resolve runs dir: %w", err)
			}
			return runWatch(cmd.OutOrStdout(), rootDir, projectID, taskIDs, timeout, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringArrayVar(&taskIDs, "task", nil, "task id(s) to watch (can repeat)")
	cmd.Flags().StringVar(&rootDir, "root", "", "runs root directory (default: ~/.run-agent/runs)")
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

type watchPhaseTotals struct {
	Active    int
	Blocked   int
	Completed int
	Failed    int
	Pending   int
}

// getWatchTaskStatus reads the latest run-info.yaml for a task and returns its status.
func getWatchTaskStatus(root, projectID, taskID string) watchTaskStatus {
	ts := watchTaskStatus{TaskID: taskID, Status: "unknown"}
	taskDir := filepath.Join(root, projectID, taskID)
	dependsOn, _ := taskdeps.ReadDependsOn(taskDir)

	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if _, doneErr := os.Stat(filepath.Join(taskDir, "DONE")); doneErr == nil {
			ts.Status = "done"
			ts.Done = true
			return ts
		}
		if len(dependsOn) > 0 {
			if blockedBy, depErr := taskdeps.BlockedBy(root, projectID, dependsOn); depErr == nil && len(blockedBy) > 0 {
				ts.Status = "blocked"
				return ts
			}
		}
		return ts
	}

	var runNames []string
	for _, e := range entries {
		if e.IsDir() {
			runNames = append(runNames, e.Name())
		}
	}
	if len(runNames) == 0 {
		if _, doneErr := os.Stat(filepath.Join(taskDir, "DONE")); doneErr == nil {
			ts.Status = "done"
			ts.Done = true
			return ts
		}
		if len(dependsOn) > 0 {
			if blockedBy, depErr := taskdeps.BlockedBy(root, projectID, dependsOn); depErr == nil && len(blockedBy) > 0 {
				ts.Status = "blocked"
			}
		}
		return ts
	}
	sort.Strings(runNames)
	info := latestReadableWatchRunInfo(runsDir, runNames)
	if info == nil {
		if _, doneErr := os.Stat(filepath.Join(taskDir, "DONE")); doneErr == nil {
			ts.Status = "done"
			ts.Done = true
			return ts
		}
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
	ts.Done = info.Status == storage.StatusCompleted || info.Status == storage.StatusFailed || info.Status == "done"

	return ts
}

func latestReadableWatchRunInfo(runsDir string, runNames []string) *storage.RunInfo {
	for i := len(runNames) - 1; i >= 0; i-- {
		infoPath := filepath.Join(runsDir, runNames[i], "run-info.yaml")
		info, err := runstate.ReadRunInfo(infoPath)
		if err != nil {
			continue
		}
		return info
	}
	return nil
}

func watchPhaseForStatus(status string) string {
	switch status {
	case storage.StatusRunning:
		return "active"
	case "blocked":
		return "blocked"
	case storage.StatusCompleted:
		return "completed"
	case "done":
		return "completed"
	case storage.StatusFailed:
		return "failed"
	default:
		return "pending"
	}
}

func watchDurationLabel(ts watchTaskStatus) string {
	if ts.Done {
		return "duration"
	}
	return "elapsed"
}

func watchDurationParts(ts watchTaskStatus) (int, int) {
	elapsed := int(ts.Elapsed)
	return elapsed / 60, elapsed % 60
}

func buildWatchPhaseTotals(statuses []watchTaskStatus) watchPhaseTotals {
	totals := watchPhaseTotals{}
	for _, ts := range statuses {
		switch watchPhaseForStatus(ts.Status) {
		case "active":
			totals.Active++
		case "blocked":
			totals.Blocked++
		case "completed":
			totals.Completed++
		case "failed":
			totals.Failed++
		default:
			totals.Pending++
		}
	}
	return totals
}

func runWatch(out io.Writer, root, projectID string, taskIDs []string, timeout time.Duration, jsonOutput bool) error {
	if len(taskIDs) == 0 {
		return fmt.Errorf("at least one --task is required")
	}

	deadline := time.Now().Add(timeout)
	previousPhases := make(map[string]string, len(taskIDs))

	if !jsonOutput {
		fmt.Fprintf(out, "Watching %d task(s) for project %q:\n", len(taskIDs), projectID)
	}

	for {
		statuses := make([]watchTaskStatus, len(taskIDs))
		allDone := true

		for i, taskID := range taskIDs {
			statuses[i] = getWatchTaskStatus(root, projectID, taskID)
			if !statuses[i].Done {
				allDone = false
			}
		}

		remaining := time.Until(deadline)
		totals := buildWatchPhaseTotals(statuses)

		if jsonOutput {
			type jsonPayload struct {
				Tasks   []watchTaskStatus `json:"tasks"`
				AllDone bool              `json:"all_done"`
			}
			enc := json.NewEncoder(out)
			_ = enc.Encode(jsonPayload{Tasks: statuses, AllDone: allDone})
		} else {
			fmt.Fprintf(
				out,
				"  phase counts: active=%d blocked=%d completed=%d failed=%d pending=%d\n",
				totals.Active,
				totals.Blocked,
				totals.Completed,
				totals.Failed,
				totals.Pending,
			)
			for _, ts := range statuses {
				minutes, seconds := watchDurationParts(ts)
				phase := watchPhaseForStatus(ts.Status)
				fmt.Fprintf(out, "  %-45s [%-9s] %-9s %s: %dm%ds\n", ts.TaskID, ts.Status, phase, watchDurationLabel(ts), minutes, seconds)

				previousPhase, hasPrevious := previousPhases[ts.TaskID]
				if !hasPrevious {
					fmt.Fprintf(out, "    transition: unknown -> %s\n", phase)
				} else if previousPhase != phase {
					fmt.Fprintf(out, "    transition: %s -> %s\n", previousPhase, phase)
				}
				previousPhases[ts.TaskID] = phase
			}
			if allDone {
				fmt.Fprintf(out, "All tasks complete.\n")
			} else {
				fmt.Fprintf(
					out,
					"Waiting for %d active and %d blocked task(s)... (timeout in %s)\n",
					totals.Active,
					totals.Blocked,
					remaining.Round(time.Second),
				)
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
