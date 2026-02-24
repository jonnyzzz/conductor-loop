package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/spf13/cobra"
)

// iteratePassTokens are keywords in review output indicating approval.
var iteratePassTokens = []string{"APPROVED", "PASS"}

// iterateFailTokens are keywords indicating rejection.
var iterateFailTokens = []string{"REJECTED", "FAIL", "CHANGES_REQUESTED"}

func newIterateCmd() *cobra.Command {
	var (
		rootDir         string
		configPath      string
		projectID       string
		taskID          string
		agent           string
		prompt          string
		promptFile      string
		reviewPrompt    string
		reviewPromptFile string
		maxIterations   int
	)

	cmd := &cobra.Command{
		Use:   "iterate",
		Short: "Run an iterative implementation-review-fix loop",
		Long: `Iterate runs a cycle of implementation runs followed by review runs.
Each iteration executes an implementation step, then a review step.
If the review output contains pass tokens (APPROVED, PASS) the loop exits with
code 0. If fail tokens appear (REJECTED, FAIL, CHANGES_REQUESTED) the next
iteration begins (up to --max-iterations). Exit code 1 means iterations exhausted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir = strings.TrimSpace(rootDir)
			projectID = strings.TrimSpace(projectID)
			taskID = strings.TrimSpace(taskID)
			prompt = strings.TrimSpace(prompt)
			promptFile = strings.TrimSpace(promptFile)
			reviewPrompt = strings.TrimSpace(reviewPrompt)
			reviewPromptFile = strings.TrimSpace(reviewPromptFile)

			if rootDir == "" {
				return fmt.Errorf("--root is required")
			}
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			if taskID == "" {
				return fmt.Errorf("--task is required")
			}
			if prompt == "" && promptFile == "" {
				return fmt.Errorf("--prompt or --prompt-file is required")
			}
			if reviewPrompt == "" && reviewPromptFile == "" {
				return fmt.Errorf("--review-prompt or --review-prompt-file is required")
			}
			if maxIterations <= 0 {
				maxIterations = 3
			}
			if configPath == "" && agent == "" {
				found, err := config.FindDefaultConfig()
				if err != nil {
					return err
				}
				configPath = found
			}

			return runIterate(cmd, rootDir, configPath, projectID, taskID, agent,
				prompt, promptFile, reviewPrompt, reviewPromptFile, maxIterations)
		},
	}

	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (required)")
	cmd.Flags().StringVar(&agent, "agent", "", "agent type")
	cmd.Flags().StringVar(&prompt, "prompt", "", "implementation prompt")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "implementation prompt file path")
	cmd.Flags().StringVar(&reviewPrompt, "review-prompt", "", "review prompt")
	cmd.Flags().StringVar(&reviewPromptFile, "review-prompt-file", "", "review prompt file path")
	cmd.Flags().IntVar(&maxIterations, "max-iterations", 3, "maximum number of implementation+review cycles (default: 3)")

	return cmd
}

type iterationResult struct {
	iteration int
	implRunID string
	revRunID  string
	verdict   string // "pass", "fail", "unknown"
}

func runIterate(cmd *cobra.Command, rootDir, configPath, projectID, taskID, agent,
	prompt, promptFile, reviewPrompt, reviewPromptFile string, maxIterations int) error {

	out := cmd.OutOrStdout()
	var prevFeedback string
	var results []iterationResult

	for i := 1; i <= maxIterations; i++ {
		fmt.Fprintf(out, "[iterate] iteration %d/%d\n", i, maxIterations)

		// Build implementation prompt with previous feedback appended.
		implPrompt := buildIteratePrompt(prompt, promptFile, prevFeedback)

		// Run implementation step.
		implRunID, implErr := runIterateStep(rootDir, configPath, projectID, taskID, agent, implPrompt, fmt.Sprintf("impl-iter%d", i))
		fmt.Fprintf(out, "[iterate] impl run: %s\n", implRunID)
		if implErr != nil {
			fmt.Fprintf(out, "[iterate] impl run failed: %v\n", implErr)
		}

		// Read impl output for review context.
		implOutput := readIterateRunOutput(rootDir, projectID, taskID, implRunID)

		// Build review prompt.
		revPromptText := buildIteratePrompt(reviewPrompt, reviewPromptFile, "")
		if implOutput != "" {
			revPromptText += "\n\n## Implementation output to review\n\n" + implOutput
		}

		// Run review step.
		revRunID, _ := runIterateStep(rootDir, configPath, projectID, taskID, agent, revPromptText, fmt.Sprintf("review-iter%d", i))
		fmt.Fprintf(out, "[iterate] review run: %s\n", revRunID)

		// Read review output and determine verdict.
		revOutput := readIterateRunOutput(rootDir, projectID, taskID, revRunID)
		verdict := detectIterateVerdict(revOutput)

		res := iterationResult{
			iteration: i,
			implRunID: implRunID,
			revRunID:  revRunID,
			verdict:   verdict,
		}
		results = append(results, res)

		fmt.Fprintf(out, "[iterate] verdict: %s\n", verdict)

		if verdict == "pass" {
			writeIterateSummary(rootDir, projectID, taskID, results, "passed")
			fmt.Fprintf(out, "[iterate] PASSED on iteration %d\n", i)
			return nil
		}

		// Carry forward review output as feedback for next iteration.
		prevFeedback = revOutput
	}

	writeIterateSummary(rootDir, projectID, taskID, results, "exhausted")
	fmt.Fprintf(out, "[iterate] EXHAUSTED max iterations (%d) without passing review\n", maxIterations)
	os.Exit(1)
	return nil
}

// runIterateStep launches a single job run and returns the run ID.
func runIterateStep(rootDir, configPath, projectID, taskID, agent, promptText, label string) (string, error) {
	// Write prompt to a temp file.
	tmpFile, err := os.CreateTemp("", "iterate-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp prompt: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(promptText); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write temp prompt: %w", err)
	}
	tmpFile.Close()

	opts := runner.JobOptions{
		RootDir:    rootDir,
		ConfigPath: configPath,
		Agent:      agent,
		PromptPath: tmpFile.Name(),
	}

	// Generate a unique sub-task ID for this step.
	subTaskID := fmt.Sprintf("%s-%s-%s", taskID, label, time.Now().Format("150405"))

	err = runner.RunJob(projectID, subTaskID, opts)
	return subTaskID, err
}

func buildIteratePrompt(inline, filePath, prevFeedback string) string {
	var text string
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			text = strings.TrimSpace(string(data))
		}
	}
	if text == "" {
		text = inline
	}
	if prevFeedback != "" {
		text += "\n\n## Previous review feedback\n\n" + prevFeedback
	}
	return text
}

func readIterateRunOutput(rootDir, projectID, taskID, runID string) string {
	// Look for output.md in the task's runs directory.
	// The sub-task directory is rootDir/projectID/runID (since runID is used as task ID).
	paths := []string{
		filepath.Join(rootDir, projectID, runID, "runs"),
	}
	for _, runDir := range paths {
		entries, err := os.ReadDir(runDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			outputPath := filepath.Join(runDir, e.Name(), "output.md")
			data, err := os.ReadFile(outputPath)
			if err == nil {
				return strings.TrimSpace(string(data))
			}
		}
	}
	return ""
}

func detectIterateVerdict(reviewOutput string) string {
	upper := strings.ToUpper(reviewOutput)
	for _, tok := range iteratePassTokens {
		if strings.Contains(upper, tok) {
			return "pass"
		}
	}
	for _, tok := range iterateFailTokens {
		if strings.Contains(upper, tok) {
			return "fail"
		}
	}
	return "unknown"
}

func writeIterateSummary(rootDir, projectID, taskID string, results []iterationResult, outcome string) {
	taskDir := filepath.Join(rootDir, projectID, taskID)
	_ = os.MkdirAll(taskDir, 0o755)
	var sb strings.Builder
	sb.WriteString("# Iterate Summary\n\n")
	sb.WriteString(fmt.Sprintf("outcome: %s\n\n", outcome))
	sb.WriteString("| iteration | impl_run | review_run | verdict |\n")
	sb.WriteString("|-----------|----------|------------|---------|\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", r.iteration, r.implRunID, r.revRunID, r.verdict))
	}
	_ = os.WriteFile(filepath.Join(taskDir, "ITERATE-SUMMARY.md"), []byte(sb.String()), 0o644)
}
