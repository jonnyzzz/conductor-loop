package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		root      string
		projectID string
		taskID    string
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects, tasks, and runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if root == "" {
				if v := os.Getenv("RUNS_DIR"); v != "" {
					root = v
				} else {
					root = "./runs"
				}
			}
			return runList(cmd.OutOrStdout(), root, projectID, taskID, jsonOut)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id (optional; lists tasks if set)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (requires --project; lists runs if set)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")

	return cmd
}

func runList(out io.Writer, root, projectID, taskID string, jsonOut bool) error {
	if projectID == "" && taskID != "" {
		return fmt.Errorf("--task requires --project")
	}

	switch {
	case projectID == "":
		return listProjects(out, root, jsonOut)
	case taskID == "":
		return listTasks(out, root, projectID, jsonOut)
	default:
		return listRuns(out, root, projectID, taskID, jsonOut)
	}
}

// listProjects lists all projects in the root directory.
func listProjects(out io.Writer, root string, jsonOut bool) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("root directory not found: %s", root)
		}
		return errors.Wrap(err, "read root directory")
	}

	var projects []string
	for _, e := range entries {
		if e.IsDir() {
			projects = append(projects, e.Name())
		}
	}
	sort.Strings(projects)

	if jsonOut {
		return encodeJSON(out, map[string]interface{}{"projects": projects})
	}

	for _, p := range projects {
		fmt.Fprintln(out, p)
	}
	return nil
}

// taskRow holds summary data for a task.
type taskRow struct {
	TaskID       string `json:"task_id"`
	Runs         int    `json:"runs"`
	LatestStatus string `json:"latest_status"`
	Done         bool   `json:"done"`
}

// listTasks lists all tasks for a project as a table.
func listTasks(out io.Writer, root, projectID string, jsonOut bool) error {
	projDir := filepath.Join(root, projectID)
	entries, err := os.ReadDir(projDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project directory not found: %s", projDir)
		}
		return errors.Wrap(err, "read project directory")
	}

	var rows []taskRow
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		taskID := e.Name()
		taskDir := filepath.Join(projDir, taskID)
		row := taskRow{TaskID: taskID}

		if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
			row.Done = true
		}

		runsDir := filepath.Join(taskDir, "runs")
		runEntries, err := os.ReadDir(runsDir)
		if err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "read runs directory for task %s", taskID)
		}

		var runNames []string
		for _, re := range runEntries {
			if re.IsDir() {
				runNames = append(runNames, re.Name())
			}
		}
		row.Runs = len(runNames)

		row.LatestStatus = "unknown"
		if len(runNames) > 0 {
			sort.Strings(runNames)
			latest := runNames[len(runNames)-1]
			infoPath := filepath.Join(runsDir, latest, "run-info.yaml")
			if info, err := storage.ReadRunInfo(infoPath); err == nil {
				row.LatestStatus = info.Status
			}
		}

		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].TaskID < rows[j].TaskID
	})

	if jsonOut {
		return encodeJSON(out, map[string]interface{}{"tasks": rows})
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK_ID\tRUNS\tLATEST_STATUS\tDONE")
	for _, row := range rows {
		done := "-"
		if row.Done {
			done = "DONE"
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", row.TaskID, row.Runs, row.LatestStatus, done)
	}
	return w.Flush()
}

// runRow holds summary data for a run.
type runRow struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Started  string `json:"started"`
	Duration string `json:"duration"`
}

// listRuns lists all runs for a task as a table.
func listRuns(out io.Writer, root, projectID, taskID string, jsonOut bool) error {
	runsDir := filepath.Join(root, projectID, taskID, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no runs found for project %s task %s", projectID, taskID)
		}
		return errors.Wrap(err, "read runs directory")
	}

	var rows []runRow
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		runID := e.Name()
		infoPath := filepath.Join(runsDir, runID, "run-info.yaml")
		row := runRow{RunID: runID}

		info, err := storage.ReadRunInfo(infoPath)
		if err != nil {
			row.Status = "unknown"
			row.Duration = "-"
		} else {
			row.Status = info.Status
			row.ExitCode = info.ExitCode
			if !info.StartTime.IsZero() {
				row.Started = info.StartTime.Format("2006-01-02 15:04:05")
				if info.Status == storage.StatusRunning || info.EndTime.IsZero() {
					row.Duration = "running"
				} else {
					dur := info.EndTime.Sub(info.StartTime).Round(time.Second)
					row.Duration = dur.String()
				}
			}
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].RunID < rows[j].RunID
	})

	if jsonOut {
		return encodeJSON(out, map[string]interface{}{"runs": rows})
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUN_ID\tSTATUS\tEXIT_CODE\tSTARTED\tDURATION")
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", row.RunID, row.Status, row.ExitCode, row.Started, row.Duration)
	}
	return w.Flush()
}

func encodeJSON(out io.Writer, v interface{}) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
