package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var (
		root      string
		projectID string
		taskID    string
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show latest task run status for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if root == "" {
				if v := os.Getenv("RUNS_DIR"); v != "" {
					root = v
				} else {
					root = "./runs"
				}
			}
			return runStatus(cmd.OutOrStdout(), root, projectID, taskID, jsonOut)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (optional; defaults to all tasks in project)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

type statusRow struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	ExitCode  *int   `json:"exit_code"`
	LatestRun string `json:"latest_run"`
	Done      bool   `json:"done"`
	PIDAlive  *bool  `json:"pid_alive"`
	DependsOn []string `json:"depends_on,omitempty"`
	BlockedBy []string `json:"blocked_by,omitempty"`
}

func runStatus(out io.Writer, root, projectID, taskID string, jsonOut bool) error {
	projectID = strings.TrimSpace(projectID)
	taskID = strings.TrimSpace(taskID)
	if projectID == "" {
		return fmt.Errorf("--project is required")
	}

	projectDir := filepath.Join(root, projectID)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project directory not found: %s", projectDir)
		}
		return errors.Wrap(err, "read project directory")
	}

	taskIDs := make([]string, 0, len(entries))
	if taskID != "" {
		taskDir := filepath.Join(projectDir, taskID)
		info, err := os.Stat(taskDir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("task directory not found: %s", taskDir)
			}
			return errors.Wrap(err, "stat task directory")
		}
		if !info.IsDir() {
			return fmt.Errorf("task path is not a directory: %s", taskDir)
		}
		taskIDs = append(taskIDs, taskID)
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				taskIDs = append(taskIDs, entry.Name())
			}
		}
		sort.Strings(taskIDs)
	}

	rows := make([]statusRow, 0, len(taskIDs))
	for _, id := range taskIDs {
		row, err := buildStatusRow(root, projectID, id)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}

	if jsonOut {
		return encodeJSON(out, map[string]interface{}{"tasks": rows})
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK_ID\tSTATUS\tEXIT_CODE\tLATEST_RUN\tDONE\tPID_ALIVE\tBLOCKED_BY")
	for _, row := range rows {
		exitCode := "-"
		if row.ExitCode != nil {
			exitCode = fmt.Sprintf("%d", *row.ExitCode)
		}
		latestRun := row.LatestRun
		if latestRun == "" {
			latestRun = "-"
		}
		pidAlive := "-"
		if row.PIDAlive != nil {
			pidAlive = fmt.Sprintf("%t", *row.PIDAlive)
		}
		blockedBy := "-"
		if len(row.BlockedBy) > 0 {
			blockedBy = strings.Join(row.BlockedBy, ",")
		}
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%t\t%s\t%s\n",
			row.TaskID,
			row.Status,
			exitCode,
			latestRun,
			row.Done,
			pidAlive,
			blockedBy,
		)
	}
	return w.Flush()
}

func buildStatusRow(root, projectID, taskID string) (statusRow, error) {
	taskDir := filepath.Join(root, projectID, taskID)
	row := statusRow{
		TaskID: taskID,
		Status: "-",
	}
	dependsOn, err := taskdeps.ReadDependsOn(taskDir)
	if err != nil {
		return row, errors.Wrapf(err, "read task dependencies for task %s", taskID)
	}
	row.DependsOn = dependsOn

	if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
		row.Done = true
	} else if err != nil && !os.IsNotExist(err) {
		return row, errors.Wrapf(err, "stat DONE file for task %s", taskID)
	}

	runsDir := filepath.Join(taskDir, "runs")
	runEntries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			if row.Done {
				row.Status = "done"
			} else if len(dependsOn) > 0 {
				blockedBy, err := taskdeps.BlockedBy(root, projectID, dependsOn)
				if err != nil {
					return row, errors.Wrapf(err, "resolve blocked dependencies for task %s", taskID)
				}
				if len(blockedBy) > 0 {
					row.Status = "blocked"
					row.BlockedBy = blockedBy
				}
			}
			return row, nil
		}
		return row, errors.Wrapf(err, "read runs directory for task %s", taskID)
	}

	runNames := make([]string, 0, len(runEntries))
	for _, runEntry := range runEntries {
		if runEntry.IsDir() {
			runNames = append(runNames, runEntry.Name())
		}
	}

	if len(runNames) == 0 {
		if row.Done {
			row.Status = "done"
		} else if len(dependsOn) > 0 {
			blockedBy, err := taskdeps.BlockedBy(root, projectID, dependsOn)
			if err != nil {
				return row, errors.Wrapf(err, "resolve blocked dependencies for task %s", taskID)
			}
			if len(blockedBy) > 0 {
				row.Status = "blocked"
				row.BlockedBy = blockedBy
			}
		}
		return row, nil
	}

	sort.Strings(runNames)
	latest := runNames[len(runNames)-1]
	row.LatestRun = latest

	infoPath := filepath.Join(runsDir, latest, "run-info.yaml")
	info, err := runstate.ReadRunInfo(infoPath)
	if err != nil {
		row.Status = "unknown"
		return row, nil
	}

	status := strings.TrimSpace(info.Status)
	if status == "" {
		status = "unknown"
	}
	row.Status = status
	row.ExitCode = intPointer(info.ExitCode)
	row.PIDAlive = boolPointer(isRunPIDAlive(info))
	return row, nil
}

func isRunPIDAlive(info *storage.RunInfo) bool {
	if info == nil {
		return false
	}
	if strings.TrimSpace(info.Status) != storage.StatusRunning {
		return false
	}
	if info.PID > 0 && runner.IsProcessAlive(info.PID) {
		return true
	}
	if info.PGID > 0 {
		alive, err := runner.IsProcessGroupAlive(info.PGID)
		if err == nil && alive {
			return true
		}
	}
	return false
}

func intPointer(value int) *int {
	v := value
	return &v
}

func boolPointer(value bool) *bool {
	v := value
	return &v
}
