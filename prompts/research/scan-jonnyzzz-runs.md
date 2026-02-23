# Research Task: Scan jonnyzzz-ai-coder Runs for Tasks & Requests

You are a research agent. Your goal is to extract all tasks, requests, goals, and open work items
from run directories in `/Users/jonnyzzz/Work/jonnyzzz-ai-coder/runs/` and
`/Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm/runs/`.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-jonnyzzz.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: runs, task, <status>, <subsystem>]
<fact text>

```

## Step 1: Enumerate recent orchestration runs in jonnyzzz-ai-coder/runs/

```bash
ls /Users/jonnyzzz/Work/jonnyzzz-ai-coder/runs/ | sort | tail -50
```

Focus on recent entries (2026-02 onwards).

## Step 2: Scan orchestration directories

```bash
ROOT=/Users/jonnyzzz/Work/jonnyzzz-ai-coder/runs

# List orchestration directories
for dir in $ROOT/orchestration-*/; do
  echo "=== $(basename $dir) ==="
  ls "$dir" | head -10
  # Read any summary/plan files
  for f in "$dir"/*.md "$dir"/*.txt; do
    [ -f "$f" ] && echo "--- $(basename $f) ---" && head -30 "$f"
  done
done

# Read recent phase files
for f in $ROOT/phase*.md $ROOT/phase*.log; do
  [ -f "$f" ] && echo "=== $(basename $f) ===" && head -40 "$f"
done
```

## Step 3: Scan swarm/runs for recent tasks

```bash
# List recent swarm run directories
ls /Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm/runs/ | sort | tail -30

ROOT=/Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm/runs

# Read recent run summaries
for dir in $(ls -t $ROOT/ | head -20); do
  echo "=== $dir ==="
  if [ -f "$ROOT/$dir/prompt.md" ]; then
    echo "--- PROMPT (first 20 lines) ---"
    head -20 "$ROOT/$dir/prompt.md"
  fi
  if [ -f "$ROOT/$dir/output.md" ]; then
    echo "--- OUTPUT (last 20 lines) ---"
    tail -20 "$ROOT/$dir/output.md"
  fi
  echo ""
done
```

## Step 4: Read launch logs for task patterns

```bash
ROOT=/Users/jonnyzzz/Work/jonnyzzz-ai-coder/runs

# Read recent launch logs to see what was attempted
for log in $(ls -t $ROOT/*.log 2>/dev/null | head -10); do
  echo "=== $(basename $log) ==="
  tail -20 "$log"
done

# Read conductor-loop related runs
for f in $ROOT/*.md; do
  [ -f "$f" ] && echo "=== $(basename $f) ===" && cat "$f" | head -60
done
```

## Step 5: Check conductor-loop runs folder inside conductor-loop repo

```bash
ls /Users/jonnyzzz/Work/conductor-loop/runs/ 2>/dev/null | head -20
ls /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/ 2>/dev/null | tail -30
```

## Step 6: Read TODOs.md, ISSUES.md, and MESSAGE-BUS.md for pending items

```bash
cat /Users/jonnyzzz/Work/conductor-loop/TODOs.md 2>/dev/null | head -100
cat /Users/jonnyzzz/Work/conductor-loop/ISSUES.md 2>/dev/null | grep -E "Status:|OPEN|PENDING" | head -30
grep -E "TODO|PENDING|OPEN|REQUEST|QUESTION" /Users/jonnyzzz/Work/conductor-loop/MESSAGE-BUS.md 2>/dev/null | head -30
```

## Step 7: Write FACTS file

Document:
1. All tasks/requests found in run directories with their dates and goals
2. Completion status of each
3. Recurring patterns or themes
4. Any open/blocked work
5. Key outcomes or decisions

Focus on items NOT already in FACTS-runs-conductor.md (those cover /run-agent/conductor-loop/).
This file covers: jonnyzzz-ai-coder/runs/, jonnyzzz-ai-coder/swarm/runs/, conductor-loop/runs/.
