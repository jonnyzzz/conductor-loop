package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/goaldecompose"
	"github.com/spf13/cobra"
)

func newGoalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goal",
		Short: "Goal decomposition utilities",
	}

	cmd.AddCommand(newGoalDecomposeCmd())
	return cmd
}

func newGoalDecomposeCmd() *cobra.Command {
	var (
		projectID   string
		goalText    string
		goalFile    string
		rootDir     string
		strategy    string
		template    string
		maxParallel int
		jsonOut     bool
		outPath     string
	)

	cmd := &cobra.Command{
		Use:   "decompose",
		Short: "Generate a deterministic workflow spec from a project goal",
		RunE: func(cmd *cobra.Command, args []string) error {
			goal, mode, source, err := loadConductorGoalInput(goalText, goalFile)
			if err != nil {
				return err
			}

			spec, err := goaldecompose.BuildSpec(goaldecompose.BuildOptions{
				ProjectID:   strings.TrimSpace(projectID),
				GoalText:    goal,
				GoalMode:    mode,
				GoalSource:  source,
				RootDir:     strings.TrimSpace(rootDir),
				Strategy:    strings.TrimSpace(strategy),
				Template:    strings.TrimSpace(template),
				MaxParallel: maxParallel,
			})
			if err != nil {
				return err
			}

			stdoutFormat := goaldecompose.OutputFormatYAML
			if jsonOut {
				stdoutFormat = goaldecompose.OutputFormatJSON
			}
			stdoutData, err := goaldecompose.EncodeSpec(spec, stdoutFormat)
			if err != nil {
				return fmt.Errorf("encode workflow spec: %w", err)
			}
			if _, err := cmd.OutOrStdout().Write(stdoutData); err != nil {
				return fmt.Errorf("write output: %w", err)
			}

			if strings.TrimSpace(outPath) != "" {
				cleanOut := filepath.Clean(strings.TrimSpace(outPath))
				fileFormat := goaldecompose.OutputFormatFromPath(cleanOut)
				fileData := stdoutData
				if fileFormat != stdoutFormat {
					fileData, err = goaldecompose.EncodeSpec(spec, fileFormat)
					if err != nil {
						return fmt.Errorf("encode %s output: %w", fileFormat, err)
					}
				}
				if err := os.MkdirAll(filepath.Dir(cleanOut), 0o755); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
				if err := os.WriteFile(cleanOut, fileData, 0o644); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project id")
	cmd.Flags().StringVar(&goalText, "goal", "", "inline goal text (mutually exclusive with --goal-file)")
	cmd.Flags().StringVar(&goalFile, "goal-file", "", "path to a goal file (mutually exclusive with --goal)")
	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory hint stored in spec metadata")
	cmd.Flags().StringVar(&strategy, "strategy", goaldecompose.DefaultStrategy, "decomposition strategy (currently only rlm)")
	cmd.Flags().StringVar(&template, "template", goaldecompose.DefaultTemplate, "orchestration prompt template")
	cmd.Flags().IntVar(&maxParallel, "max-parallel", goaldecompose.DefaultMaxParallel, "maximum parallel tasks in generated workflow")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON (default is YAML)")
	cmd.Flags().StringVar(&outPath, "out", "", "write workflow spec to file (format inferred from extension: .json or .yaml/.yml)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func loadConductorGoalInput(goalText, goalFile string) (goal string, mode string, source string, err error) {
	goalText = strings.TrimSpace(goalText)
	goalFile = strings.TrimSpace(goalFile)

	switch {
	case goalText != "" && goalFile != "":
		return "", "", "", fmt.Errorf("--goal and --goal-file are mutually exclusive")
	case goalText == "" && goalFile == "":
		return "", "", "", fmt.Errorf("one of --goal or --goal-file is required")
	case goalFile != "":
		data, readErr := os.ReadFile(goalFile)
		if readErr != nil {
			return "", "", "", fmt.Errorf("read goal file: %w", readErr)
		}
		goal = strings.TrimSpace(string(data))
		if goal == "" {
			return "", "", "", fmt.Errorf("goal file %q is empty", goalFile)
		}
		return goal, "file", filepath.ToSlash(filepath.Clean(goalFile)), nil
	default:
		return goalText, "inline", "inline", nil
	}
}
