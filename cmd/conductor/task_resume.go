package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func newTaskResumeCmd() *cobra.Command {
	var (
		server     string
		project    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "resume <task-id>",
		Short: "Resume an exhausted task by removing its DONE file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskResume(server, args[0], project, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:14355", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// taskResumeResponse is the JSON response from POST /api/projects/{p}/tasks/{t}/resume.
type taskResumeResponse struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
	Resumed   bool   `json:"resumed"`
}

func taskResume(server, taskID, project string, jsonOutput bool) error {
	url := server + "/api/projects/" + project + "/tasks/" + taskID + "/resume"

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

	var result taskResumeResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	fmt.Printf("Task %s/%s resumed (DONE file removed)\n", result.ProjectID, result.TaskID)
	return nil
}
