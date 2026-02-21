package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
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

// generateTaskID returns a task ID in the format task-YYYYMMDD-HHMMSS-xxxxxx
// where xxxxxx is a 6-character random hex string.
func generateTaskID() string {
	now := time.Now().UTC()
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback: use time-based bytes (nanosecond lower bits)
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

// loadPrompt returns the prompt text from inline --prompt or --prompt-file.
// Exactly one of promptText or promptFile must be non-empty; returns an error otherwise.
func loadPrompt(promptText, promptFile string) (string, error) {
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
		promptFile  string
		projectRoot string
		attachMode  string
		wait        bool
		follow      bool
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a job to the conductor server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if taskID == "" {
				taskID = generateTaskID()
			}
			promptText, err := loadPrompt(prompt, promptFile)
			if err != nil {
				return err
			}
			req := jobCreateRequest{
				ProjectID:   project,
				TaskID:      taskID,
				AgentType:   agent,
				Prompt:      promptText,
				ProjectRoot: projectRoot,
				AttachMode:  attachMode,
			}
			return jobSubmit(cmd.OutOrStdout(), server, req, wait, follow, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:14355", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (optional; auto-generated if omitted)")
	cmd.Flags().StringVar(&agent, "agent", "", "agent type, e.g. claude (required)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "task prompt (mutually exclusive with --prompt-file)")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "path to file containing task prompt (mutually exclusive with --prompt)")
	cmd.Flags().StringVar(&projectRoot, "project-root", "", "working directory for the task")
	cmd.Flags().StringVar(&attachMode, "attach-mode", "create", "attach mode: create, attach, or resume")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task completion by polling run status")
	cmd.Flags().BoolVar(&follow, "follow", false, "stream task output after submission (implies --wait)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")

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

	cmd.Flags().StringVar(&server, "server", "http://localhost:14355", "conductor server URL")
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

func jobSubmit(out io.Writer, server string, req jobCreateRequest, wait bool, follow bool, jsonOutput bool) error {
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
		fmt.Fprintf(out, "%s\n", strings.TrimSpace(string(data)))
		return nil
	}

	fmt.Fprintf(out, "Task created: %s, run_id: %s\n", result.TaskID, result.RunID)

	if follow {
		// Wait for a run to start (it may not exist yet immediately after submit),
		// then stream its output with follow=true.
		runID, err := waitForRunStart(server, result.ProjectID, result.TaskID, 30*time.Second)
		if err != nil {
			return err
		}
		return taskLogs(out, server, result.ProjectID, result.TaskID, runID, true, 0)
	}

	if wait {
		return waitForRun(out, server, result.RunID)
	}
	return nil
}

// pollInterval is used by waitForRun; overridable in tests.
var pollInterval = 2 * time.Second

// followRetryInterval is used by waitForRunStart; overridable in tests.
var followRetryInterval = time.Second

// waitForRunStart polls the task until a run is available, then returns the run ID.
// It retries every followRetryInterval for up to maxWait.
func waitForRunStart(server, project, taskID string, maxWait time.Duration) (string, error) {
	deadline := time.Now().Add(maxWait)
	for {
		runID, err := resolveLatestRunID(server, project, taskID)
		if err == nil {
			return runID, nil
		}
		if !strings.Contains(err.Error(), "no runs found") {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for run to start for task %s", taskID)
		}
		time.Sleep(followRetryInterval)
	}
}

func waitForRun(out io.Writer, server, runID string) error {
	url := server + "/api/v1/runs/" + runID
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
		var run jobRunResponse
		if err := json.Unmarshal(data, &run); err != nil {
			return fmt.Errorf("decode run: %w", err)
		}
		if !run.EndTime.IsZero() {
			fmt.Fprintf(out, "Run %s completed: status=%s exit_code=%d\n", runID, run.Status, run.ExitCode)
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
