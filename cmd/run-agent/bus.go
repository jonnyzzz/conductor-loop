package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/spf13/cobra"
)

func newBusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bus",
		Short: "Read and post messages to the message bus",
	}
	cmd.AddCommand(newBusPostCmd())
	cmd.AddCommand(newBusReadCmd())
	cmd.AddCommand(newBusDiscoverCmd())
	return cmd
}

var busDiscoveryFileNames = []string{
	"TASK-MESSAGE-BUS.md",
	"PROJECT-MESSAGE-BUS.md",
	"MESSAGE-BUS.md",
}

func normalizeInferredIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "." || trimmed == string(filepath.Separator) {
		return ""
	}
	return trimmed
}

// resolveBusFilePath computes the message bus file path from the project/task hierarchy.
// If taskID is non-empty, returns <root>/<project>/<task>/TASK-MESSAGE-BUS.md.
// Otherwise, returns <root>/<project>/PROJECT-MESSAGE-BUS.md.
// root defaults to storage.runs_dir from config, then ~/.run-agent/runs when empty.
func resolveBusFilePath(root, projectID, taskID string) (string, error) {
	var err error
	root, err = config.ResolveRunsDir(root)
	if err != nil {
		return "", fmt.Errorf("resolve runs dir: %w", err)
	}
	if taskID != "" {
		return filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md"), nil
	}
	return filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md"), nil
}

// discoverBusFilePath searches upward from startDir (or CWD when empty) and
// returns the nearest known bus file.
func discoverBusFilePath(startDir string) (string, error) {
	dir := strings.TrimSpace(startDir)
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current working directory: %w", err)
		}
		dir = cwd
	}
	dir = filepath.Clean(dir)
	start := dir

	for {
		for _, name := range busDiscoveryFileNames {
			candidate := filepath.Join(dir, name)
			info, err := os.Lstat(candidate)
			if err == nil {
				if info.Mode().IsRegular() {
					return candidate, nil
				}
				continue
			}
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("stat %q: %w", candidate, err)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no message bus file found from %q (searched for %s)", start, strings.Join(busDiscoveryFileNames, ", "))
}

func inferMessageScopeFromBusPath(path string) (projectID, taskID string) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return "", ""
	}

	base := filepath.Base(clean)
	switch base {
	case "TASK-MESSAGE-BUS.md":
		taskDir := filepath.Dir(clean)
		projectDir := filepath.Dir(taskDir)
		taskID = filepath.Base(taskDir)
		projectID = filepath.Base(projectDir)
	case "PROJECT-MESSAGE-BUS.md", "MESSAGE-BUS.md":
		projectID = filepath.Base(filepath.Dir(clean))
	default:
		return "", ""
	}

	projectID = normalizeInferredIdentifier(projectID)
	taskID = normalizeInferredIdentifier(taskID)
	return projectID, taskID
}

func inferMessageScopeFromTaskFolder(path string) (projectID, taskID string) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return "", ""
	}

	taskID = normalizeInferredIdentifier(filepath.Base(clean))
	projectID = normalizeInferredIdentifier(filepath.Base(filepath.Dir(clean)))
	return projectID, taskID
}

func inferMessageScopeFromRunFolder(path string) (projectID, taskID, runID string) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return "", "", ""
	}

	runID = normalizeInferredIdentifier(filepath.Base(clean))
	runsDir := filepath.Dir(clean)
	if normalizeInferredIdentifier(filepath.Base(runsDir)) != "runs" {
		return "", "", ""
	}

	taskDir := filepath.Dir(runsDir)
	taskID = normalizeInferredIdentifier(filepath.Base(taskDir))
	projectID = normalizeInferredIdentifier(filepath.Base(filepath.Dir(taskDir)))
	if projectID == "" || taskID == "" || runID == "" {
		return "", "", ""
	}
	return projectID, taskID, runID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func resolveBusPostPath(busPath, root, projectID, taskID string) (string, error) {
	path := strings.TrimSpace(busPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("MESSAGE_BUS"))
	}
	if path == "" && strings.TrimSpace(projectID) != "" {
		resolved, err := resolveBusFilePath(root, projectID, taskID)
		if err != nil {
			return "", err
		}
		path = resolved
	}
	if path != "" {
		return path, nil
	}

	discovered, err := discoverBusFilePath("")
	if err != nil {
		return "", fmt.Errorf("--bus or --project is required (or set MESSAGE_BUS env var, or run from a directory with MESSAGE-BUS.md/PROJECT-MESSAGE-BUS.md/TASK-MESSAGE-BUS.md): %w", err)
	}
	return discovered, nil
}

