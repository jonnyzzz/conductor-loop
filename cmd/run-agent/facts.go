package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/facts"
	"github.com/spf13/cobra"
)

func newFactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts",
		Short: "Manage project facts (cross-task knowledge sharing)",
	}
	cmd.AddCommand(newFactsPromoteCmd())
	return cmd
}

func newFactsPromoteCmd() *cobra.Command {
	var (
		projectID  string
		root       string
		dryRun     bool
		filterType string
		sinceStr   string
	)

	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote FACT messages from all task buses into PROJECT-FACTS.md",
		Long: `Scan all task message buses in a project and promote messages of the
specified type into <root>/<project>/PROJECT-FACTS.md.

Facts are deduplicated by content hash, so re-running is safe and idempotent.
With --dry-run, no file is written but the count of promotable facts is shown.

Example:
  run-agent facts promote --project my-project --root ./runs
  run-agent facts promote --project my-project --root ./runs --dry-run
  run-agent facts promote --project my-project --root ./runs --filter-type DECISION`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = strings.TrimSpace(projectID)
			if projectID == "" {
				projectID = strings.TrimSpace(os.Getenv("JRUN_PROJECT_ID"))
			}
			if projectID == "" {
				return fmt.Errorf("--project is required (or set JRUN_PROJECT_ID env var)")
			}

			var rootErr error
			root, rootErr = config.ResolveRunsDir(root)
			if rootErr != nil {
				return fmt.Errorf("resolve runs dir: %w", rootErr)
			}

			var since time.Time
			if sinceStr != "" {
				var parseErr error
				since, parseErr = time.Parse(time.RFC3339, sinceStr)
				if parseErr != nil {
					return fmt.Errorf("invalid --since value %q: must be RFC3339 (e.g. 2026-01-01T00:00:00Z): %w", sinceStr, parseErr)
				}
			}

			cfg := facts.PromoteConfig{
				RootDir:    root,
				ProjectID:  projectID,
				DryRun:     dryRun,
				FilterType: filterType,
				Since:      since,
			}

			promoted, already, err := facts.PromoteFacts(cfg)
			if err != nil {
				return fmt.Errorf("promote facts: %w", err)
			}

			if dryRun {
				fmt.Printf("dry-run: would promote %d fact(s) (%d already present)\n", promoted, already)
			} else {
				fmt.Printf("promoted %d fact(s), %d already present\n", promoted, already)
				if promoted > 0 {
					factsFile := root + "/" + projectID + "/PROJECT-FACTS.md"
					fmt.Printf("written: %s\n", factsFile)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID (or set JRUN_PROJECT_ID env var)")
	cmd.Flags().StringVar(&root, "root", "", "root directory for projects (default: ~/.run-agent/runs)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be promoted without writing the file")
	cmd.Flags().StringVar(&filterType, "filter-type", "FACT", "message type to promote (default: FACT)")
	cmd.Flags().StringVar(&sinceStr, "since", "", "only promote messages after this time (RFC3339, e.g. 2026-01-01T00:00:00Z)")

	return cmd
}
