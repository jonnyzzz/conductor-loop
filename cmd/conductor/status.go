package main

import (
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

func newStatusCmd() *cobra.Command {
	var (
		server     string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show conductor server status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serverStatus(server, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output response as JSON")

	return cmd
}

// conductorStatusResponse is the JSON response from GET /api/v1/status.
type conductorStatusResponse struct {
	ActiveRunsCount  int      `json:"active_runs_count"`
	UptimeSeconds    float64  `json:"uptime_seconds"`
	ConfiguredAgents []string `json:"configured_agents"`
	Version          string   `json:"version"`
}

func serverStatus(server string, jsonOutput bool) error {
	resp, err := http.Get(server + "/api/v1/status") //nolint:noctx
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

	var result conductorStatusResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Version:\t%s\n", result.Version)
	fmt.Fprintf(w, "Uptime:\t%s\n", formatUptime(result.UptimeSeconds))
	fmt.Fprintf(w, "Active Runs:\t%d\n", result.ActiveRunsCount)
	agents := strings.Join(result.ConfiguredAgents, ", ")
	if agents == "" {
		agents = "(none)"
	}
	fmt.Fprintf(w, "Configured Agents:\t%s\n", agents)
	return w.Flush()
}

func formatUptime(seconds float64) string {
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
