package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newTaskLogsCmd() *cobra.Command {
	var (
		server  string
		project string
		runID   string
		follow  bool
		tail    int
	)

	cmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "Stream task output via the conductor server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return taskLogs(cmd.OutOrStdout(), server, project, args[0], runID, follow, tail)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&runID, "run", "", "specific run ID (default: latest)")
	cmd.Flags().BoolVar(&follow, "follow", false, "keep streaming; reconnect if connection drops")
	cmd.Flags().IntVar(&tail, "tail", 0, "output last N lines only (0 = all)")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

// resolveLatestRunID fetches the task detail and returns the best run ID.
// It prefers the last running run; falls back to the last run overall.
func resolveLatestRunID(server, project, taskID string) (string, error) {
	url := server + "/api/projects/" + project + "/tasks/" + taskID

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

	// Prefer the last running run; fall back to the last run overall.
	chosen := detail.Runs[len(detail.Runs)-1].RunID
	for i := len(detail.Runs) - 1; i >= 0; i-- {
		if detail.Runs[i].Status == "running" {
			chosen = detail.Runs[i].RunID
			break
		}
	}
	return chosen, nil
}

func taskLogs(out io.Writer, server, project, taskID, runID string, follow bool, tail int) error {
	if runID == "" {
		var err error
		runID, err = resolveLatestRunID(server, project, taskID)
		if err != nil {
			return err
		}
	}

	streamURL := server + "/api/projects/" + project + "/tasks/" + taskID + "/runs/" + runID + "/stream?name=stdout"

	retryWait := 2 * time.Second
	const maxRetryWait = 30 * time.Second

	var allLines []string // used only when tail > 0

	for {
		done, lines, err := streamSSELines(out, streamURL, tail > 0)
		if tail > 0 {
			allLines = append(allLines, lines...)
		}

		if done {
			if tail > 0 {
				printTailLines(out, allLines, tail)
			}
			return nil
		}

		if err != nil && !follow {
			return fmt.Errorf("task logs: %w", err)
		}

		if !follow {
			return nil
		}

		// Connection dropped; wait and retry.
		time.Sleep(retryWait)
		retryWait *= 2
		if retryWait > maxRetryWait {
			retryWait = maxRetryWait
		}
	}
}

// streamSSELines reads an SSE stream from url.
// If buffer is true, it collects data lines and returns them without writing to out.
// If buffer is false, it writes lines to out as they arrive.
// Returns (done bool, lines []string, err error).
// done=true means "event: done" was received from the server.
func streamSSELines(out io.Writer, url string, buffer bool) (done bool, lines []string, err error) {
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

		// Default or "log" event: output the data lines.
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
		// Ignore id: and other SSE fields.
	}

	if serr := scanner.Err(); serr != nil {
		return false, lines, serr
	}
	return false, lines, nil
}

// printTailLines prints the last n lines from lines to out.
func printTailLines(out io.Writer, lines []string, n int) {
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