func resolveBusPostMessageContext(projectID, taskID, runID, busPath string) (resolvedProjectID, resolvedTaskID, resolvedRunID string) {
	resolvedProjectID = strings.TrimSpace(projectID)
	resolvedTaskID = strings.TrimSpace(taskID)
	resolvedRunID = strings.TrimSpace(runID)

	busProjectID, busTaskID := inferMessageScopeFromBusPath(busPath)
	runProjectID, runTaskID, runRunID := inferMessageScopeFromRunFolder(os.Getenv("RUN_FOLDER"))
	taskProjectID, taskTaskID := inferMessageScopeFromTaskFolder(os.Getenv("TASK_FOLDER"))

	if resolvedProjectID == "" {
		resolvedProjectID = firstNonEmpty(
			busProjectID,
			runProjectID,
			taskProjectID,
			os.Getenv("JRUN_PROJECT_ID"),
		)
	}
	if resolvedTaskID == "" {
		resolvedTaskID = firstNonEmpty(
			busTaskID,
			runTaskID,
			taskTaskID,
			os.Getenv("JRUN_TASK_ID"),
		)
	}
	if resolvedRunID == "" {
		resolvedRunID = firstNonEmpty(
			runRunID,
			os.Getenv("JRUN_ID"),
		)
	}

	return resolvedProjectID, resolvedTaskID, resolvedRunID
}

