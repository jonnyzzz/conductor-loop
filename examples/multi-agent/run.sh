#!/bin/bash
set -e

echo "========================================"
echo "Multi-Agent Code Review Comparison"
echo "========================================"
echo ""

# Check if conductor is available
if ! command -v conductor &> /dev/null; then
    echo "Error: 'conductor' command not found"
    exit 1
fi

echo "Starting Conductor server..."
run-agent serve --config config.yaml &
SERVER_PID=$!
sleep 3

echo ""
echo "Running code review with 3 agents in parallel..."
echo "This may take 1-2 minutes..."
echo ""

# Create tasks for all three agents
echo "[1/3] Starting Claude review..."
run-agent server job submit \
  --project-id multi-agent-demo \
  --task-id review-claude \
  --agent claude \
  --prompt-file prompts/code-review.md &
CLAUDE_PID=$!

echo "[2/3] Starting Codex review..."
run-agent server job submit \
  --project-id multi-agent-demo \
  --task-id review-codex \
  --agent codex \
  --prompt-file prompts/code-review.md &
CODEX_PID=$!

echo "[3/3] Starting Gemini review..."
run-agent server job submit \
  --project-id multi-agent-demo \
  --task-id review-gemini \
  --agent gemini \
  --prompt-file prompts/code-review.md &
GEMINI_PID=$!

# Wait for all tasks to complete
echo ""
echo "Waiting for all agents to complete..."
wait $CLAUDE_PID 2>/dev/null || echo "Claude task submitted"
wait $CODEX_PID 2>/dev/null || echo "Codex task submitted"
wait $GEMINI_PID 2>/dev/null || echo "Gemini task submitted"

echo ""
echo "Waiting additional 30 seconds for agents to finish..."
sleep 30

echo ""
echo "========================================"
echo "Reviews complete! Running comparison..."
echo "========================================"

# Run comparison script
./compare.sh

# Cleanup
echo ""
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "Done! Check runs/multi-agent-demo/ for detailed output."
