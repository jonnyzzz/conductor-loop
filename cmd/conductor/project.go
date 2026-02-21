package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectStatsCmd())
	cmd.AddCommand(newProjectGCCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	return cmd
}

func newProjectListCmd() *cobra.Command {
	var (
		server     string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return projectList(server, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// projectSummaryResponse is a project entry in the list response.
type projectSummaryResponse struct {
	ID           string    `json:"id"`
	LastActivity time.Time `json:"last_activity"`
	TaskCount    int       `json:"task_count"`
}

// projectListAPIResponse is the JSON response from GET /api/projects.
type projectListAPIResponse struct {
	Projects []projectSummaryResponse `json:"projects"`
}

func projectList(server string, jsonOutput bool) error {
	resp, err := http.Get(server + "/api/projects") //nolint:noctx
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

	var result projectListAPIResponse
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
}

func newProjectStatsCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return projectStats(server, project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// projectStatsResponse is the JSON response from GET /api/projects/{id}/stats.
type projectStatsResponse struct {
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

func projectStats(server, project string, jsonOutput bool) error {
	url := server + "/api/projects/" + project + "/stats"

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

	var result projectStatsResponse
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
	fmt.Fprintf(w, "Message bus size:\t%s\n", formatBytes(result.MessageBusTotalBytes))
	return w.Flush()
}

func newProjectGCCmd() *cobra.Command {
	var (
		server     string
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
			return projectGC(cmd.OutOrStdout(), server, project, olderThan, dryRun, keepFailed, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&olderThan, "older-than", "168h", "delete runs older than this duration (default: 7 days)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without deleting")
	cmd.Flags().BoolVar(&keepFailed, "keep-failed", false, "keep failed runs (exit code != 0)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck
	return cmd
}

// projectGCResponse is the JSON response from POST /api/projects/{id}/gc.
type projectGCResponse struct {
	DeletedRuns int64 `json:"deleted_runs"`
	FreedBytes  int64 `json:"freed_bytes"`
	DryRun      bool  `json:"dry_run"`
}

func projectGC(out io.Writer, server, project, olderThan string, dryRun, keepFailed, jsonOutput bool) error {
	url := fmt.Sprintf("%s/api/projects/%s/gc?older_than=%s&dry_run=%v&keep_failed=%v",
		server, project, olderThan, dryRun, keepFailed)
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
		fmt.Fprintf(out, "%s\n", strings.TrimSpace(string(data)))
		return nil
	}

	var result projectGCResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if result.DryRun {
		fmt.Fprintf(out, "DRY RUN: would delete %d runs, free %s\n", result.DeletedRuns, formatBytes(result.FreedBytes))
	} else {
		fmt.Fprintf(out, "Deleted %d runs, freed %s\n", result.DeletedRuns, formatBytes(result.FreedBytes))
	}
	return nil
}

func newProjectDeleteCmd() *cobra.Command {
	var (
		server     string
		force      bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete an entire project (all tasks and runs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return projectDelete(cmd.OutOrStdout(), server, args[0], force, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().BoolVar(&force, "force", false, "stop running tasks and delete anyway")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// projectDeleteResponse is the JSON response from DELETE /api/projects/{id}.
type projectDeleteResponse struct {
	ProjectID    string `json:"project_id"`
	DeletedTasks int    `json:"deleted_tasks"`
	FreedBytes   int64  `json:"freed_bytes"`
}

func projectDelete(out io.Writer, server, projectID string, force, jsonOutput bool) error {
	url := server + "/api/projects/" + projectID
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
		fmt.Fprintf(out, "%s\n", strings.TrimSpace(string(data)))
		return nil
	}

	var result projectDeleteResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	fmt.Fprintf(out, "Project %s deleted (%d tasks, %s freed).\n",
		result.ProjectID, result.DeletedTasks, formatBytes(result.FreedBytes))
	return nil
}

// formatBytes converts byte counts to a human-readable string.
func formatBytes(n int64) string {
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
