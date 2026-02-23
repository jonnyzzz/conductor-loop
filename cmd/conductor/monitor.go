package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newMonitorCmd() *cobra.Command {
	var (
		server         string
		project        string
		interval       time.Duration
		staleThreshold time.Duration
		rateLimit      time.Duration
		todosFile      string
		agent          string
	)

	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor tasks defined in TODOs.md",
		Long: `Monitors tasks defined in a markdown file (default: TODOs.md).
- Starts missing tasks found in TODOs.md (unchecked items).
- Resumes failed/unfinished tasks (with rate limiting).
- Detects stale running tasks (no output activity) and restarts them.
- Marks completed tasks as done in TODOs.md.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(cmd.OutOrStdout(), server, project, todosFile, agent, interval, staleThreshold, rateLimit)
		},
	}

	cmd.Flags().StringVar(&server, "server", "http://localhost:14355", "conductor server URL")
	cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
	cmd.Flags().StringVar(&todosFile, "todos", "TODOs.md", "path to TODOs file")
	cmd.Flags().StringVar(&agent, "agent", "claude", "agent to use for new tasks")
	cmd.Flags().DurationVar(&interval, "interval", 10*time.Second, "monitoring loop interval")
	cmd.Flags().DurationVar(&staleThreshold, "stale-threshold", 5*time.Minute, "threshold for stale task detection")
	cmd.Flags().DurationVar(&rateLimit, "rate-limit", 1*time.Minute, "minimum interval between resumes for a task")
	cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck

	return cmd
}

type todoTask struct {
	ID          string
	Description string
	LineNum     int // 0-based line number in file
	Checked     bool
}

func parseTodos(filename string) ([]todoTask, []string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var tasks []todoTask

	// Regex for "- [ ] task-ID: description" or "- [x] task-ID..."
	// Matches: "- [ ] task-123", "- [x] task-123: desc", "- [ ] task-123 desc"
	// Group 1: checked status (space, x, X)
	// Group 2: task ID
	// Group 3: description (optional, includes leading separator)
	re := regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(task-[a-zA-Z0-9-]+)(.*)$`)

	for i, line := range lines {
		matches := re.FindStringSubmatch(line)
		if matches != nil {
			checked := strings.TrimSpace(matches[1]) != ""
			taskID := matches[2]
			desc := matches[3]
			desc = strings.TrimPrefix(desc, ":")
			desc = strings.TrimSpace(desc)

			tasks = append(tasks, todoTask{
				ID:          taskID,
				Description: desc,
				LineNum:     i,
				Checked:     checked,
			})
		}
	}

	return tasks, lines, nil
}

