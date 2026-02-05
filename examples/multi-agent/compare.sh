#!/bin/bash

echo "========================================"
echo "MULTI-AGENT CODE REVIEW COMPARISON"
echo "========================================"
echo ""

# Find latest run for each agent
CLAUDE_RUN=$(ls -t runs/multi-agent-demo/review-claude/ 2>/dev/null | head -1)
CODEX_RUN=$(ls -t runs/multi-agent-demo/review-codex/ 2>/dev/null | head -1)
GEMINI_RUN=$(ls -t runs/multi-agent-demo/review-gemini/ 2>/dev/null | head -1)

# Function to display agent output
show_output() {
    local agent=$1
    local run_dir=$2

    if [ -z "$run_dir" ]; then
        echo "[$agent] No run found"
        return
    fi

    local output_file="runs/multi-agent-demo/review-$agent/$run_dir/output.md"
    local info_file="runs/multi-agent-demo/review-$agent/$run_dir/run-info.yaml"

    echo "========================================"
    echo "$agent OUTPUT"
    echo "========================================"

    if [ -f "$info_file" ]; then
        echo "Status: $(grep 'status:' $info_file | awk '{print $2}')"
        echo "Duration: $(grep 'start_time:' $info_file) to $(grep 'end_time:' $info_file)"
    fi

    echo ""

    if [ -f "$output_file" ]; then
        cat "$output_file"
    else
        echo "Warning: output.md not found"
        echo ""
        echo "Agent stderr:"
        cat "runs/multi-agent-demo/review-$agent/$run_dir/agent-stderr.txt" 2>/dev/null || echo "(no stderr)"
    fi

    echo ""
    echo ""
}

# Show each agent's output
show_output "claude" "$CLAUDE_RUN"
show_output "codex" "$CODEX_RUN"
show_output "gemini" "$GEMINI_RUN"

# Summary
echo "========================================"
echo "SUMMARY"
echo "========================================"
echo "Claude: $([ -n "$CLAUDE_RUN" ] && echo "✓ Completed" || echo "✗ Not found")"
echo "Codex:  $([ -n "$CODEX_RUN" ] && echo "✓ Completed" || echo "✗ Not found")"
echo "Gemini: $([ -n "$GEMINI_RUN" ] && echo "✓ Completed" || echo "✗ Not found")"
echo ""
echo "To view individual outputs:"
echo "  Claude: runs/multi-agent-demo/review-claude/$CLAUDE_RUN/output.md"
echo "  Codex:  runs/multi-agent-demo/review-codex/$CODEX_RUN/output.md"
echo "  Gemini: runs/multi-agent-demo/review-gemini/$GEMINI_RUN/output.md"
