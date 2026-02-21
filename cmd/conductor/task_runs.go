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

func newTaskRunsCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "runs <task-id>",
		Short: "List all runs for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskRuns(cmd.OutOrStdout(), server, project, args[0], jsonOutput, limit)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:14355", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of runs to show")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// runsListAPIResponse is the paginated JSON response from GET /api/projects/{id}/tasks/{taskID}/runs.
type runsListAPIResponse struct {
	Items   []runListItem `json:"items"`
	Total   int           `json:"total"`
	HasMore bool          `json:"has_more"`
}

// runListItem is a run entry in the task runs list response.
type runListItem struct {
	ID           string     `json:"id"`
	Agent        string     `json:"agent"`
	AgentVersion string     `json:"agent_version"`
	Status       string     `json:"status"`
	ExitCode     int        `json:"exit_code"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	ErrorSummary string     `json:"error_summary"`
}

func taskRuns(out io.Writer, server, project, taskID string, jsonOutput bool, limit int) error {
	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/runs?limit=%d", server, project, taskID, limit)

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
		fmt.Fprintf(out, "%s\n", strings.TrimSpace(string(data)))
		return nil
	}

	var result runsListAPIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if len(result.Items) == 0 {
		fmt.Fprintf(out, "no runs found for task %s\n", taskID)
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
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
			formatRunDuration(run.StartTime, run.EndTime),
			started, errSummary)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if result.HasMore {
		fmt.Fprintf(out, "(showing %d of %d runs; use --limit to see more)\n", len(result.Items), result.Total)
	}
	return nil
}

// formatRunDuration computes a human-readable duration between start and end.
// If end is nil, returns "running".
func formatRunDuration(start time.Time, end *time.Time) string {
	if end == nil {
		return "running"
	}
	d := end.Sub(start)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