func runMonitor(out io.Writer, server, project, todosFile, agent string, interval, staleThreshold, rateLimit time.Duration) error {
	fmt.Fprintf(out, "Starting monitor loop for project %s using %s\n", project, todosFile)
	fmt.Fprintf(out, "Agent: %s, Interval: %v, Stale Threshold: %v, Rate Limit: %v\n", agent, interval, staleThreshold, rateLimit)

	lastResumeTime := make(map[string]time.Time)

	for {
		// 1. Parse TODOs
		todos, fileLines, err := parseTodos(todosFile)
		if err != nil {
			fmt.Fprintf(out, "Error parsing %s: %v\n", todosFile, err)
			time.Sleep(interval)
			continue
		}

		// 2. Fetch server tasks
		serverTasks, err := fetchProjectTasksMap(server, project)
		if err != nil {
			fmt.Fprintf(out, "Error fetching tasks: %v\n", err)
			time.Sleep(interval)
			continue
		}

		updatedFile := false
		lines := append([]string(nil), fileLines...) // Copy lines

		// 3. Logic
		for _, todo := range todos {
			st, exists := serverTasks[todo.ID]

			// A. Start missing tasks
			if !todo.Checked && !exists {
				fmt.Fprintf(out, "Starting missing task: %s\n", todo.ID)
				// Create task
				req := jobCreateRequest{
					ProjectID: project,
					TaskID:    todo.ID,
					AgentType: agent,
					Prompt:    todo.Description,
				}
				if req.Prompt == "" {
					req.Prompt = "Task " + todo.ID // Fallback
				}

				if err := jobSubmit(out, server, req, false, false, false); err != nil {
					fmt.Fprintf(out, "Failed to start task %s: %v\n", todo.ID, err)
				}
				continue
			}

			if !exists {
				continue
			}

			// B. Auto-finalize completed tasks
			// Case 1: Task is already DONE in system.
			if !todo.Checked && (st.Done || st.Status == "completed") {
				fmt.Fprintf(out, "Task %s is completed. Marking as DONE in %s.\n", todo.ID, todosFile)
				lines[todo.LineNum] = strings.Replace(lines[todo.LineNum], "[ ]", "[x]", 1)
				updatedFile = true
				continue
			}

			// Case 2: Task finished successfully with output, but not yet marked DONE in system.
			if !todo.Checked && st.Status != "running" && st.RunCount > 0 {
				if st.LastRunStatus == "completed" && st.LastRunExitCode == 0 && st.LastRunOutputSize > 0 {
					fmt.Fprintf(out, "Task %s finished successfully (size=%d). Marking as DONE in system...\n", todo.ID, st.LastRunOutputSize)
					// taskStop writes DONE file and stops runs (which are already stopped here).
					if err := taskStop(server, todo.ID, project, false); err != nil {
						fmt.Fprintf(out, "Failed to mark task %s as DONE: %v\n", todo.ID, err)
					} else {
						// Also mark in TODOs.md immediately
						fmt.Fprintf(out, "Task %s is completed. Marking as DONE in %s.\n", todo.ID, todosFile)
						lines[todo.LineNum] = strings.Replace(lines[todo.LineNum], "[ ]", "[x]", 1)
						updatedFile = true
					}
					continue
				}
			}

			// C. Resume failed/unfinished tasks
			// "unfinished" usually means it's not done/completed.
			// If it's failed or error or stopped, we resume.
			shouldResume := st.Status == "failed" || st.Status == "error" || st.Status == "stopped"
			if !todo.Checked && shouldResume {
				if time.Since(lastResumeTime[todo.ID]) > rateLimit {
					fmt.Fprintf(out, "Resuming failed/stopped task: %s (status: %s)\n", todo.ID, st.Status)
					if err := taskResume(server, todo.ID, project, false); err != nil {
						fmt.Fprintf(out, "Failed to resume task %s: %v\n", todo.ID, err)
					}
					lastResumeTime[todo.ID] = time.Now()
				}
				continue
			}

			// D. Detect stale running tasks
			if !todo.Checked && st.Status == "running" {
				if time.Since(st.LastActivity) > staleThreshold {
					if time.Since(lastResumeTime[todo.ID]) > rateLimit {
						fmt.Fprintf(out, "Task %s is stale (last activity: %v). Restarting...\n", todo.ID, st.LastActivity)
						// Stop then resume
						if err := taskStop(server, todo.ID, project, false); err != nil {
							fmt.Fprintf(out, "Failed to stop stale task %s: %v\n", todo.ID, err)
						} else {
							// Wait a bit?
							time.Sleep(1 * time.Second)
							if err := taskResume(server, todo.ID, project, false); err != nil {
								fmt.Fprintf(out, "Failed to resume stale task %s: %v\n", todo.ID, err)
							}
							lastResumeTime[todo.ID] = time.Now()
						}
					}
				}
			}
		}

		if updatedFile {
			if err := os.WriteFile(todosFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				fmt.Fprintf(out, "Error updating %s: %v\n", todosFile, err)
			} else {
				fmt.Fprintf(out, "Updated %s\n", todosFile)
			}
		}

		time.Sleep(interval)
	}
}

func fetchProjectTasksMap(server, project string) (map[string]taskListItem, error) {
	url := server + "/api/projects/" + project + "/tasks"
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result taskListAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	m := make(map[string]taskListItem)
	for _, item := range result.Items {
		m[item.ID] = item
	}
	return m, nil
}
