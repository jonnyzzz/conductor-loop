package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		root         string
		projectID    string
		taskID       string
		statusFilter string
		jsonOut      bool
		activityOut  bool
		driftAfter   time.Duration
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
			return runListWithOptions(
				cmd.OutOrStdout(),
				root,
				projectID,
				taskID,
				statusFilter,
				jsonOut,
				activityOptions{
					Enabled:    activityOut,
					DriftAfter: driftAfter,
				},
			)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id (optional; lists tasks if set)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (requires --project; lists runs if set)")
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter tasks by status: running, active, done, failed, blocked (only applies when --project is set)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&activityOut, "activity", false, "include recent activity signals for task rows (latest bus message, output activity, meaningful-signal age, analysis drift risk)")
	cmd.Flags().DurationVar(&driftAfter, "drift-after", defaultAnalysisDriftAfter, "mark analysis-drift risk when a running task has no meaningful bus signal for this duration (used with --activity)")

	return cmd
}

func runList(out io.Writer, root, projectID, taskID, statusFilter string, jsonOut bool) error {
	return runListWithOptions(out, root, projectID, taskID, statusFilter, jsonOut, activityOptions{})
}

func runListWithOptions(out io.Writer, root, projectID, taskID, statusFilter string, jsonOut bool, opts activityOptions) error {
	if projectID == "" && taskID != "" {
		return fmt.Errorf("--task requires --project")
	}

	switch {
	case projectID == "":
		return listProjects(out, root, jsonOut)
	case taskID == "":
		return listTasksWithOptions(out, root, projectID, statusFilter, jsonOut, opts)
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
	TaskID       string               `json:"task_id"`
	Runs         int                  `json:"runs"`
	LatestStatus string               `json:"latest_status"`
	Done         bool                 `json:"done"`
	LastActivity string               `json:"last_activity"` // ISO 8601 or ""
	DependsOn    []string             `json:"depends_on,omitempty"`
	BlockedBy    []string             `json:"blocked_by,omitempty"`
	Activity     *taskActivitySignals `json:"activity,omitempty"`

	latestRunID    string    `json:"-"`
	latestRunStart time.Time `json:"-"`
	latestOutput   string    `json:"-"`
	latestStdout   string    `json:"-"`
	latestStderr   string    `json:"-"`
}

// listTasks lists all tasks for a project as a table.
func listTasks(out io.Writer, root, projectID, statusFilter string, jsonOut bool) error {
	return listTasksWithOptions(out, root, projectID, statusFilter, jsonOut, activityOptions{})
}

func listTasksWithOptions(out io.Writer, root, projectID, statusFilter string, jsonOut bool, opts activityOptions) error {
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
		dependsOn, err := taskdeps.ReadDependsOn(taskDir)
		if err != nil {
			return errors.Wrapf(err, "read task dependencies for task %s", taskID)
		}
		row.DependsOn = dependsOn

		if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
			row.Done = true
		}

		// Record task dir modification time as last activity
		if info, err := e.Info(); err == nil {
			row.LastActivity = info.ModTime().UTC().Format(time.RFC3339)
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

		// Determine status: "-" when no runs, "done" when DONE file exists and no runs
		row.LatestStatus = "-"
		if row.Done && len(runNames) == 0 {
			row.LatestStatus = "done"
		}
		if len(runNames) > 0 {
			sort.Strings(runNames)
			latestRunID, info := latestReadableRunInfo(runsDir, runNames)
			if info != nil {
				row.latestRunID = latestRunID
				row.LatestStatus = info.Status
				row.latestRunStart = info.StartTime
				row.latestOutput = info.OutputPath
				row.latestStdout = info.StdoutPath
				row.latestStderr = info.StderrPath
			} else if row.Done {
				// A DONE marker is authoritative even if newest run directories are orphaned.
				row.LatestStatus = "done"
			}
		} else if !row.Done && len(dependsOn) > 0 {
			blockedBy, err := taskdeps.BlockedBy(root, projectID, dependsOn)
			if err != nil {
				return errors.Wrapf(err, "resolve blocked dependencies for task %s", taskID)
			}
			if len(blockedBy) > 0 {
				row.LatestStatus = "blocked"
				row.BlockedBy = blockedBy
			}
		}

		attachListTaskActivity(&row, taskDir, opts)
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].TaskID < rows[j].TaskID
	})

	rows = filterRowsByStatus(rows, statusFilter)

	if jsonOut {
		return encodeJSON(out, map[string]interface{}{"tasks": rows})
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if opts.Enabled {
		fmt.Fprintln(w, "TASK_ID\tRUNS\tLATEST_STATUS\tDONE\tBLOCKED_BY\tLAST_ACTIVITY\tLAST_BUS\tLAST_OUTPUT\tMEANINGFUL_AGE\tDRIFT_RISK\tDRIFT_REASON")
	} else {
		fmt.Fprintln(w, "TASK_ID\tRUNS\tLATEST_STATUS\tDONE\tBLOCKED_BY\tLAST_ACTIVITY")
	}
	for _, row := range rows {
		done := "-"
		if row.Done {
			done = "DONE"
		}
		blockedBy := "-"
		if len(row.BlockedBy) > 0 {
			blockedBy = strings.Join(row.BlockedBy, ",")
		}
		lastActivity := "-"
		if row.LastActivity != "" {
			if t, err := time.Parse(time.RFC3339, row.LastActivity); err == nil {
				lastActivity = t.Local().Format("2006-01-02 15:04")
			}
		}
		if opts.Enabled {
			fmt.Fprintf(
				w,
				"%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.TaskID,
				row.Runs,
				row.LatestStatus,
				done,
				blockedBy,
				lastActivity,
				formatActivityBusSummary(listActivityBus(row.Activity)),
				formatActivityTimestamp(listActivityOutputTimestamp(row.Activity)),
				formatActivityAge(listActivityAge(row.Activity)),
				formatActivityRiskText(row.Activity),
				listActivityReason(row.Activity),
			)
			continue
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n", row.TaskID, row.Runs, row.LatestStatus, done, blockedBy, lastActivity)
	}
	return w.Flush()
}

func attachListTaskActivity(row *taskRow, taskDir string, opts activityOptions) {
	if row == nil || !opts.Enabled {
		return
	}
	signals := collectTaskActivitySignals(
		taskDir,
		row.latestRunID,
		row.LatestStatus,
		row.latestRunStart,
		row.latestOutput,
		row.latestStdout,
		row.latestStderr,
		opts,
	)
	row.Activity = &signals
}

func latestReadableRunInfo(runsDir string, runNames []string) (string, *storage.RunInfo) {
	for i := len(runNames) - 1; i >= 0; i-- {
		runID := runNames[i]
		infoPath := filepath.Join(runsDir, runID, "run-info.yaml")
		info, err := runstate.ReadRunInfo(infoPath)
		if err != nil {
			continue
		}
		return runID, info
	}
	return "", nil
}

func listActivityAge(signals *taskActivitySignals) *int64 {
	if signals == nil {
		return nil
	}
	return signals.MeaningfulSignalAgeSeconds
}

func listActivityBus(signals *taskActivitySignals) *activityBusMessage {
	if signals == nil {
		return nil
	}
	return signals.LatestBusMessage
}

func listActivityOutputTimestamp(signals *taskActivitySignals) *string {
	if signals == nil {
		return nil
	}
	return signals.LatestOutputActivityAt
}

func listActivityReason(signals *taskActivitySignals) string {
	if signals == nil {
		return "-"
	}
	return safeField(signals.DriftReason)
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

		info, err := runstate.ReadRunInfo(infoPath)
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

// filterRowsByStatus filters task rows by status. Supported values: "running"/"active", "done", "failed", "blocked".
// Unknown values log a warning and return all rows unchanged.
func filterRowsByStatus(rows []taskRow, filter string) []taskRow {
	switch strings.ToLower(filter) {
	case "":
		return rows
	case "running", "active":
		var out []taskRow
		for _, r := range rows {
			if r.LatestStatus == "running" {
				out = append(out, r)
			}
		}
		return out
	case "done":
		var out []taskRow
		for _, r := range rows {
			if r.Done {
				out = append(out, r)
			}
		}
		return out
	case "failed":
		var out []taskRow
		for _, r := range rows {
			if r.LatestStatus == "failed" {
				out = append(out, r)
			}
		}
		return out
	case "blocked":
		var out []taskRow
		for _, r := range rows {
			if r.LatestStatus == "blocked" {
				out = append(out, r)
			}
		}
		return out
	default:
		fmt.Fprintf(os.Stderr, "warning: unknown --status filter %q; showing all tasks\n", filter)
		return rows
	}
}

func encodeJSON(out io.Writer, v interface{}) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
