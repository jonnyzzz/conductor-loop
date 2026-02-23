# Research Task: Scan All conductor-loop Run-Agent Task Directories

You are a research agent. Your goal is to extract all tasks, requests, goals, and open work items
from every run-agent task directory in `/Users/jonnyzzz/run-agent/conductor-loop/`.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: runs, task, <status>, <subsystem>]
<fact text>

```

## Step 1: Enumerate all task directories

```bash
ls /Users/jonnyzzz/run-agent/conductor-loop/ | grep -v "^PROJECT-" | sort
```

## Step 2: For each task directory, extract task info

For each task dir (skip the already-known facts/validate tasks from 2026-02-23):

```bash
ROOT=/Users/jonnyzzz/run-agent/conductor-loop

for task_dir in $ROOT/task-*/; do
  task=$(basename "$task_dir")
  # Read TASK.md to get the original request
  if [ -f "$task_dir/TASK.md" ]; then
    echo "=== $task ==="
    cat "$task_dir/TASK.md"
    echo ""
  fi

  # Check DONE status
  if [ -f "$task_dir/DONE" ]; then
    echo "STATUS: DONE"
  else
    echo "STATUS: NOT DONE"
  fi

  # Get latest run's output.md summary (last 30 lines)
  latest=$(ls -t "$task_dir/runs/" 2>/dev/null | head -1)
  if [ -n "$latest" ] && [ -f "$task_dir/runs/$latest/output.md" ]; then
    echo "--- OUTPUT SUMMARY ---"
    tail -30 "$task_dir/runs/$latest/output.md"
  fi

  echo "=========================="
done
```

## Step 3: Check TASK-MESSAGE-BUS.md for each task

```bash
ROOT=/Users/jonnyzzz/run-agent/conductor-loop
for task_dir in $ROOT/task-*/; do
  task=$(basename "$task_dir")
  if [ -f "$task_dir/TASK-MESSAGE-BUS.md" ]; then
    size=$(wc -c < "$task_dir/TASK-MESSAGE-BUS.md")
    if [ "$size" -gt 100 ]; then
      echo "=== $task BUS ==="
      grep -E "DECISION:|FACT:|ANSWER:|RESULT:|DONE|completed|SUCCESS|FAIL" "$task_dir/TASK-MESSAGE-BUS.md" | head -10
    fi
  fi
done
```

## Step 4: Check PROJECT-MESSAGE-BUS.md

```bash
cat /Users/jonnyzzz/run-agent/conductor-loop/PROJECT-MESSAGE-BUS.md | head -200
```

## Step 5: Write FACTS file

Document:
1. Each task: its ID, date, original request/goal, completion status, outcome
2. Open/incomplete tasks with what was attempted
3. Recurring themes, patterns across tasks
4. Failed tasks with root causes if known
5. New facts not in existing FACTS files

Focus on tasks from 2026-02-20 through 2026-02-23 (excluding the facts/validate tasks from 2026-02-23 afternoon which we already know about).

Group facts by:
- Completed implementation tasks
- Failed/blocked tasks
- Open/pending tasks
- Key decisions made during runs
