package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

func newGCCmd() *cobra.Command {
	var (
		root            string
		olderThan       time.Duration
		dryRun          bool
		project         string
		keepFailed      bool
		rotateBus       bool
		busMaxSize      string
		deleteDoneTasks bool
	)

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Clean up old run directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if root == "" {
				if v := os.Getenv("RUNS_DIR"); v != "" {
					root = v
				} else {
					root = "./runs"
				}
			}
			maxBytes, err := parseSizeBytes(busMaxSize)
			if err != nil {
				return fmt.Errorf("invalid --bus-max-size %q: %w", busMaxSize, err)
			}
			return runGC(root, project, olderThan, dryRun, keepFailed, rotateBus, maxBytes, deleteDoneTasks)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().DurationVar(&olderThan, "older-than", 168*time.Hour, "delete runs older than this duration")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be deleted without deleting")
	cmd.Flags().StringVar(&project, "project", "", "limit gc to a specific project (optional)")
	cmd.Flags().BoolVar(&keepFailed, "keep-failed", false, "keep runs with non-zero exit codes")
	cmd.Flags().BoolVar(&rotateBus, "rotate-bus", false, "rotate message bus files that exceed --bus-max-size")
	cmd.Flags().StringVar(&busMaxSize, "bus-max-size", "10MB", "size threshold for bus file rotation (e.g. 10MB, 5MB, 100KB)")
	cmd.Flags().BoolVar(&deleteDoneTasks, "delete-done-tasks", false,
		"delete task directories that have DONE file, empty runs/, and are older than --older-than")

	return cmd
}

func runGC(root, project string, olderThan time.Duration, dryRun, keepFailed bool, rotateBus bool, busMaxBytes int64, deleteDoneTasks bool) error {
	cutoff := time.Now().Add(-olderThan)

	projects, err := listProjectDirs(root, project)
	if err != nil {
		return err
	}

	var (
		deletedCount     int
		freedBytes       int64
		rotatedCount     int
		deletedTaskCount int
	)

	for _, proj := range projects {
		projDir := filepath.Join(root, proj)
		taskEntries, err := os.ReadDir(projDir)
		if err != nil {
			return fmt.Errorf("read project directory %s: %w", projDir, err)
		}

		// Rotate project-level bus file if requested
		if rotateBus {
			projBus := filepath.Join(projDir, "PROJECT-MESSAGE-BUS.md")
			rotated, err := rotateBusFile(projBus, busMaxBytes, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: rotate %s: %v\n", projBus, err)
			} else if rotated {
				rotatedCount++
			}
		}

		for _, taskEntry := range taskEntries {
			if !taskEntry.IsDir() {
				continue
			}
			taskDir := filepath.Join(projDir, taskEntry.Name())

			// Rotate task-level bus file if requested
			if rotateBus {
				taskBus := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
				rotated, err := rotateBusFile(taskBus, busMaxBytes, dryRun)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: rotate %s: %v\n", taskBus, err)
				} else if rotated {
					rotatedCount++
				}
			}

			runsDir := filepath.Join(taskDir, "runs")
			runEntries, err := os.ReadDir(runsDir)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					if deleteDoneTasks {
						deleted, err := gcTaskDir(taskDir, cutoff, dryRun)
						if err != nil {
							fmt.Fprintf(os.Stderr, "warning: %v\n", err)
						} else if deleted {
							deletedTaskCount++
						}
					}
					continue
				}
				return fmt.Errorf("read runs directory %s: %w", runsDir, err)
			}

			for _, runEntry := range runEntries {
				if !runEntry.IsDir() {
					continue
				}
				runDir := filepath.Join(runsDir, runEntry.Name())
				deleted, freed, err := gcRun(runDir, cutoff, dryRun, keepFailed)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: %v\n", err)
					continue
				}
				if deleted {
					deletedCount++
					freedBytes += freed
				}
			}

			if deleteDoneTasks {
				deleted, err := gcTaskDir(taskDir, cutoff, dryRun)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: %v\n", err)
				} else if deleted {
					deletedTaskCount++
				}
			}
		}
	}

	action := "Deleted"
	if dryRun {
		action = "Would delete"
	}
	fmt.Printf("%s %d runs, freed %.1f MB\n", action, deletedCount, float64(freedBytes)/(1024*1024))

	if deleteDoneTasks && deletedTaskCount > 0 {
		fmt.Printf("%s %d task directories (DONE + empty runs)\n", action, deletedTaskCount)
	}

	if rotateBus && rotatedCount > 0 {
		rotateAction := "Rotated"
		if dryRun {
			rotateAction = "Would rotate"
		}
		fmt.Printf("%s %d message bus file(s)\n", rotateAction, rotatedCount)
	}

	return nil
}

