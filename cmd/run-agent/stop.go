package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

const (
	stopPollInterval = 500 * time.Millisecond
	stopTimeout      = 30 * time.Second
)

func newStopCmd() *cobra.Command {
	var (
		runDir    string
		root      string
		projectID string
		taskID    string
		runID     string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(runDir, root, projectID, taskID, runID, force)
		},
	}

	cmd.Flags().StringVar(&runDir, "run-dir", "", "path to run directory (alternative to --root/--project/--task)")
	cmd.Flags().StringVar(&root, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&runID, "run", "", "run id (optional, defaults to latest running run)")
	cmd.Flags().BoolVar(&force, "force", false, "send SIGKILL if process does not stop within timeout")

	return cmd
}

func runStop(runDir, root, projectID, taskID, runID string, force bool) error {
	resolvedRunDir, err := resolveRunDir(runDir, root, projectID, taskID, runID)
	if err != nil {
		return err
	}

	infoPath := filepath.Join(resolvedRunDir, "run-info.yaml")
	info, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		return fmt.Errorf("read run-info from %s: %w", resolvedRunDir, err)
	}

	if info.Status != storage.StatusRunning {
		fmt.Fprintf(os.Stderr, "run %s is not running (status: %s)\n", info.RunID, info.Status)
		return nil
	}
	if !storage.CanTerminateProcess(info) {
		return fmt.Errorf("run %s is externally owned and cannot be stopped by conductor", info.RunID)
	}

	pgid := info.PGID
	if pgid <= 0 {
		pgid = info.PID
	}
	if pgid <= 0 {
		return fmt.Errorf("run %s has no valid PID/PGID in run-info", info.RunID)
	}

	if !runner.IsProcessAlive(info.PID) {
		fmt.Fprintf(os.Stderr, "run %s (PID %d) is no longer alive\n", info.RunID, info.PID)
		return nil
	}

	if err := runner.TerminateProcessGroup(pgid); err != nil {
		return fmt.Errorf("terminate process group (pgid=%d): %w", pgid, err)
	}

	deadline := time.Now().Add(stopTimeout)
	stopped := false
	for time.Now().Before(deadline) {
		time.Sleep(stopPollInterval)
		if !runner.IsProcessAlive(info.PID) {
			stopped = true
			break
		}
	}

	if !stopped {
		if force {
			fmt.Fprintf(os.Stderr, "run %s (PID %d) did not stop within timeout, sending SIGKILL\n", info.RunID, info.PID)
			if err := runner.KillProcessGroup(pgid); err != nil {
				return fmt.Errorf("kill process group (pgid=%d): %w", pgid, err)
			}
		} else {
			return fmt.Errorf("run %s (PID %d) did not stop within %s; use --force to send SIGKILL", info.RunID, info.PID, stopTimeout)
		}
	}

	fmt.Fprintf(os.Stdout, "Stopped run %s (PID %d)\n", info.RunID, info.PID)
	return nil
}

func resolveRunDir(runDir, root, projectID, taskID, runID string) (string, error) {
	if runDir != "" {
		if _, err := os.Stat(runDir); err != nil {
			return "", fmt.Errorf("run directory %s: %w", runDir, err)
		}
		return runDir, nil
	}

	if root == "" {
		return "", fmt.Errorf("--root is required when --run-dir is not specified")
	}
	if projectID == "" {
		return "", fmt.Errorf("--project is required when --run-dir is not specified")
	}
	if taskID == "" {
		return "", fmt.Errorf("--task is required when --run-dir is not specified")
	}

	runsDir := filepath.Join(root, projectID, taskID, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no runs found for project %s task %s", projectID, taskID)
		}
		return "", fmt.Errorf("read runs directory %s: %w", runsDir, err)
	}

	type runEntry struct {
		id  string
		dir string
	}
	var running []runEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if runID != "" && e.Name() != runID {
			continue
		}
		dir := filepath.Join(runsDir, e.Name())
		info, err := storage.ReadRunInfo(filepath.Join(dir, "run-info.yaml"))
		if err != nil {
			continue
		}
		if info.Status == storage.StatusRunning {
			running = append(running, runEntry{id: e.Name(), dir: dir})
		}
	}

	if len(running) == 0 {
		if runID != "" {
			return "", fmt.Errorf("no running run %s found for project %s task %s", runID, projectID, taskID)
		}
		return "", fmt.Errorf("no running runs found for project %s task %s", projectID, taskID)
	}

	sort.Slice(running, func(i, j int) bool {
		return running[i].id < running[j].id
	})
	return running[len(running)-1].dir, nil
}
