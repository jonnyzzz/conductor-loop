package main

// server.go — "run-agent server" subcommand group.
//
// All subcommands here are API clients that talk to a running run-agent server
// (started with "run-agent serve"). They require the server to be running.
//
// Contrast with the top-level run-agent commands (task, job, bus, list, watch,
// etc.) which operate entirely on the local filesystem without any server.

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

const defaultServerURL = "http://localhost:14355"

// newServerCmd returns the "run-agent server" subcommand group.
func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage and query a running run-agent server (requires run-agent serve)",
		Long: `Commands under "run-agent server" are API clients that talk to a running
run-agent server. Start the server with "run-agent serve" first.

All other run-agent commands (task, job, bus, list, watch, gc, etc.) work
directly on the local filesystem and do NOT require the server to be running.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newServerStatusCmd())
	cmd.AddCommand(newServerTaskCmd())
	cmd.AddCommand(newServerJobCmd())
	cmd.AddCommand(newServerProjectCmd())
	cmd.AddCommand(newServerWatchCmd())
	cmd.AddCommand(newServerBusCmd())
	cmd.AddCommand(newServerUpdateCmd())

	return cmd
}

// ─── status ───────────────────────────────────────────────────────────────────

type serverStatusResponse struct {
	ActiveRunsCount  int                 `json:"active_runs_count"`
	UptimeSeconds    float64             `json:"uptime_seconds"`
	ConfiguredAgents []string            `json:"configured_agents"`
	Version          string              `json:"version"`
	RunningTasks     []serverRunningTask `json:"running_tasks,omitempty"`
}

type serverRunningTask struct {
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Agent     string    `json:"agent"`
	Started   time.Time `json:"started"`
}

func newServerStatusCmd() *cobra.Command {
	var (
		serverURL  string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show run-agent server status",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(serverURL + "/api/v1/status") //nolint:noctx
			if err != nil {
				return fmt.Errorf("get status: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverStatusResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Version:\t%s\n", result.Version)
			fmt.Fprintf(w, "Uptime:\t%s\n", serverFormatUptime(result.UptimeSeconds))
			fmt.Fprintf(w, "Active Runs:\t%d\n", result.ActiveRunsCount)
			agents := strings.Join(result.ConfiguredAgents, ", ")
			if agents == "" {
				agents = "(none)"
			}
			fmt.Fprintf(w, "Configured Agents:\t%s\n", agents)
			if err := w.Flush(); err != nil {
				return err
			}

			if len(result.RunningTasks) > 0 {
				fmt.Println()
				fmt.Println("Running tasks:")
				tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintf(tw, "  PROJECT\tTASK\tRUN\tAGENT\tSTARTED\n")
				for _, task := range result.RunningTasks {
					runID := task.RunID
					if len(runID) > 20 {
						runID = runID[:20] + "..."
					}
					started := "-"
					if !task.Started.IsZero() {
						started = task.Started.Local().Format("15:04:05")
					}
					fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
						task.ProjectID, task.TaskID, runID, task.Agent, started)
				}
				if err := tw.Flush(); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

func serverFormatUptime(seconds float64) string {
	d := time.Duration(math.Round(seconds)) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ─── update ──────────────────────────────────────────────────────────────────

type serverSelfUpdateStatus struct {
	State               string    `json:"state"`
	BinaryPath          string    `json:"binary_path,omitempty"`
	RequestedAt         time.Time `json:"requested_at,omitempty"`
	StartedAt           time.Time `json:"started_at,omitempty"`
	FinishedAt          time.Time `json:"finished_at,omitempty"`
	ActiveRunsAtRequest int       `json:"active_runs_at_request,omitempty"`
	ActiveRunsNow       int       `json:"active_runs_now"`
	ActiveRunsError     string    `json:"active_runs_error,omitempty"`
	LastError           string    `json:"last_error,omitempty"`
	LastNote            string    `json:"last_note,omitempty"`
}

type serverSelfUpdateRequest struct {
	BinaryPath string `json:"binary_path"`
}

func newServerUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Request or inspect safe server self-update",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newServerUpdateStatusCmd())
	cmd.AddCommand(newServerUpdateStartCmd())
	return cmd
}

func newServerUpdateStatusCmd() *cobra.Command {
	var (
		serverURL  string
		jsonOutput bool
	)
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current server self-update state",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(serverURL + "/api/v1/admin/self-update") //nolint:noctx
			if err != nil {
				return fmt.Errorf("get self-update status: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}
			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var status serverSelfUpdateStatus
			if err := json.Unmarshal(data, &status); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "State:\t%s\n", status.State)
			fmt.Fprintf(w, "Active Root Runs:\t%d\n", status.ActiveRunsNow)
			if status.ActiveRunsError != "" {
				fmt.Fprintf(w, "Active Run Error:\t%s\n", status.ActiveRunsError)
			}
			if status.BinaryPath != "" {
				fmt.Fprintf(w, "Candidate Binary:\t%s\n", status.BinaryPath)
			}
			if !status.RequestedAt.IsZero() {
				fmt.Fprintf(w, "Requested At:\t%s\n", status.RequestedAt.Format(time.RFC3339))
			}
			if !status.StartedAt.IsZero() {
				fmt.Fprintf(w, "Started At:\t%s\n", status.StartedAt.Format(time.RFC3339))
			}
			if !status.FinishedAt.IsZero() {
				fmt.Fprintf(w, "Finished At:\t%s\n", status.FinishedAt.Format(time.RFC3339))
			}
			if status.LastError != "" {
				fmt.Fprintf(w, "Last Error:\t%s\n", status.LastError)
			}
			if status.LastNote != "" {
				fmt.Fprintf(w, "Last Note:\t%s\n", status.LastNote)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	return cmd
}

func newServerUpdateStartCmd() *cobra.Command {
	var (
		serverURL  string
		binaryPath string
		jsonOutput bool
	)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Request a safe server self-update to a new binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := serverSelfUpdateRequest{BinaryPath: binaryPath}
			reqBody, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("encode request: %w", err)
			}
			resp, err := http.Post(serverURL+"/api/v1/admin/self-update", "application/json", bytes.NewReader(reqBody)) //nolint:noctx
			if err != nil {
				return fmt.Errorf("request self-update: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}
			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var status serverSelfUpdateStatus
			if err := json.Unmarshal(data, &status); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
			fmt.Printf("Self-update state: %s\n", status.State)
			if status.State == "deferred" {
				fmt.Printf("Update is waiting for %d active root run(s) to finish.\n", status.ActiveRunsNow)
			} else {
				fmt.Println("Update handoff has started.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&binaryPath, "binary", "", "path to the candidate run-agent binary")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "binary") //nolint:errcheck
	return cmd
}

// ─── task ─────────────────────────────────────────────────────────────────────

type serverTaskDetail struct {
	ProjectID    string               `json:"project_id"`
	TaskID       string               `json:"task_id"`
	Status       string               `json:"status"`
	LastActivity time.Time            `json:"last_activity"`
	DependsOn    []string             `json:"depends_on,omitempty"`
	BlockedBy    []string             `json:"blocked_by,omitempty"`
	Runs         []serverTaskRunEntry `json:"runs"`
}

type serverTaskRunEntry struct {
	RunID     string    `json:"run_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	ExitCode  int       `json:"exit_code"`
}