func newBusPostCmd() *cobra.Command {
	var (
		busPath   string
		root      string
		msgType   string
		projectID string
		taskID    string
		runID     string
		body      string
	)

	cmd := &cobra.Command{
		Use:   "post",
		Short: "Post a message to the message bus",
		Long: `Post a message to the message bus.

The bus file path is resolved in this order:
  1. --bus flag
  2. MESSAGE_BUS environment variable
  3. --project (+ optional --task) auto-resolve from project/task hierarchy
  4. Auto-discover nearest bus file from current directory
  5. Error

When --project is specified without --bus, the path is auto-resolved:
  - With --task:    <root>/<project>/<task>/TASK-MESSAGE-BUS.md
  - Without --task: <root>/<project>/PROJECT-MESSAGE-BUS.md

Message project/task/run values are resolved in this order:
  1. Explicit flags (--project/--task/--run)
  2. Context inference (resolved bus path, RUN_FOLDER, TASK_FOLDER)
  3. JRUN_PROJECT_ID/JRUN_TASK_ID/JRUN_ID environment variables
  4. Error (project_id required)

Use "run-agent bus discover" to preview auto-discovery from your current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedBusPath, err := resolveBusPostPath(busPath, root, projectID, taskID)
			if err != nil {
				return err
			}
			busPath = resolvedBusPath

			projectID, taskID, runID = resolveBusPostMessageContext(projectID, taskID, runID, busPath)
			if projectID == "" {
				return fmt.Errorf("project id is empty and could not be inferred; provide --project, set JRUN_PROJECT_ID, or use a scoped bus path (TASK-MESSAGE-BUS.md/PROJECT-MESSAGE-BUS.md)")
			}

			if body == "" {
				info, err := os.Stdin.Stat()
				if err == nil && (info.Mode()&os.ModeCharDevice) == 0 {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("read stdin: %w", err)
					}
					body = string(data)
				}
			}
			bus, err := messagebus.NewMessageBus(busPath)
			if err != nil {
				return err
			}
			msg := &messagebus.Message{
				Type:      msgType,
				ProjectID: projectID,
				TaskID:    taskID,
				RunID:     runID,
				Body:      body,
			}
			msgID, err := bus.AppendMessage(msg)
			if err != nil {
				return err
			}
			fmt.Printf("msg_id: %s\n", msgID)
			return nil
		},
	}

	cmd.Flags().StringVar(&busPath, "bus", "", "path to message bus file (uses MESSAGE_BUS env var if not set)")
	cmd.Flags().StringVar(&root, "root", "", "root directory for project/task bus resolution (default: ~/.run-agent/runs)")
	cmd.Flags().StringVar(&msgType, "type", "INFO", "message type")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID (optional; inferred from context if omitted)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (optional; inferred from context if omitted)")
	cmd.Flags().StringVar(&runID, "run", "", "run ID (optional; inferred from context if omitted)")
	cmd.Flags().StringVar(&body, "body", "", "message body (reads from stdin if not provided and stdin is a pipe)")

	return cmd
}

func newBusReadCmd() *cobra.Command {
	var (
		busPath   string
		root      string
		projectID string
		taskID    string
		tail      int
		follow    bool
	)

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Read messages from the message bus",
		Long: `Read messages from the message bus.

The bus file path is resolved in this order:
  1. --project (+ optional --task) auto-resolve from project/task hierarchy
  2. --bus flag
  3. MESSAGE_BUS environment variable
  4. Auto-discover nearest bus file from current directory
  5. Error

When --project is specified, the path is auto-resolved:
  - With --task:    <root>/<project>/<task>/TASK-MESSAGE-BUS.md
  - Without --task: <root>/<project>/PROJECT-MESSAGE-BUS.md

Specifying both --bus and --project is an error.

Use "run-agent bus discover" to preview auto-discovery from your current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve bus path: project/task hierarchy > --bus > MESSAGE_BUS env
			if projectID != "" {
				if busPath != "" {
					return fmt.Errorf("cannot specify both --bus and --project")
				}
				resolved, resolveErr := resolveBusFilePath(root, projectID, taskID)
				if resolveErr != nil {
					return resolveErr
				}
				busPath = resolved
			}
			if busPath == "" {
				busPath = os.Getenv("MESSAGE_BUS")
			}
			if busPath == "" {
				discovered, err := discoverBusFilePath("")
				if err != nil {
					return fmt.Errorf("--bus or --project is required (or set MESSAGE_BUS env var, or run from a directory with MESSAGE-BUS.md/PROJECT-MESSAGE-BUS.md/TASK-MESSAGE-BUS.md): %w", err)
				}
				busPath = discovered
			}
			bus, err := messagebus.NewMessageBus(busPath, messagebus.WithPollInterval(500*time.Millisecond))
			if err != nil {
				return err
			}
			var messages []*messagebus.Message
			if tail > 0 {
				messages, err = bus.ReadLastN(tail)
			} else {
				messages, err = bus.ReadMessages("")
			}
			if err != nil {
				return err
			}
			for _, msg := range messages {
				printBusMessage(msg)
			}
			if !follow {
				return nil
			}
			var lastID string
			if len(messages) > 0 {
				lastID = messages[len(messages)-1].MsgID
			}
			for {
				time.Sleep(500 * time.Millisecond)
				newMsgs, err := bus.ReadMessages(lastID)
				if err != nil {
					if errors.Is(err, messagebus.ErrSinceIDNotFound) {
						lastID = ""
						continue
					}
					return err
				}
				for _, msg := range newMsgs {
					printBusMessage(msg)
					lastID = msg.MsgID
				}
			}
		},
	}

	cmd.Flags().StringVar(&busPath, "bus", "", "path to message bus file (uses MESSAGE_BUS env var if not set)")
	cmd.Flags().StringVar(&root, "root", "", "root directory for project/task bus resolution (default: ~/.run-agent/runs)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID (with --root to resolve bus path; without --task reads project-level bus)")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID (requires --project; resolves task-level bus)")
	cmd.Flags().IntVar(&tail, "tail", 20, "print last N messages")
	cmd.Flags().BoolVar(&follow, "follow", false, "watch for new messages (Ctrl-C to exit)")

	return cmd
}

func newBusDiscoverCmd() *cobra.Command {
	var fromDir string

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Detect the nearest message bus file from current directory",
		Long: `Detect the nearest message bus file by searching upward from the current directory.

Search order within each directory:
  1. TASK-MESSAGE-BUS.md
  2. PROJECT-MESSAGE-BUS.md
  3. MESSAGE-BUS.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := discoverBusFilePath(fromDir)
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromDir, "from", "", "start searching from this directory (defaults to current working directory)")
	return cmd
}

func printBusMessage(msg *messagebus.Message) {
	ts := msg.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] (%s) %s\n", ts, msg.Type, msg.Body)
}