// gcTaskDir deletes a task directory if it has a DONE file, an empty runs/ subdir, and is older than cutoff.
// Returns true if the task directory was deleted (or would be in dry-run mode).
func gcTaskDir(taskDir string, cutoff time.Time, dryRun bool) (bool, error) {
	// Check DONE file exists
	doneFile := filepath.Join(taskDir, "DONE")
	if _, err := os.Stat(doneFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat DONE file in %s: %w", taskDir, err)
	}

	// Check runs/ subdir is empty (or missing)
	runsDir := filepath.Join(taskDir, "runs")
	runEntries, err := os.ReadDir(runsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("read runs dir in %s: %w", taskDir, err)
	}
	// Count subdirectories remaining in runs/
	for _, e := range runEntries {
		if e.IsDir() {
			return false, nil // still has runs, skip
		}
	}

	// Check task dir mtime older than cutoff
	info, err := os.Stat(taskDir)
	if err != nil {
		return false, fmt.Errorf("stat task dir %s: %w", taskDir, err)
	}
	if !info.ModTime().Before(cutoff) {
		return false, nil
	}

	taskName := filepath.Base(taskDir)
	if dryRun {
		fmt.Printf("[dry-run] would delete task dir %s (DONE + empty)\n", taskName)
		return true, nil
	}

	fmt.Printf("deleting task dir %s (DONE + empty)\n", taskName)
	if err := os.RemoveAll(taskDir); err != nil {
		return false, fmt.Errorf("delete task dir %s: %w", taskDir, err)
	}
	return true, nil
}

// rotateBusFile renames busPath to busPath.<YYYYMMDD-HHMMSS>.archived if it exceeds maxBytes.
// Returns true if the file was rotated (or would be in dry-run mode).
func rotateBusFile(busPath string, maxBytes int64, dryRun bool) (bool, error) {
	info, err := os.Stat(busPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.Size() <= maxBytes {
		return false, nil
	}

	timestamp := time.Now().Format("20060102-150405")
	archivedPath := busPath + "." + timestamp + ".archived"

	sizeMB := float64(info.Size()) / (1024 * 1024)
	baseName := filepath.Base(busPath)

	if dryRun {
		fmt.Printf("[dry-run] would rotate %s (%.1f MB → archived)\n", baseName, sizeMB)
		return true, nil
	}

	if err := os.Rename(busPath, archivedPath); err != nil {
		return false, fmt.Errorf("rename %s: %w", busPath, err)
	}
	fmt.Printf("Rotated %s (%.1f MB → archived)\n", baseName, sizeMB)
	return true, nil
}

// parseSizeBytes parses a human-readable size string like "10MB", "5MB", "100KB", "1GB" into bytes.
func parseSizeBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	upper := strings.ToUpper(s)
	var multiplier int64
	var numStr string

	switch {
	case strings.HasSuffix(upper, "GB"):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "MB"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "KB"):
		multiplier = 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "B"):
		multiplier = 1
		numStr = s[:len(s)-1]
	default:
		// treat as plain bytes
		multiplier = 1
		numStr = s
	}

	var n int64
	if _, err := fmt.Sscanf(strings.TrimSpace(numStr), "%d", &n); err != nil {
		return 0, fmt.Errorf("cannot parse number from %q", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("size must be non-negative")
	}
	return n * multiplier, nil
}

func listProjectDirs(root, project string) ([]string, error) {
	if project != "" {
		return []string{project}, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read root directory %s: %w", root, err)
	}
	var projects []string
	for _, e := range entries {
		if e.IsDir() {
			projects = append(projects, e.Name())
		}
	}
	return projects, nil
}

func gcRun(runDir string, cutoff time.Time, dryRun, keepFailed bool) (deleted bool, freed int64, err error) {
	infoPath := filepath.Join(runDir, "run-info.yaml")
	info, readErr := storage.ReadRunInfo(infoPath)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return false, 0, nil // skip: may be active or mid-creation
		}
		return false, 0, fmt.Errorf("skip %s: read run-info: %w", runDir, readErr)
	}

	// never delete active runs
	if info.Status == storage.StatusRunning {
		return false, 0, nil
	}

	// only delete completed or failed runs
	if info.Status != storage.StatusCompleted && info.Status != storage.StatusFailed {
		return false, 0, nil
	}

	// honour --keep-failed
	if keepFailed && info.Status == storage.StatusFailed {
		return false, 0, nil
	}

	// determine run age from start_time; fall back to end_time
	runTime := info.StartTime
	if runTime.IsZero() {
		runTime = info.EndTime
	}
	if runTime.IsZero() || !runTime.Before(cutoff) {
		return false, 0, nil
	}

	size := dirSize(runDir)
	startStr := info.StartTime.Format("2006-01-02 15:04:05")
	if dryRun {
		fmt.Printf("[dry-run] would delete %s (status=%s, started=%s, size=%.1f MB)\n",
			runDir, info.Status, startStr, float64(size)/(1024*1024))
	} else {
		fmt.Printf("deleting %s (status=%s, started=%s, size=%.1f MB)\n",
			runDir, info.Status, startStr, float64(size)/(1024*1024))
		if err := os.RemoveAll(runDir); err != nil {
			return false, 0, fmt.Errorf("delete %s: %w", runDir, err)
		}
	}
	return true, size, nil
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
