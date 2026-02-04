#!/bin/bash
set -euo pipefail

BASE_DIR="$(pwd)"
RUNS_DIR="${BASE_DIR}/runs5"
mkdir -p "$RUNS_DIR"
export RUNS_DIR

PROMPT_FILES=(
  "$BASE_DIR"/prompts/planning-round4/subsystem-*.md
  "$BASE_DIR"/prompts/planning-round4/topic-*.md
)

run_all() {
  local agent="$1"
  printf "%s\n" "${PROMPT_FILES[@]}" | xargs -n 1 -P 4 -I {} ../run-agent.sh "$agent" "$BASE_DIR" "{}"
}

run_all claude
run_all gemini