type serverTaskListItem struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Status       string    `json:"status"`
	LastActivity time.Time `json:"last_activity"`
	RunCount     int       `json:"run_count"`
	BlockedBy    []string  `json:"blocked_by,omitempty"`
}

type serverTaskListResponse struct {
	Items   []serverTaskListItem `json:"items"`
	Total   int                  `json:"total"`
	HasMore bool                 `json:"has_more"`
}

type serverTaskStopResponse struct {
	StoppedRuns int `json:"stopped_runs"`
}

type serverTaskResumeResponse struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
	Resumed   bool   `json:"resumed"`
}

func newServerTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks via the server API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newServerTaskStatusCmd())
	cmd.AddCommand(newServerTaskStopCmd())
	cmd.AddCommand(newServerTaskListCmd())
	cmd.AddCommand(newServerTaskDeleteCmd())
	cmd.AddCommand(newServerTaskResumeCmd())
	cmd.AddCommand(newServerTaskLogsCmd())
	cmd.AddCommand(newServerTaskRunsCmd())
	return cmd
}

func newServerTaskStatusCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "status <task-id>",
		Short: "Get the status of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			url := serverURL + "/api/v1/tasks/" + taskID
			if project != "" {
				url += "?project_id=" + project
			}

			resp, err := http.Get(url) //nolint:noctx
			if err != nil {
				return fmt.Errorf("get task: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverTaskDetail
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			fmt.Printf("Task:   %s/%s\n", result.ProjectID, result.TaskID)
			fmt.Printf("Status: %s\n", result.Status)
			if len(result.DependsOn) > 0 {
				fmt.Printf("Depends on: %s\n", strings.Join(result.DependsOn, ", "))
			}
			if len(result.BlockedBy) > 0 {
				fmt.Printf("Blocked by: %s\n", strings.Join(result.BlockedBy, ", "))
			}
			if !result.LastActivity.IsZero() {
				fmt.Printf("Last activity: %s\n", result.LastActivity.Format(time.RFC3339))
			}

			if len(result.Runs) > 0 {
				fmt.Println()
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "RUN ID\tSTATUS\tSTART TIME\tEND TIME\tEXIT CODE")
				for _, run := range result.Runs {
					endTime := "-"
					if !run.EndTime.IsZero() {
						endTime = run.EndTime.Format(time.RFC3339)
					}
					exitCode := "-"
					if !run.EndTime.IsZero() {
						exitCode = fmt.Sprintf("%d", run.ExitCode)
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						run.RunID, run.Status,
						run.StartTime.Format(time.RFC3339),
						endTime, exitCode)
				}
				w.Flush() //nolint:errcheck
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

func newServerTaskStopCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "stop <task-id>",
		Short: "Stop all running runs of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			url := serverURL + "/api/v1/tasks/" + taskID
			if project != "" {
				url += "?project_id=" + project
			}

			req, err := http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("stop task: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverTaskStopResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			fmt.Printf("Task %s: stopped %d run(s)\n", taskID, result.StoppedRuns)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

func newServerTaskListCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		status     string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := serverURL + "/api/projects/" + project + "/tasks"
			if status != "" {
				url += "?status=" + status
			}

			resp, err := http.Get(url) //nolint:noctx
			if err != nil {
				return fmt.Errorf("get tasks: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverTaskListResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TASK ID\tSTATUS\tRUNS\tBLOCKED_BY\tLAST ACTIVITY")
			for _, t := range result.Items {
				lastActivity := "-"
				if !t.LastActivity.IsZero() {
					lastActivity = t.LastActivity.Format("2006-01-02 15:04")
				}
				blockedBy := "-"
				if len(t.BlockedBy) > 0 {
					blockedBy = strings.Join(t.BlockedBy, ",")
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", t.ID, t.Status, t.RunCount, blockedBy, lastActivity)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if result.HasMore {
				fmt.Printf("(showing %d of %d tasks; use --limit to see more)\n", len(result.Items), result.Total)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&status, "status", "", "filter by status: running, active, done, failed, blocked")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func newServerTaskDeleteCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Delete a task and all its runs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			url := serverURL + "/api/projects/" + project + "/tasks/" + taskID

			req, err := http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("delete task: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}

			switch resp.StatusCode {
			case http.StatusNoContent:
				if jsonOutput {
					fmt.Printf(`{"task_id":%q,"deleted":true}`+"\n", taskID)
				} else {
					fmt.Printf("Task %s deleted.\n", taskID)
				}
				return nil
			case http.StatusConflict:
				return fmt.Errorf("task %s has running runs; stop them first", taskID)
			case http.StatusNotFound:
				return fmt.Errorf("task %s not found in project %s", taskID, project)
			default:
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func newServerTaskResumeCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "resume <task-id>",
		Short: "Resume an exhausted task by removing its DONE file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			url := serverURL + "/api/projects/" + project + "/tasks/" + taskID + "/resume"

			req, err := http.NewRequest(http.MethodPost, url, nil)
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("resume task: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}

			switch resp.StatusCode {
			case http.StatusOK:
				// success
			case http.StatusNotFound:
				return fmt.Errorf("task %s not found in project %s", taskID, project)
			case http.StatusBadRequest:
				return fmt.Errorf("cannot resume task %s: %s", taskID, strings.TrimSpace(string(data)))
			default:
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverTaskResumeResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			fmt.Printf("Task %s/%s resumed (DONE file removed)\n", result.ProjectID, result.TaskID)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// ─── task logs ────────────────────────────────────────────────────────────────

func newServerTaskLogsCmd() *cobra.Command {
	var (
		serverURL string
		project   string
		runID     string
		follow    bool
		tail      int
	)

	cmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "Stream task output via the server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return serverTaskLogs(cmd.OutOrStdout(), serverURL, project, args[0], runID, follow, tail)
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&runID, "run", "", "specific run ID (default: latest)")
	cmd.Flags().BoolVar(&follow, "follow", false, "keep streaming; reconnect if connection drops")
	cmd.Flags().IntVar(&tail, "tail", 0, "output last N lines only (0 = all)")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func serverResolveLatestRunID(serverURL, project, taskID string) (string, error) {
	url := serverURL + "/api/projects/" + project + "/tasks/" + taskID

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("task logs: fetch task: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("task logs: read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("task logs: task %s not found in project %s", taskID, project)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task logs: server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var detail struct {
		Runs []struct {
			RunID  string `json:"run_id"`
			Status string `json:"status"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return "", fmt.Errorf("task logs: decode response: %w", err)
	}

	if len(detail.Runs) == 0 {
		return "", fmt.Errorf("task logs: no runs found for task %s", taskID)
	}

	chosen := detail.Runs[len(detail.Runs)-1].RunID
	for i := len(detail.Runs) - 1; i >= 0; i-- {
		if detail.Runs[i].Status == "running" {
			chosen = detail.Runs[i].RunID
			break
		}
	}
	return chosen, nil
}

func serverTaskLogs(out io.Writer, serverURL, project, taskID, runID string, follow bool, tail int) error {
	if runID == "" {
		var err error
		runID, err = serverResolveLatestRunID(serverURL, project, taskID)
		if err != nil {
			return err
		}
	}

	streamURL := serverURL + "/api/projects/" + project + "/tasks/" + taskID + "/runs/" + runID + "/stream?name=stdout"

	retryWait := 2 * time.Second
	const maxRetryWait = 30 * time.Second

	var allLines []string

	for {
		done, lines, err := serverStreamSSELines(out, streamURL, tail > 0)
		if tail > 0 {
			allLines = append(allLines, lines...)
		}

		if done {
			if tail > 0 {
				serverPrintTailLines(out, allLines, tail)
			}
			return nil
		}

		if err != nil && !follow {
			return fmt.Errorf("task logs: %w", err)
		}

		if !follow {
			return nil
		}

		time.Sleep(retryWait)
		retryWait *= 2
		if retryWait > maxRetryWait {
			retryWait = maxRetryWait
		}
	}
}

func serverStreamSSELines(out io.Writer, url string, buffer bool) (done bool, lines []string, err error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return false, nil, fmt.Errorf("not found: %s", strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	var (
		currentEvent string
		currentData  []string
	)

	flush := func() (isDone bool) {
		event := currentEvent
		data := currentData
		currentEvent = ""
		currentData = currentData[:0]

		switch event {
		case "done":
			return true
		case "heartbeat", "error":
			return false
		}

		for _, line := range data {
			if buffer {
				lines = append(lines, line)
			} else {
				fmt.Fprintln(out, line)
			}
		}
		return false
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if flush() {
				return true, lines, nil
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentData = append(currentData, strings.TrimPrefix(line, "data: "))
		}
	}

	if serr := scanner.Err(); serr != nil {
		return false, lines, serr
	}
	return false, lines, nil
}

func serverPrintTailLines(out io.Writer, lines []string, n int) {
	if n <= 0 || len(lines) == 0 {
		return
	}
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for _, line := range lines[start:] {
		fmt.Fprintln(out, line)
	}
}

// ─── task runs ────────────────────────────────────────────────────────────────

type serverRunListItem struct {
	ID           string     `json:"id"`
	Agent        string     `json:"agent"`
	AgentVersion string     `json:"agent_version"`
	Status       string     `json:"status"`
	ExitCode     int        `json:"exit_code"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	ErrorSummary string     `json:"error_summary"`
}

type serverRunsListResponse struct {
	Items   []serverRunListItem `json:"items"`
	Total   int                 `json:"total"`
	HasMore bool                `json:"has_more"`
}

func newServerTaskRunsCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "runs <task-id>",
		Short: "List all runs for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/runs?limit=%d", serverURL, project, taskID, limit)

			resp, err := http.Get(url) //nolint:noctx
			if err != nil {
				return fmt.Errorf("get runs: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("task %s not found in project %s", taskID, project)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverRunsListResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			if len(result.Items) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no runs found for task %s\n", taskID)
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "RUN ID\tAGENT\tSTATUS\tEXIT\tDURATION\tSTARTED\tERROR")
			for _, run := range result.Items {
				errSummary := run.ErrorSummary
				if len(errSummary) > 40 {
					errSummary = errSummary[:40]
				}
				started := run.StartTime.Format("2006-01-02 15:04:05")
				exitCode := "-"
				if run.EndTime != nil {
					exitCode = fmt.Sprintf("%d", run.ExitCode)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					run.ID, run.Agent, run.Status, exitCode,
					serverFormatRunDuration(run.StartTime, run.EndTime),
					started, errSummary)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if result.HasMore {
				fmt.Fprintf(cmd.OutOrStdout(), "(showing %d of %d runs; use --limit to see more)\n", len(result.Items), result.Total)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of runs to show")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func serverFormatRunDuration(start time.Time, end *time.Time) string {
	if end == nil {
		return "running"
	}
	d := end.Sub(start)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// ─── job ──────────────────────────────────────────────────────────────────────

type serverJobCreateRequest struct {
	ProjectID   string   `json:"project_id"`
	TaskID      string   `json:"task_id"`
	AgentType   string   `json:"agent_type"`
	Prompt      string   `json:"prompt"`
	ProjectRoot string   `json:"project_root,omitempty"`
	AttachMode  string   `json:"attach_mode,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

type serverJobCreateResponse struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
}

type serverJobRunResponse struct {
	RunID     string    `json:"run_id"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	ExitCode  int       `json:"exit_code"`
}

type serverJobTaskResponse struct {
	ProjectID    string    `json:"project_id"`
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"`
	LastActivity time.Time `json:"last_activity"`
}

type serverJobTaskListResponse struct {
	Tasks []serverJobTaskResponse `json:"tasks"`
}

func newServerJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Manage jobs via the server API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newServerJobSubmitCmd())
	cmd.AddCommand(newServerJobListCmd())
	return cmd
}

func serverGenerateTaskID() string {
	now := time.Now().UTC()
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		ns := now.UnixNano()
		b[0] = byte(ns)
		b[1] = byte(ns >> 8)
		b[2] = byte(ns >> 16)
	}
	return fmt.Sprintf("task-%s-%s-%s",
		now.Format("20060102"),
		now.Format("150405"),
		hex.EncodeToString(b[:]),
	)
}

func serverLoadPrompt(promptText, promptFile string) (string, error) {
	switch {
	case promptText != "" && promptFile != "":
		return "", fmt.Errorf("--prompt and --prompt-file are mutually exclusive")
	case promptText != "":
		return promptText, nil
	case promptFile != "":
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("read prompt file: %w", err)
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return "", fmt.Errorf("prompt file %q is empty", promptFile)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("one of --prompt or --prompt-file is required")
	}
}

func newServerJobSubmitCmd() *cobra.Command {
	var (
		serverURL   string
		project     string
		taskID      string
		agent       string
		prompt      string
		promptFile  string
		projectRoot string
		attachMode  string
		dependsOn   []string
		wait        bool
		follow      bool
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a job to the run-agent server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if taskID == "" {
				taskID = serverGenerateTaskID()
			}
			promptText, err := serverLoadPrompt(prompt, promptFile)
			if err != nil {
				return err
			}
			reqBody := serverJobCreateRequest{
				ProjectID:   project,
				TaskID:      taskID,
				AgentType:   agent,
				Prompt:      promptText,
				ProjectRoot: projectRoot,
				AttachMode:  attachMode,
				DependsOn:   dependsOn,
			}
			return serverJobSubmit(cmd.OutOrStdout(), serverURL, reqBody, wait, follow, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (optional; auto-generated if omitted)")
	cmd.Flags().StringVar(&agent, "agent", "", "agent type, e.g. claude (required)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "task prompt (mutually exclusive with --prompt-file)")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "path to file containing task prompt")
	cmd.Flags().StringVar(&projectRoot, "project-root", "", "working directory for the task")
	cmd.Flags().StringVar(&attachMode, "attach-mode", "create", "attach mode: create, attach, or resume")
	cmd.Flags().StringArrayVar(&dependsOn, "depends-on", nil, "task dependencies (repeat or comma-separate)")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task completion by polling run status")
	cmd.Flags().BoolVar(&follow, "follow", false, "stream task output after submission (implies --wait)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")

	return cmd
}

// serverJobPollInterval is the default poll interval; package-level for testing override.
var serverJobPollInterval = 2 * time.Second

// serverFollowRetryInterval is used by serverWaitForRunStart; overridable in tests.
var serverFollowRetryInterval = time.Second

func serverJobSubmit(out io.Writer, serverURL string, req serverJobCreateRequest, wait bool, follow bool, jsonOutput bool) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := http.Post(serverURL+"/api/v1/tasks", "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result serverJobCreateResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if jsonOutput {
		fmt.Fprintf(out, "%s\n", strings.TrimSpace(string(data)))
		return nil
	}

	fmt.Fprintf(out, "Task created: %s, run_id: %s\n", result.TaskID, result.RunID)

	if follow {
		runID, err := serverWaitForRunStart(serverURL, result.ProjectID, result.TaskID, 30*time.Second)
		if err != nil {
			return err
		}
		return serverTaskLogs(out, serverURL, result.ProjectID, result.TaskID, runID, true, 0)
	}

	if wait {
		return serverWaitForRun(out, serverURL, result.RunID)
	}
	return nil
}

func serverWaitForRunStart(serverURL, project, taskID string, maxWait time.Duration) (string, error) {
	deadline := time.Now().Add(maxWait)
	for {
		runID, err := serverResolveLatestRunID(serverURL, project, taskID)
		if err == nil {
			return runID, nil
		}
		if !strings.Contains(err.Error(), "no runs found") {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for run to start for task %s", taskID)
		}
		time.Sleep(serverFollowRetryInterval)
	}
}

func serverWaitForRun(out io.Writer, serverURL, runID string) error {
	url := serverURL + "/api/v1/runs/" + runID
	fmt.Fprintf(out, "Waiting for run %s to complete...\n", runID)
	for {
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			return fmt.Errorf("poll run: %w", err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read run response: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
		}
		var run serverJobRunResponse
		if err := json.Unmarshal(data, &run); err != nil {
			return fmt.Errorf("decode run: %w", err)
		}
		if !run.EndTime.IsZero() {
			fmt.Fprintf(out, "Run %s completed: status=%s exit_code=%d\n", runID, run.Status, run.ExitCode)
			return nil
		}
		time.Sleep(serverJobPollInterval)
	}
}

func newServerJobListCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks on the run-agent server",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := serverURL + "/api/v1/tasks"
			if project != "" {
				url += "?project_id=" + project
			}

			resp, err := http.Get(url) //nolint:noctx
			if err != nil {
				return fmt.Errorf("list tasks: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverJobTaskListResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			if len(result.Tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tTASK\tSTATUS\tLAST ACTIVITY")
			for _, task := range result.Tasks {
				activity := task.LastActivity.Format(time.RFC3339)
				if task.LastActivity.IsZero() {
					activity = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.ProjectID, task.TaskID, task.Status, activity)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "filter by project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// ─── project ──────────────────────────────────────────────────────────────────

type serverProjectSummary struct {
	ID           string    `json:"id"`
	LastActivity time.Time `json:"last_activity"`
	TaskCount    int       `json:"task_count"`
}

type serverProjectListResponse struct {
	Projects []serverProjectSummary `json:"projects"`
}

type serverProjectStatsResponse struct {
	ProjectID            string `json:"project_id"`
	TotalTasks           int    `json:"total_tasks"`
	TotalRuns            int    `json:"total_runs"`
	RunningRuns          int    `json:"running_runs"`
	CompletedRuns        int    `json:"completed_runs"`
	FailedRuns           int    `json:"failed_runs"`
	CrashedRuns          int    `json:"crashed_runs"`
	MessageBusFiles      int    `json:"message_bus_files"`
	MessageBusTotalBytes int64  `json:"message_bus_total_bytes"`
}

type serverProjectGCResponse struct {
	DeletedRuns int64 `json:"deleted_runs"`
	FreedBytes  int64 `json:"freed_bytes"`
	DryRun      bool  `json:"dry_run"`
}

type serverProjectDeleteResponse struct {
	ProjectID    string `json:"project_id"`
	DeletedTasks int    `json:"deleted_tasks"`
	FreedBytes   int64  `json:"freed_bytes"`
}

func newServerProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects via the server API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newServerProjectListCmd())
	cmd.AddCommand(newServerProjectStatsCmd())
	cmd.AddCommand(newServerProjectGCCmd())
	cmd.AddCommand(newServerProjectDeleteCmd())
	return cmd
}

func newServerProjectListCmd() *cobra.Command {
	var (
		serverURL  string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(serverURL + "/api/projects") //nolint:noctx
			if err != nil {
				return fmt.Errorf("get projects: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverProjectListResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tTASKS\tLAST ACTIVITY")
			for _, p := range result.Projects {
				lastActivity := "-"
				if !p.LastActivity.IsZero() {
					lastActivity = p.LastActivity.Format("2006-01-02 15:04")
				}
				fmt.Fprintf(w, "%s\t%d\t%s\n", p.ID, p.TaskCount, lastActivity)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

func newServerProjectStatsCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := serverURL + "/api/projects/" + project + "/stats"

			resp, err := http.Get(url) //nolint:noctx
			if err != nil {
				return fmt.Errorf("get project stats: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Printf("%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverProjectStatsResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Project:\t%s\n", result.ProjectID)
			fmt.Fprintf(w, "Tasks:\t%d\n", result.TotalTasks)
			fmt.Fprintf(w, "Runs (total):\t%d\n", result.TotalRuns)
			fmt.Fprintf(w, "  Running:\t%d\n", result.RunningRuns)
			fmt.Fprintf(w, "  Completed:\t%d\n", result.CompletedRuns)
			fmt.Fprintf(w, "  Failed:\t%d\n", result.FailedRuns)
			fmt.Fprintf(w, "  Crashed:\t%d\n", result.CrashedRuns)
			fmt.Fprintf(w, "Message bus files:\t%d\n", result.MessageBusFiles)
			fmt.Fprintf(w, "Message bus size:\t%s\n", serverFormatBytes(result.MessageBusTotalBytes))
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func newServerProjectGCCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		olderThan  string
		dryRun     bool
		keepFailed bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Garbage collect old runs for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/projects/%s/gc?older_than=%s&dry_run=%v&keep_failed=%v",
				serverURL, project, olderThan, dryRun, keepFailed)
			resp, err := http.Post(url, "", nil) //nolint:noctx
			if err != nil {
				return fmt.Errorf("gc project: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverProjectGCResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			if result.DryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "DRY RUN: would delete %d runs, free %s\n", result.DeletedRuns, serverFormatBytes(result.FreedBytes))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d runs, freed %s\n", result.DeletedRuns, serverFormatBytes(result.FreedBytes))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&olderThan, "older-than", "168h", "delete runs older than this duration (default: 7 days)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without deleting")
	cmd.Flags().BoolVar(&keepFailed, "keep-failed", false, "keep failed runs (exit code != 0)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func newServerProjectDeleteCmd() *cobra.Command {
	var (
		serverURL  string
		force      bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete an entire project (all tasks and runs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			url := serverURL + "/api/projects/" + projectID
			if force {
				url += "?force=true"
			}

			req, err := http.NewRequest(http.MethodDelete, url, nil) //nolint:noctx
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("delete project: %w", err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}

			if resp.StatusCode == http.StatusConflict {
				var errResp map[string]interface{}
				if jsonErr := json.Unmarshal(data, &errResp); jsonErr == nil {
					if errObj, ok := errResp["error"].(map[string]interface{}); ok {
						if msg, ok := errObj["message"].(string); ok {
							return fmt.Errorf("%s", msg)
						}
					}
				}
				return fmt.Errorf("project has running tasks; stop them first or use --force")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}

			if jsonOutput {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.TrimSpace(string(data)))
				return nil
			}

			var result serverProjectDeleteResponse
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Project %s deleted (%d tasks, %s freed).\n",
				result.ProjectID, result.DeletedTasks, serverFormatBytes(result.FreedBytes))
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().BoolVar(&force, "force", false, "stop running tasks and delete anyway")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

func serverFormatBytes(n int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.2f GB", float64(n)/GB)
	case n >= MB:
		return fmt.Sprintf("%.2f MB", float64(n)/MB)
	case n >= KB:
		return fmt.Sprintf("%.2f KB", float64(n)/KB)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// ─── watch ────────────────────────────────────────────────────────────────────

type serverWatchStatus struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	RunCount int    `json:"run_count"`
	Done     bool   `json:"done"`
}

func isServerTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "done", "error":
		return true
	}
	return false
}

func newServerWatchCmd() *cobra.Command {
	var (
		serverURL  string
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
			effectiveInterval := 5 * time.Second
			if cmd.Flags().Changed("interval") {
				effectiveInterval = interval
			}
			return serverRunWatch(cmd.OutOrStdout(), serverURL, project, taskIDs, timeout, effectiveInterval, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringArrayVar(&taskIDs, "task", nil, "task ID(s) to watch (can repeat; default: all tasks in project)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "max wait time (exit code 1 on timeout)")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "poll interval")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output final status as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func serverRunWatch(out io.Writer, serverURL, project string, taskIDs []string, timeout, interval time.Duration, jsonOutput bool) error {
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
			statuses []serverWatchStatus
			err      error
		)

		if watchingAll {
			statuses, err = serverFetchAllTaskStatuses(serverURL, project)
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
			statuses = make([]serverWatchStatus, 0, len(taskIDs))
			for _, taskID := range taskIDs {
				s, fetchErr := serverFetchSingleTaskStatus(serverURL, project, taskID)
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
			w.Flush() //nolint:errcheck
			fmt.Fprintln(out)
		}

		if allDone {
			if jsonOutput {
				type jsonResult struct {
					Tasks   []serverWatchStatus `json:"tasks"`
					AllDone bool                `json:"all_done"`
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

func serverFetchAllTaskStatuses(serverURL, project string) ([]serverWatchStatus, error) {
	url := serverURL + "/api/projects/" + project + "/tasks"
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

	var result serverTaskListResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	statuses := make([]serverWatchStatus, 0, len(result.Items))
	for _, item := range result.Items {
		statuses = append(statuses, serverWatchStatus{
			TaskID:   item.ID,
			Status:   item.Status,
			RunCount: item.RunCount,
			Done:     isServerTerminalStatus(item.Status),
		})
	}
	return statuses, nil
}

func serverFetchSingleTaskStatus(serverURL, project, taskID string) (serverWatchStatus, error) {
	url := serverURL + "/api/projects/" + project + "/tasks/" + taskID
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return serverWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("fetch task %s: %w", taskID, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return serverWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return serverWatchStatus{TaskID: taskID, Status: "not_found", Done: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return serverWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var detail struct {
		ID     string            `json:"id"`
		Status string            `json:"status"`
		Done   bool              `json:"done"`
		Runs   []json.RawMessage `json:"runs"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return serverWatchStatus{TaskID: taskID, Status: "unknown"}, fmt.Errorf("decode response: %w", err)
	}

	return serverWatchStatus{
		TaskID:   taskID,
		Status:   detail.Status,
		RunCount: len(detail.Runs),
		Done:     detail.Done || isServerTerminalStatus(detail.Status),
	}, nil
}

// ─── bus ──────────────────────────────────────────────────────────────────────

type serverBusMessage struct {
	MsgID     string    `json:"msg_id"`
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Body      string    `json:"body"`
}

type serverBusMessagesResponse struct {
	Messages []serverBusMessage `json:"messages"`
}

type serverBusPostRequest struct {
	Type string `json:"type"`
	Body string `json:"body"`
}

type serverBusPostResponse struct {
	MsgID string `json:"msg_id"`
}

func newServerBusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bus",
		Short: "Read and post to the message bus via the server API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newServerBusReadCmd())
	cmd.AddCommand(newServerBusPostCmd())
	return cmd
}

func newServerBusReadCmd() *cobra.Command {
	var (
		serverURL  string
		project    string
		taskID     string
		tail       int
		follow     bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Read messages from the project or task message bus",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serverBusRead(cmd.OutOrStdout(), serverURL, project, taskID, tail, follow, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (optional; reads task-level bus if set)")
	cmd.Flags().IntVar(&tail, "tail", 0, "show last N messages (0 = all)")
	cmd.Flags().BoolVar(&follow, "follow", false, "stream new messages via SSE")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as raw JSON array")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func serverBusRead(out io.Writer, serverURL, project, taskID string, tail int, follow bool, jsonOutput bool) error {
	var baseURL string
	if taskID != "" {
		baseURL = serverURL + "/api/projects/" + project + "/tasks/" + taskID + "/messages"
	} else {
		baseURL = serverURL + "/api/projects/" + project + "/messages"
	}

	msgs, err := serverFetchBusMessages(baseURL)
	if err != nil {
		return err
	}

	if tail > 0 && len(msgs) > tail {
		msgs = msgs[len(msgs)-tail:]
	}

	if err := serverPrintBusMessages(out, msgs, jsonOutput); err != nil {
		return err
	}

	if !follow {
		return nil
	}

	streamURL := baseURL + "/stream"
	retryWait := 2 * time.Second
	const maxRetryWait = 30 * time.Second

	for {
		err := serverStreamBusSSE(out, streamURL, jsonOutput)
		if err == nil {
			return nil
		}

		fmt.Fprintf(out, "[run-agent server bus] connection lost: %v; reconnecting in %s...\n", err, retryWait)
		time.Sleep(retryWait)
		retryWait *= 2
		if retryWait > maxRetryWait {
			retryWait = maxRetryWait
		}
	}
}

func serverFetchBusMessages(url string) ([]serverBusMessage, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("bus read: fetch messages: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bus read: read response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("bus read: not found: %s", strings.TrimSpace(string(data)))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bus read: server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result serverBusMessagesResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("bus read: decode response: %w", err)
	}

	return result.Messages, nil
}

func serverPrintBusMessages(out io.Writer, msgs []serverBusMessage, jsonOutput bool) error {
	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(msgs)
	}

	if len(msgs) == 0 {
		fmt.Fprintln(out, "(no messages)")
		return nil
	}

	for _, msg := range msgs {
		fmt.Fprintln(out, serverFormatBusMessage(msg))
	}
	return nil
}

func serverFormatBusMessage(msg serverBusMessage) string {
	ts := msg.Timestamp.UTC().Format("2006-01-02 15:04:05")
	msgType := fmt.Sprintf("%-12s", msg.Type)
	body := msg.Body
	if idx := strings.IndexByte(body, '\n'); idx >= 0 {
		body = body[:idx] + "..."
	}
	return fmt.Sprintf("[%s] %s  %s", ts, msgType, body)
}

func serverStreamBusSSE(out io.Writer, url string, jsonOutput bool) error {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("not found: %s", strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	var (
		currentEvent string
		currentData  []string
	)

	flush := func() (isDone bool) {
		event := currentEvent
		data := currentData
		currentEvent = ""
		currentData = currentData[:0]

		switch event {
		case "done":
			return true
		case "heartbeat", "error":
			return false
		}

		for _, d := range data {
			if d == "" || d == "{}" {
				continue
			}
			var msg serverBusMessage
			if err := json.Unmarshal([]byte(d), &msg); err != nil {
				continue
			}
			if jsonOutput {
				enc := json.NewEncoder(out)
				_ = enc.Encode(msg)
			} else {
				fmt.Fprintln(out, serverFormatBusMessage(msg))
			}
		}
		return false
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if flush() {
				return nil
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentData = append(currentData, strings.TrimPrefix(line, "data: "))
		}
	}

	if serr := scanner.Err(); serr != nil {
		return serr
	}
	return fmt.Errorf("stream closed unexpectedly")
}

func newServerBusPostCmd() *cobra.Command {
	var (
		serverURL string
		project   string
		taskID    string
		msgType   string
		body      string
	)

	cmd := &cobra.Command{
		Use:   "post",
		Short: "Post a message to the project or task message bus",
		RunE: func(cmd *cobra.Command, args []string) error {
			if body == "" {
				info, err := os.Stdin.Stat()
				if err == nil && (info.Mode()&os.ModeCharDevice) == 0 {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("read stdin: %w", err)
					}
					body = string(data)
				}
			}
			return serverBusPost(cmd.OutOrStdout(), serverURL, project, taskID, msgType, body)
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", defaultServerURL, "run-agent server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (optional; posts to task-level bus if set)")
	cmd.Flags().StringVar(&msgType, "type", "INFO", "message type")
	cmd.Flags().StringVar(&body, "body", "", "message body (reads from stdin if not provided and stdin is a pipe)")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func serverBusPost(out io.Writer, serverURL, project, taskID, msgType, body string) error {
	var url string
	if taskID != "" {
		url = serverURL + "/api/projects/" + project + "/tasks/" + taskID + "/messages"
	} else {
		url = serverURL + "/api/projects/" + project + "/messages"
	}

	reqBody, err := json.Marshal(serverBusPostRequest{Type: msgType, Body: body})
	if err != nil {
		return fmt.Errorf("bus post: encode request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("bus post: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bus post: read response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bus post: server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result serverBusPostResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("bus post: decode response: %w", err)
	}

	fmt.Fprintf(out, "msg_id: %s\n", result.MsgID)
	return nil
}
