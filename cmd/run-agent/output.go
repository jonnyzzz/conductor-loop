package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
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
	)

	cmd := &cobra.Command{
		Use:   "output",
		Short: "Print output from a completed run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOutput(runDir, root, projectID, taskID, runID, file, tail)
		},
	}

	cmd.Flags().StringVar(&runDir, "run-dir", "", "direct path to run directory (overrides --project/--task/--run)")
	cmd.Flags().StringVar(&root, "root", "", "root directory (default: ./runs or RUNS_DIR env)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&runID, "run", "", "run id (uses most recent if omitted)")
	cmd.Flags().IntVar(&tail, "tail", 0, "print last N lines only (0 = all)")
	cmd.Flags().StringVar(&file, "file", "output", "file to print: output (default), stdout, stderr, prompt")

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

	if root == "" {
		if v := os.Getenv("RUNS_DIR"); v != "" {
			root = v
		} else {
			root = "./runs"
		}
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
