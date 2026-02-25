package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/spf13/cobra"
)

// reviewApprovalTokens are case-insensitive body substrings indicating approval.
var reviewApprovalTokens = []string{"APPROVED", "LGTM", "+1"}

// reviewRejectionTokens are case-insensitive body substrings indicating rejection.
var reviewRejectionTokens = []string{"REJECTED", "BLOCKED", "CHANGES_REQUESTED"}

func newReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review quality gates for multi-agent workflows",
	}
	cmd.AddCommand(newReviewQuorumCmd())
	return cmd
}

func newReviewQuorumCmd() *cobra.Command {
	var (
		root      string
		projectID string
		taskID    string
		runs      string
		required  int
	)

	cmd := &cobra.Command{
		Use:   "quorum",
		Short: "Check review quorum from message bus evidence",
		Long: `Check whether enough independent reviewers approved a change.

Reads DECISION and REVIEW messages from the task/project bus filtered to
the provided run IDs. Counts approval tokens (APPROVED, LGTM, +1) and
rejection tokens (REJECTED, BLOCKED, CHANGES_REQUESTED).

Exit 0 when approvals >= --required AND no rejections.
Exit 1 otherwise.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runs = strings.TrimSpace(runs)
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			if runs == "" {
				return fmt.Errorf("--runs is required")
			}

			runIDs := splitAndTrim(runs, ",")
			if len(runIDs) == 0 {
				return fmt.Errorf("--runs is empty")
			}
			if required <= 0 {
				required = 2
			}
			var rootErr error
			root, rootErr = config.ResolveRunsDir(root)
			if rootErr != nil {
				return fmt.Errorf("resolve runs dir: %w", rootErr)
			}

			return runReviewQuorum(cmd.OutOrStdout(), root, projectID, taskID, runIDs, required)
		},
	}

	cmd.Flags().StringVar(&root, "root", "", "run-agent root directory (default: ~/.run-agent/runs)")
	cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&taskID, "task", "", "task id (optional, scopes to task bus)")
	cmd.Flags().StringVar(&runs, "runs", "", "comma-separated run IDs to consider")
	cmd.Flags().IntVar(&required, "required", 2, "number of approvals required for quorum (default: 2)")

	return cmd
}

type quorumResult struct {
	approvals  int
	rejections int
	approvalMsgIDs []string
	rejectionMsgIDs []string
}

func runReviewQuorum(out io.Writer, root, projectID, taskID string, runIDs []string, required int) error {
	busPath := quorumBusPath(root, projectID, taskID)
	mb, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return fmt.Errorf("open message bus: %w", err)
	}

	msgs, err := mb.ReadMessages("")
	if err != nil {
		return fmt.Errorf("read messages: %w", err)
	}

	runIDSet := make(map[string]struct{}, len(runIDs))
	for _, id := range runIDs {
		runIDSet[id] = struct{}{}
	}

	result := quorumResult{}
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		// Filter to relevant message types.
		msgType := strings.ToUpper(msg.Type)
		if msgType != "DECISION" && msgType != "REVIEW" {
			continue
		}
		// Filter to messages from provided run IDs.
		if len(runIDSet) > 0 && msg.RunID != "" {
			if _, ok := runIDSet[msg.RunID]; !ok {
				continue
			}
		}

		body := strings.ToUpper(msg.Body)
		if containsAny(body, reviewApprovalTokens) {
			result.approvals++
			result.approvalMsgIDs = append(result.approvalMsgIDs, msg.MsgID)
		} else if containsAny(body, reviewRejectionTokens) {
			result.rejections++
			result.rejectionMsgIDs = append(result.rejectionMsgIDs, msg.MsgID)
		}
	}

	fmt.Fprintf(out, "review quorum check\n")
	fmt.Fprintf(out, "  runs considered: %s\n", strings.Join(runIDs, ", "))
	fmt.Fprintf(out, "  approvals: %d (required: %d)\n", result.approvals, required)
	fmt.Fprintf(out, "  rejections: %d\n", result.rejections)
	if len(result.approvalMsgIDs) > 0 {
		fmt.Fprintf(out, "  approval messages: %s\n", strings.Join(result.approvalMsgIDs, ", "))
	}
	if len(result.rejectionMsgIDs) > 0 {
		fmt.Fprintf(out, "  rejection messages: %s\n", strings.Join(result.rejectionMsgIDs, ", "))
	}

	if result.approvals >= required && result.rejections == 0 {
		fmt.Fprintf(out, "result: QUORUM MET\n")
		return nil
	}

	reason := ""
	if result.rejections > 0 {
		reason = fmt.Sprintf("rejection veto (%d rejection(s))", result.rejections)
	} else {
		reason = fmt.Sprintf("insufficient approvals (%d/%d)", result.approvals, required)
	}
	fmt.Fprintf(out, "result: QUORUM NOT MET — %s\n", reason)
	return fmt.Errorf("quorum not met: %s", reason)
}

func quorumBusPath(root, projectID, taskID string) string {
	if taskID != "" {
		return filepath.Join(root, projectID, taskID, "TASK-MESSAGE-BUS.md")
	}
	return filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
}

func containsAny(s string, tokens []string) bool {
	for _, tok := range tokens {
		if strings.Contains(s, strings.ToUpper(tok)) {
			return true
		}
	}
	return false
}
