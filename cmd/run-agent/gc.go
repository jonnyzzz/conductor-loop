package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

func newGCCmd() *cobra.Command {
	var (
		root       string
		olderThan  time.Duration
		dryRun     bool
		project    string
		keepFailed bool
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
			return runGC(root, project, olderThan, dryRun, keepFailed)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().DurationVar(&olderThan, "older-than", 168*time.Hour, "delete runs older than this duration")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be deleted without deleting")
	cmd.Flags().StringVar(&project, "project", "", "limit gc to a specific project (optional)")
	cmd.Flags().BoolVar(&keepFailed, "keep-failed", false, "keep runs with non-zero exit codes")

	return cmd
}

func runGC(root, project string, olderThan time.Duration, dryRun, keepFailed bool) error {
	cutoff := time.Now().Add(-olderThan)

	projects, err := listProjectDirs(root, project)
	if err != nil {
		return err
	}

	var (
		deletedCount int
		freedBytes   int64
	)

	for _, proj := range projects {
		projDir := filepath.Join(root, proj)
		taskEntries, err := os.ReadDir(projDir)
		if err != nil {
			return fmt.Errorf("read project directory %s: %w", projDir, err)
		}

		for _, taskEntry := range taskEntries {
			if !taskEntry.IsDir() {
				continue
			}
			runsDir := filepath.Join(projDir, taskEntry.Name(), "runs")
			runEntries, err := os.ReadDir(runsDir)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
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
		}
	}

	action := "Deleted"
	if dryRun {
		action = "Would delete"
	}
	fmt.Printf("%s %d runs, freed %.1f MB\n", action, deletedCount, float64(freedBytes)/(1024*1024))
	return nil
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
