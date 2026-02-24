package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newOutputSynthesizeCmd() *cobra.Command {
	var (
		root      string
		projectID string
		taskID    string
		runs      string
		outFile   string
		strict    bool
	)

	cmd := &cobra.Command{
		Use:   "synthesize",
		Short: "Aggregate output from multiple runs into a single synthesized artifact",
		Long: `Synthesize concatenates output.md (or agent-stdout.txt as fallback) from
multiple runs into a single markdown document. Useful for aggregating results
from parallel sub-agents.

Run selectors via --runs accept comma-separated run IDs or absolute run paths.
When --root/--project/--task are given, run IDs are resolved relative to that task.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runs = strings.TrimSpace(runs)
			if runs == "" {
				return fmt.Errorf("--runs is required (comma-separated run IDs or absolute paths)")
			}

			runSelectors := splitAndTrim(runs, ",")
			if len(runSelectors) == 0 {
				return fmt.Errorf("--runs is empty")
			}

			var out io.Writer = cmd.OutOrStdout()
			if outFile != "" {
				f, err := os.Create(outFile)
				if err != nil {
					return fmt.Errorf("create output file: %w", err)
				}
				defer f.Close()
				out = f
			}

			return runOutputSynthesize(out, root, projectID, taskID, runSelectors, strict)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&taskID, "task", "", "task id")
	cmd.Flags().StringVar(&runs, "runs", "", "comma-separated run IDs or absolute run directory paths")
	cmd.Flags().StringVar(&outFile, "out", "", "write synthesized output to this file instead of stdout")
	cmd.Flags().BoolVar(&strict, "strict", false, "fail if any run has no output artifact")

	return cmd
}

func runOutputSynthesize(out io.Writer, root, projectID, taskID string, runSelectors []string, strict bool) error {
	if len(runSelectors) == 0 {
		return fmt.Errorf("no run selectors provided")
	}

	var missingCount int
	for _, sel := range runSelectors {
		runDir := resolveRunDirForSynthesize(root, projectID, taskID, sel)

		content, sourcePath, err := readRunOutputForSynthesize(runDir)
		if err != nil {
			if strict {
				return fmt.Errorf("run %s: %w", sel, err)
			}
			fmt.Fprintf(out, "## run: %s\n\n", sel)
			fmt.Fprintf(out, "> WARNING: no output found for this run (path: %s)\n\n", runDir)
			missingCount++
			continue
		}

		fmt.Fprintf(out, "## run: %s\n\n", sel)
		fmt.Fprintf(out, "source: %s\n\n", sourcePath)
		fmt.Fprintf(out, "%s\n\n", strings.TrimSpace(content))
		fmt.Fprintf(out, "---\n\n")
	}

	if missingCount > 0 && !strict {
		fmt.Fprintf(out, "> %d run(s) had no output artifact\n", missingCount)
	}
	return nil
}

// resolveRunDirForSynthesize resolves a run selector to an absolute run directory path.
// If the selector is already an absolute path, it is used directly.
// Otherwise it is interpreted as a run ID under root/project/task/runs/.
func resolveRunDirForSynthesize(root, projectID, taskID, sel string) string {
	if filepath.IsAbs(sel) {
		return sel
	}
	if root != "" && projectID != "" && taskID != "" {
		return filepath.Join(root, projectID, taskID, "runs", sel)
	}
	return sel
}

// readRunOutputForSynthesize reads the best available output for a run directory.
// Prefers output.md; falls back to agent-stdout.txt.
func readRunOutputForSynthesize(runDir string) (content, sourcePath string, err error) {
	outputPath := filepath.Join(runDir, "output.md")
	if data, e := os.ReadFile(outputPath); e == nil {
		return string(data), outputPath, nil
	}

	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if data, e := os.ReadFile(stdoutPath); e == nil {
		return string(data), stdoutPath, nil
	}

	return "", runDir, fmt.Errorf("no output.md or agent-stdout.txt found in %s", runDir)
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
