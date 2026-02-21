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
