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

type busMessage struct {
	MsgID     string    `json:"msg_id"`
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Body      string    `json:"body"`
}

type busMessagesResponse struct {
	Messages []busMessage `json:"messages"`
}

func newBusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bus",
		Short: "Read and post to the message bus",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newBusReadCmd())
	return cmd
}

func newBusReadCmd() *cobra.Command {
	var (
		server     string
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
			return conductorBusRead(cmd.OutOrStdout(), server, project, taskID, tail, follow, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (optional; reads task-level bus if set)")
	cmd.Flags().IntVar(&tail, "tail", 0, "show last N messages (0 = all)")
	cmd.Flags().BoolVar(&follow, "follow", false, "stream new messages via SSE (keep watching)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as raw JSON array")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

func conductorBusRead(out io.Writer, server, project, taskID string, tail int, follow bool, jsonOutput bool) error {
	var baseURL string
	if taskID != "" {
		baseURL = server + "/api/projects/" + project + "/tasks/" + taskID + "/messages"
	} else {
		baseURL = server + "/api/projects/" + project + "/messages"
	}

	// Fetch existing messages first.
	msgs, err := fetchBusMessages(baseURL)
	if err != nil {
		return err
	}

	// Apply tail filtering.
	if tail > 0 && len(msgs) > tail {
		msgs = msgs[len(msgs)-tail:]
	}

	if err := printBusMessages(out, msgs, jsonOutput); err != nil {
		return err
	}

	if !follow {
		return nil
	}

	// Follow mode: connect to SSE stream.
	streamURL := baseURL + "/stream"
	retryWait := 2 * time.Second
	const maxRetryWait = 30 * time.Second

	for {
		err := streamBusSSE(out, streamURL, jsonOutput)
		if err == nil {
			return nil // clean done
		}

		fmt.Fprintf(out, "[conductor bus] connection lost: %v; reconnecting in %s...\n", err, retryWait)
		time.Sleep(retryWait)
		retryWait *= 2
		if retryWait > maxRetryWait {
			retryWait = maxRetryWait
		}
	}
}

func fetchBusMessages(url string) ([]busMessage, error) {
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

	var result busMessagesResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("bus read: decode response: %w", err)
	}

	return result.Messages, nil
}

func printBusMessages(out io.Writer, msgs []busMessage, jsonOutput bool) error {
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
		fmt.Fprintln(out, formatBusMessage(msg))
	}
	return nil
}

func formatBusMessage(msg busMessage) string {
	ts := msg.Timestamp.UTC().Format("2006-01-02 15:04:05")
	msgType := fmt.Sprintf("%-12s", msg.Type)
	body := msg.Body
	if idx := strings.IndexByte(body, '\n'); idx >= 0 {
		body = body[:idx] + "..."
	}
	return fmt.Sprintf("[%s] %s  %s", ts, msgType, body)
}

func streamBusSSE(out io.Writer, url string, jsonOutput bool) error {
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

		// Default or "message" event: decode and print each data line.
		for _, d := range data {
			if d == "" || d == "{}" {
				continue
			}
			var msg busMessage
			if err := json.Unmarshal([]byte(d), &msg); err != nil {
				// Not a valid message; skip.
				continue
			}
			if jsonOutput {
				enc := json.NewEncoder(out)
				_ = enc.Encode(msg)
			} else {
				fmt.Fprintln(out, formatBusMessage(msg))
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
