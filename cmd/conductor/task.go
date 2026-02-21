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

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskStatusCmd())
	cmd.AddCommand(newTaskStopCmd())
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskDeleteCmd())
	cmd.AddCommand(newTaskResumeCmd())
	cmd.AddCommand(newTaskLogsCmd())
	return cmd
}

func newTaskStatusCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "status <task-id>",
		Short: "Get the status of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskStatus(server, args[0], project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// taskDetailResponse is the JSON response from GET /api/v1/tasks/<task_id>.
type taskDetailResponse struct {
	ProjectID    string           `json:"project_id"`
	TaskID       string           `json:"task_id"`
	Status       string           `json:"status"`
	LastActivity time.Time        `json:"last_activity"`
	Runs         []taskRunSummary `json:"runs"`
}

// taskRunSummary is a run entry in the task detail response.
type taskRunSummary struct {
	RunID     string    `json:"run_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	ExitCode  int       `json:"exit_code"`
}

func newTaskStopCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "stop <task-id>",
		Short: "Stop all running runs of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskStop(server, args[0], project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// taskStopResponse is the JSON response from DELETE /api/v1/tasks/<task_id>.
type taskStopResponse struct {
	StoppedRuns int `json:"stopped_runs"`
}

func taskStop(server, taskID, project string, jsonOutput bool) error {
	url := server + "/api/v1/tasks/" + taskID
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

	var result taskStopResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	fmt.Printf("Task %s: stopped %d run(s)\n", taskID, result.StoppedRuns)
	return nil
}

func newTaskListCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskList(server, project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// taskListItem is a task entry in the project tasks list response.
type taskListItem struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Status       string    `json:"status"`
	LastActivity time.Time `json:"last_activity"`
	RunCount     int       `json:"run_count"`
}

// taskListAPIResponse is the paginated JSON response from GET /api/projects/{id}/tasks.
type taskListAPIResponse struct {
	Items   []taskListItem `json:"items"`
	Total   int            `json:"total"`
	HasMore bool           `json:"has_more"`
}

func taskList(server, project string, jsonOutput bool) error {
	url := server + "/api/projects/" + project + "/tasks"

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

	var result taskListAPIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK ID\tSTATUS\tRUNS\tLAST ACTIVITY")
	for _, t := range result.Items {
		lastActivity := "-"
		if !t.LastActivity.IsZero() {
			lastActivity = t.LastActivity.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", t.ID, t.Status, t.RunCount, lastActivity)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if result.HasMore {
		fmt.Printf("(showing %d of %d tasks; use --limit to see more)\n", len(result.Items), result.Total)
	}
	return nil
}

func newTaskDeleteCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Delete a task and all its runs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskDelete(server, args[0], project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func taskDelete(server, taskID, project string, jsonOutput bool) error {
	url := server + "/api/projects/" + project + "/tasks/" + taskID

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
			fmt.Printf(`{"task_id":%q,"deleted":true}` + "\n", taskID)
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
}

func taskStatus(server, taskID, project string, jsonOutput bool) error {
	url := server + "/api/v1/tasks/" + taskID
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

	var result taskDetailResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	fmt.Printf("Task:   %s/%s\n", result.ProjectID, result.TaskID)
	fmt.Printf("Status: %s\n", result.Status)
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
		w.Flush()
	}
	return nil
}
