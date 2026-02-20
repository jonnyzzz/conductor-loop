package main

import (
	"bytes"
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

func newJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Manage jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newJobSubmitCmd())
	cmd.AddCommand(newJobListCmd())
	return cmd
}

func newJobSubmitCmd() *cobra.Command {
	var (
		server      string
		project     string
		taskID      string
		agent       string
		prompt      string
		projectRoot string
		attachMode  string
		wait        bool
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a job to the conductor server",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := jobCreateRequest{
				ProjectID:   project,
				TaskID:      taskID,
				AgentType:   agent,
				Prompt:      prompt,
				ProjectRoot: projectRoot,
				AttachMode:  attachMode,
			}
			return jobSubmit(server, req, wait, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (required)")
	cmd.Flags().StringVar(&agent, "agent", "", "agent type, e.g. claude (required)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "task prompt (required)")
	cmd.Flags().StringVar(&projectRoot, "project-root", "", "working directory for the task")
	cmd.Flags().StringVar(&attachMode, "attach-mode", "create", "attach mode: create, attach, or resume")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task completion by polling run status")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("agent")
	_ = cmd.MarkFlagRequired("prompt")

	return cmd
}

func newJobListCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks on the conductor server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return jobList(server, project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "filter by project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// jobCreateRequest is the JSON body for POST /api/v1/tasks.
type jobCreateRequest struct {
	ProjectID   string `json:"project_id"`
	TaskID      string `json:"task_id"`
	AgentType   string `json:"agent_type"`
	Prompt      string `json:"prompt"`
	ProjectRoot string `json:"project_root,omitempty"`
	AttachMode  string `json:"attach_mode,omitempty"`
}

// jobCreateResponse is the JSON response from POST /api/v1/tasks.
type jobCreateResponse struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
}

// jobRunResponse is the JSON response from GET /api/v1/runs/<run_id>.
type jobRunResponse struct {
	RunID     string    `json:"run_id"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	ExitCode  int       `json:"exit_code"`
}

// jobTaskResponse is the JSON for a single task in GET /api/v1/tasks.
type jobTaskResponse struct {
	ProjectID    string    `json:"project_id"`
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"`
	LastActivity time.Time `json:"last_activity"`
}

// jobTaskListResponse is the JSON response from GET /api/v1/tasks.
type jobTaskListResponse struct {
	Tasks []jobTaskResponse `json:"tasks"`
}

func jobSubmit(server string, req jobCreateRequest, wait bool, jsonOutput bool) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := http.Post(server+"/api/v1/tasks", "application/json", bytes.NewReader(body))
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

	var result jobCreateResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if jsonOutput {
		fmt.Printf("%s\n", strings.TrimSpace(string(data)))
		return nil
	}

	fmt.Printf("Task created: %s, run_id: %s\n", result.TaskID, result.RunID)

	if wait {
		return waitForRun(server, result.RunID)
	}
	return nil
}

// pollInterval is used by waitForRun; overridable in tests.
var pollInterval = 2 * time.Second

func waitForRun(server, runID string) error {
	url := server + "/api/v1/runs/" + runID
	fmt.Printf("Waiting for run %s to complete...\n", runID)
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
		var run jobRunResponse
		if err := json.Unmarshal(data, &run); err != nil {
			return fmt.Errorf("decode run: %w", err)
		}
		if !run.EndTime.IsZero() {
			fmt.Printf("Run %s completed: status=%s exit_code=%d\n", runID, run.Status, run.ExitCode)
			return nil
		}
		time.Sleep(pollInterval)
	}
}

func jobList(server, project string, jsonOutput bool) error {
	url := server + "/api/v1/tasks"
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

	var result jobTaskListResponse
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
}
