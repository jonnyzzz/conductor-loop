#!/bin/bash
# Usage: ./run-round.sh <round-number>
# Launches 3 orchestrators in parallel for one round of the evolution pipeline.
# Each orchestrator runs 5 iterations of 3-agent workgroups internally.

ROUND=${1:-1}
cd /Users/jonnyzzz/Work/conductor-loop
ROOT=/Users/jonnyzzz/run-agent
TS=$(date +%Y%m%d-%H%M%S)

echo "=== EVOLUTION ROUND $ROUND === TS=$TS ==="

# Element 1: Consolidate facts & docs
./bin/run-agent job \
  --config config.local.yaml --agent claude --project conductor-loop \
  --task "task-${TS}-evo-r${ROUND}-consolidate" \
  --root "$ROOT" --cwd /Users/jonnyzzz/Work/conductor-loop \
  --prompt-file prompts/evolution/orch-consolidate.md \
  --timeout 90m &
PID1=$!
echo "consolidate: $PID1"

# Element 2: Architecture pages
./bin/run-agent job \
  --config config.local.yaml --agent codex --project conductor-loop \
  --task "task-${TS}-evo-r${ROUND}-architecture" \
  --root "$ROOT" --cwd /Users/jonnyzzz/Work/conductor-loop \
  --prompt-file prompts/evolution/orch-architecture.md \
  --timeout 90m &
PID2=$!
echo "architecture: $PID2"

# Element 3: Next tasks
./bin/run-agent job \
  --config config.local.yaml --agent gemini --project conductor-loop \
  --task "task-${TS}-evo-r${ROUND}-nexttasks" \
  --root "$ROOT" --cwd /Users/jonnyzzz/Work/conductor-loop \
  --prompt-file prompts/evolution/orch-next-tasks.md \
  --timeout 90m &
PID3=$!
echo "nexttasks: $PID3"

echo "Waiting for all 3 orchestrators (round $ROUND)..."
wait $PID1 $PID2 $PID3
EXIT=$?

echo "Round $ROUND complete (exit=$EXIT)"
git log --oneline -5

# Push after each round
git push origin main
echo "Pushed round $ROUND"
