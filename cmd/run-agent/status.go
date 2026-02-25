package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var (
		root         string
		projectID    string
		taskID       string
		statusFilter string
		jsonOut      bool
		conciseOut   bool
		activityOut  bool
		driftAfter   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show latest task run status for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			root, err = config.ResolveRunsDir(root)
			if err != nil {
				return fmt.Errorf("resolve runs dir: %w", err)
			}
			return runStatusWithOptions(
				cmd.OutOrStdout(),
				root,
				projectID,
				taskID,
				statusFilter,
				jsonOut,
				conciseOut,
				activityOptions{
					Enabled:    activityOut,
					DriftAfter: driftAfter,
				},
			)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ~/.run-agent/runs)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (optional; defaults to all tasks in project)")
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter rows by status: running, active, completed, failed, blocked, done, pending")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&conciseOut, "concise", false, "output concise tab-separated rows: task_id status exit_code latest_run done pid_alive")
	cmd.Flags().BoolVar(&activityOut, "activity", false, "include recent activity signals (latest bus message, output activity, meaningful-signal age, analysis drift risk)")
	cmd.Flags().DurationVar(&driftAfter, "drift-after", defaultAnalysisDriftAfter, "mark analysis-drift risk when a running task has no meaningful bus signal for this duration (used with --activity)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

type statusRow struct {
	TaskID    string               `json:"task_id"`
	Status    string               `json:"status"`
	ExitCode  *int                 `json:"exit_code"`
	LatestRun string               `json:"latest_run"`
	Done      bool                 `json:"done"`
	PIDAlive  *bool                `json:"pid_alive"`
	DependsOn []string             `json:"depends_on,omitempty"`
	BlockedBy []string             `json:"blocked_by,omitempty"`
	Activity  *taskActivitySignals `json:"activity,omitempty"`

	latestRunStart time.Time `json:"-"`
	latestOutput   string    `json:"-"`
	latestStdout   string    `json:"-"`
	latestStderr   string    `json:"-"`
}

func runStatus(out io.Writer, root, projectID, taskID, statusFilter string, jsonOut, conciseOut bool) error {
	return runStatusWithOptions(out, root, projectID, taskID, statusFilter, jsonOut, conciseOut, activityOptions{})
}

func runStatusWithOptions(out io.Writer, root, projectID, taskID, statusFilter string, jsonOut, conciseOut bool, opts activityOptions) error {
	projectID = strings.TrimSpace(projectID)
	taskID = strings.TrimSpace(taskID)
	if projectID == "" {
		return fmt.Errorf("--project is required")
	}
	if jsonOut && conciseOut {
		return fmt.Errorf("--concise cannot be used with --json")
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
		row, err := buildStatusRowWithOptions(root, projectID, id, opts)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}

	rows = filterStatusRows(rows, statusFilter)

	if jsonOut {
		return encodeJSON(out, map[string]interface{}{"tasks": rows})
	}

	if conciseOut {
		if len(rows) == 0 {
			_, err := fmt.Fprintln(out, statusEmptyMessage(projectID, taskID, statusFilter))
			return err
		}
		for _, row := range rows {
			if opts.Enabled {
				fmt.Fprintf(
					out,
					"%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					row.TaskID,
					row.Status,
					statusExitCode(row.ExitCode),
					statusLatestRun(row.LatestRun),
					row.Done,
					statusPIDAlive(row.PIDAlive),
					formatActivityAge(statusActivityAge(row.Activity)),
					formatActivityRiskFlag(row.Activity),
					statusActivityBusType(row.Activity),
					statusActivityBusTimestamp(row.Activity),
					statusActivityBusPreview(row.Activity),
					statusActivityOutputRaw(row.Activity),
				)
				continue
			}
			fmt.Fprintf(
				out,
				"%s\t%s\t%s\t%s\t%t\t%s\n",
				row.TaskID,
				row.Status,
				statusExitCode(row.ExitCode),
				statusLatestRun(row.LatestRun),
				row.Done,
				statusPIDAlive(row.PIDAlive),
			)
		}
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if opts.Enabled {
		fmt.Fprintln(w, "TASK_ID\tSTATUS\tEXIT_CODE\tLATEST_RUN\tDONE\tPID_ALIVE\tBLOCKED_BY\tLAST_BUS\tLAST_OUTPUT\tMEANINGFUL_AGE\tDRIFT_RISK\tDRIFT_REASON")
	} else {
		fmt.Fprintln(w, "TASK_ID\tSTATUS\tEXIT_CODE\tLATEST_RUN\tDONE\tPID_ALIVE\tBLOCKED_BY")
	}
	for _, row := range rows {
		blockedBy := "-"
		if len(row.BlockedBy) > 0 {
			blockedBy = strings.Join(row.BlockedBy, ",")
		}
		if opts.Enabled {
			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.TaskID,
				row.Status,
				statusExitCode(row.ExitCode),
				statusLatestRun(row.LatestRun),
				row.Done,
				statusPIDAlive(row.PIDAlive),
				blockedBy,
				formatActivityBusSummary(statusActivityBus(row.Activity)),
				formatActivityTimestamp(statusActivityOutputTimestamp(row.Activity)),
				formatActivityAge(statusActivityAge(row.Activity)),
				formatActivityRiskText(row.Activity),
				statusActivityReason(row.Activity),
			)
			continue
		}
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%t\t%s\t%s\n",
			row.TaskID,
			row.Status,
			statusExitCode(row.ExitCode),
			statusLatestRun(row.LatestRun),
			row.Done,
			statusPIDAlive(row.PIDAlive),
			blockedBy,
		)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if len(rows) == 0 && strings.TrimSpace(statusFilter) != "" {
		_, err := fmt.Fprintln(out, statusEmptyMessage(projectID, taskID, statusFilter))
		return err
	}
	return nil
}

func filterStatusRows(rows []statusRow, statusFilter string) []statusRow {
	normalized := strings.ToLower(strings.TrimSpace(statusFilter))
	if normalized == "" {
		return rows
	}
	filtered := make([]statusRow, 0, len(rows))
	for _, row := range rows {
		if statusRowMatchesFilter(row, normalized) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func statusRowMatchesFilter(row statusRow, statusFilter string) bool {
	switch statusFilter {
	case "active":
		return strings.EqualFold(row.Status, storage.StatusRunning)
	case "done", "all_finished":
		return row.Done
	case "pending":
		return row.Status == "-"
	case "partial_failure":
		return strings.EqualFold(row.Status, storage.StatusPartialFail)
	default:
		return strings.EqualFold(row.Status, statusFilter)
	}
}

func statusEmptyMessage(projectID, taskID, statusFilter string) string {
	projectID = strings.TrimSpace(projectID)
	taskID = strings.TrimSpace(taskID)
	statusFilter = strings.TrimSpace(statusFilter)

	if statusFilter != "" {
		if taskID != "" {
			return fmt.Sprintf("No status rows matched --status %q for task %s.", statusFilter, taskID)
		}
		return fmt.Sprintf("No status rows matched --status %q in project %s.", statusFilter, projectID)
	}
	if taskID != "" {
		return fmt.Sprintf("No status rows found for task %s.", taskID)
	}
	return fmt.Sprintf("No status rows found in project %s.", projectID)
}

func statusExitCode(code *int) string {
	if code == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *code)
}

func statusLatestRun(runID string) string {
	if runID == "" {
		return "-"
	}
	return runID
}

func statusPIDAlive(alive *bool) string {
	if alive == nil {
		return "-"
	}
	return fmt.Sprintf("%t", *alive)
}

func buildStatusRowWithOptions(root, projectID, taskID string, opts activityOptions) (statusRow, error) {
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
			attachStatusActivity(&row, taskDir, opts)
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
		attachStatusActivity(&row, taskDir, opts)
		return row, nil
	}

	sort.Strings(runNames)
	latest := runNames[len(runNames)-1]
	row.LatestRun = latest

	infoPath := filepath.Join(runsDir, latest, "run-info.yaml")
	info, err := runstate.ReadRunInfo(infoPath)
	if err != nil {
		row.Status = "unknown"
		attachStatusActivity(&row, taskDir, opts)
		return row, nil
	}

	status := strings.TrimSpace(info.Status)
	if status == "" {
		status = "unknown"
	}
	row.Status = status
	row.ExitCode = intPointer(info.ExitCode)
	row.PIDAlive = boolPointer(isRunPIDAlive(info))
	row.latestRunStart = info.StartTime
	row.latestOutput = info.OutputPath
	row.latestStdout = info.StdoutPath
	row.latestStderr = info.StderrPath
	attachStatusActivity(&row, taskDir, opts)
	return row, nil
}

func attachStatusActivity(row *statusRow, taskDir string, opts activityOptions) {
	if row == nil || !opts.Enabled {
		return
	}
	signals := collectTaskActivitySignals(taskDir, row.LatestRun, row.Status, row.latestRunStart, row.latestOutput, row.latestStdout, row.latestStderr, opts)
	row.Activity = &signals
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

func statusActivityAge(signals *taskActivitySignals) *int64 {
	if signals == nil {
		return nil
	}
	return signals.MeaningfulSignalAgeSeconds
}

func statusActivityBus(signals *taskActivitySignals) *activityBusMessage {
	if signals == nil {
		return nil
	}
	return signals.LatestBusMessage
}

func statusActivityBusType(signals *taskActivitySignals) string {
	msg := statusActivityBus(signals)
	if msg == nil {
		return "-"
	}
	return safeField(msg.Type)
}

func statusActivityBusTimestamp(signals *taskActivitySignals) string {
	msg := statusActivityBus(signals)
	if msg == nil {
		return "-"
	}
	return safeField(msg.Timestamp)
}

func statusActivityBusPreview(signals *taskActivitySignals) string {
	msg := statusActivityBus(signals)
	if msg == nil {
		return "-"
	}
	return safeField(msg.BodyPreview)
}

func statusActivityOutputTimestamp(signals *taskActivitySignals) *string {
	if signals == nil {
		return nil
	}
	return signals.LatestOutputActivityAt
}

func statusActivityOutputRaw(signals *taskActivitySignals) string {
	ts := statusActivityOutputTimestamp(signals)
	if ts == nil {
		return "-"
	}
	return safeField(*ts)
}

func statusActivityReason(signals *taskActivitySignals) string {
	if signals == nil {
		return "-"
	}
	return safeField(signals.DriftReason)
}

func intPointer(value int) *int {
	v := value
	return &v
}

func boolPointer(value bool) *bool {
	v := value
	return &v
}
