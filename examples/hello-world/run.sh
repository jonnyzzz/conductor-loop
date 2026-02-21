#!/bin/bash
set -e

echo "========================================"
echo "Conductor Loop - Hello World Example"
echo "========================================"
echo ""

# Check if conductor is available
if ! command -v conductor &> /dev/null; then
    echo "Error: 'conductor' command not found"
    echo "Please ensure Conductor Loop is installed and on your PATH"
    exit 1
fi

echo "Step 1: Starting Conductor server..."
run-agent serve --config config.yaml &
SERVER_PID=$!
echo "Server started (PID: $SERVER_PID)"
sleep 3

echo ""
echo "Step 2: Creating hello-world task..."
run-agent server job submit \
  --project-id hello-world \
  --task-id greeting \
  --agent codex \
  --prompt-file prompt.md

echo ""
echo "Step 3: Waiting for task to complete..."
sleep 10

echo ""
echo "Step 4: Viewing results..."
echo "========================================"

# Find the latest run directory
RUN_DIR=$(ls -t runs/hello-world/greeting/ | head -1)
if [ -z "$RUN_DIR" ]; then
    echo "Error: No run directory found"
    kill $SERVER_PID
    exit 1
fi

echo "Run directory: runs/hello-world/greeting/$RUN_DIR"
echo ""

# Display run info
echo "Run Info:"
echo "--------"
cat "runs/hello-world/greeting/$RUN_DIR/run-info.yaml"
echo ""

# Display output
echo "Output:"
echo "-------"
if [ -f "runs/hello-world/greeting/$RUN_DIR/output.md" ]; then
    cat "runs/hello-world/greeting/$RUN_DIR/output.md"
else
    echo "Warning: output.md not found"
    echo "Agent stdout:"
    cat "runs/hello-world/greeting/$RUN_DIR/agent-stdout.txt" || echo "(no stdout)"
    echo ""
    echo "Agent stderr:"
    cat "runs/hello-world/greeting/$RUN_DIR/agent-stderr.txt" || echo "(no stderr)"
fi

echo ""
echo "========================================"
echo "Example complete!"
echo "========================================"

# Cleanup
echo ""
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo "Done. Check runs/hello-world/greeting/$RUN_DIR/ for full output."
