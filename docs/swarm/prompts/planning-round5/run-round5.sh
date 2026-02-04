#!/bin/bash
set -euo pipefail
BASE_DIR="$(pwd)"
RUNS_DIR="${BASE_DIR}/runs6"
mkdir -p "$RUNS_DIR"
export RUNS_DIR

PROMPT="$BASE_DIR/prompts/planning-round5/review.md"

../run-agent.sh claude "$BASE_DIR" "$PROMPT" &
../run-agent.sh gemini "$BASE_DIR" "$PROMPT" &
../run-agent.sh codex "$BASE_DIR" "$PROMPT" &

wait
