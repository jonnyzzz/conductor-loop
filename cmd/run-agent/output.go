package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

// followPollInterval, followFileWaitTimeout, and followNoDataTimeout control the
// polling behaviour of --follow. They are package-level vars so tests can shorten them.
var (
	followPollInterval    = 500 * time.Millisecond
	followFileWaitTimeout = 5 * time.Second
	followNoDataTimeout   = 60 * time.Second
)

func newOutputCmd() *cobra.Command {
	var (
		root      string
		projectID string
		taskID    string
		runID     string
		runDir    string
		tail      int
		file      string
		follow    bool
	)

	cmd := &cobra.Command{
		Use:   "output",
		Short: "Print output from a completed run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if follow {
				return runFollowOutput(runDir, root, projectID, taskID, runID, file)
			}
			return runOutput(runDir, root, projectID, taskID, runID, file, tail)
		},
	}

	cmd.Flags().StringVar(&runDir, "run-dir", "", "direct path to run directory (overrides --project/--task/--run)")
	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ~/.run-agent/runs)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&runID, "run", "", "run id (uses most recent if omitted)")
	cmd.Flags().IntVar(&tail, "tail", 0, "print last N lines only (0 = all)")
	cmd.Flags().StringVar(&file, "file", "output", "file to print: output (default), stdout, stderr, prompt")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow output as it is written (for running jobs)")

	cmd.AddCommand(newOutputSynthesizeCmd())
	return cmd
}

func runOutput(runDir, root, projectID, taskID, runID, file string, tail int) error {
	resolved, err := resolveOutputRunDir(runDir, root, projectID, taskID, runID)
	if err != nil {
		return err
	}

	filePath, err := resolveOutputFile(resolved, file)
	if err != nil {
		return err
	}

	return printFile(filePath, tail)
}

func runFollowOutput(runDir, root, projectID, taskID, runID, file string) error {
	resolved, err := resolveOutputRunDir(runDir, root, projectID, taskID, runID)
	if err != nil {
		return err
	}
	return followOutput(resolved, file)
}

// resolveOutputRunDir resolves the run directory path.
// If runDir is given, it is used directly.
// Otherwise, root/project/task are used to find the most recent run (or a specific runID).
func resolveOutputRunDir(runDir, root, projectID, taskID, runID string) (string, error) {
	if runDir != "" {
		if _, err := os.Stat(runDir); err != nil {
			return "", fmt.Errorf("run directory %s: %w", runDir, err)
		}
		return runDir, nil
	}

	var rootErr error
	root, rootErr = config.ResolveRunsDir(root)
	if rootErr != nil {
		return "", fmt.Errorf("resolve runs dir: %w", rootErr)
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

	var runDirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if runID != "" && e.Name() != runID {
			continue
		}
		runDirs = append(runDirs, e.Name())
	}

	if len(runDirs) == 0 {
		if runID != "" {
			return "", fmt.Errorf("run %s not found for project %s task %s", runID, projectID, taskID)
		}
		return "", fmt.Errorf("no runs found for project %s task %s", projectID, taskID)
	}

	sort.Strings(runDirs)
	chosen := runDirs[len(runDirs)-1]
	return filepath.Join(runsDir, chosen), nil
}

// resolveOutputFile determines the actual file path to print based on the --file flag.
func resolveOutputFile(runDir, file string) (string, error) {
	switch strings.ToLower(file) {
	case "output", "":
		// Try output.md first, fall back to agent-stdout.txt
		outputMD := filepath.Join(runDir, "output.md")
		if _, err := os.Stat(outputMD); err == nil {
			return outputMD, nil
		}
		fallback := filepath.Join(runDir, "agent-stdout.txt")
		if _, err := os.Stat(fallback); err == nil {
			return fallback, nil
		}
		return "", fmt.Errorf("file not found: %s (also tried agent-stdout.txt)", outputMD)
	case "stdout":
		p := filepath.Join(runDir, "agent-stdout.txt")
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("file not found: %s", p)
		}
		return p, nil
	case "stderr":
		p := filepath.Join(runDir, "agent-stderr.txt")
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("file not found: %s", p)
		}
		return p, nil
	case "prompt":
		p := filepath.Join(runDir, "prompt.md")
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("file not found: %s", p)
		}
		return p, nil
	default:
		return "", fmt.Errorf("unknown --file value %q: must be output, stdout, stderr, or prompt", file)
	}
}

// printFile prints the contents of filePath to stdout.
// If tail > 0, only the last tail lines are printed.
func printFile(filePath string, tail int) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	if tail <= 0 {
		// Print everything
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		return scanner.Err()
	}

	// Collect last N lines using a ring buffer approach
	lines := make([]string, tail)
	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines[count%tail] = scanner.Text()
		count++
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	start := 0
	n := count
	if count > tail {
		start = count % tail
		n = tail
	}
	for i := 0; i < n; i++ {
		fmt.Println(lines[(start+i)%tail])
	}
	return nil
}

// followOutput tails the output file of a run in real-time.
// If the run is already complete it prints all content and exits immediately.
func followOutput(runDir, file string) error {
	runInfoPath := filepath.Join(runDir, "run-info.yaml")

	// If already complete, just print everything and exit.
	if info, err := storage.ReadRunInfo(runInfoPath); err == nil && info.Status != storage.StatusRunning {
		filePath, err := resolveOutputFile(runDir, file)
		if err != nil {
			return err
		}
		return printFile(filePath, 0)
	}

	// For a running job, follow agent-stdout.txt (or the appropriate live file).
	outputPath := resolveFollowFilePath(runDir, file)

	// Wait for the file to appear (it may not exist yet at run start).
	deadline := time.Now().Add(followFileWaitTimeout)
	for {
		if _, err := os.Stat(outputPath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("output file not found: %s", outputPath)
		}
		time.Sleep(followPollInterval)
	}

	// Handle Ctrl+C gracefully.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	offset := int64(0)
	lastData := time.Now()

	for {
		select {
		case <-sigCh:
			fmt.Println()
			return nil
		default:
		}

		// Read and print any new content.
		if n := drainNewContent(outputPath, offset); n > 0 {
			offset += n
			lastData = time.Now()
		}

		// Check if the run has completed.
		if info, err := storage.ReadRunInfo(runInfoPath); err == nil && info.Status != storage.StatusRunning {
			drainNewContent(outputPath, offset)
			return nil
		}

		// Stop if no new data has arrived for too long.
		if time.Since(lastData) > followNoDataTimeout {
			return nil
		}

		time.Sleep(followPollInterval)
	}
}

// drainNewContent reads bytes from path starting at offset, writes them to stdout,
// and returns the byte count written.
func drainNewContent(path string, offset int64) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0
	}
	n, _ := io.Copy(os.Stdout, f)
	return n
}

// resolveFollowFilePath returns the live file to tail for a running job.
// For "output" or "stdout" it uses agent-stdout.txt (written in real-time).
func resolveFollowFilePath(runDir, file string) string {
	switch strings.ToLower(file) {
	case "stderr":
		return filepath.Join(runDir, "agent-stderr.txt")
	case "prompt":
		return filepath.Join(runDir, "prompt.md")
	default: // output, stdout, ""
		return filepath.Join(runDir, "agent-stdout.txt")
	}
}
