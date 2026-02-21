# Task: Web UI Auto-Refresh for Task List + Project/Task Navigation

## Overview

Improve the conductor-loop web UI in two areas:

1. **Auto-refresh task list** — Currently the task list for a selected project does NOT auto-update when new runs arrive. Add auto-refresh so the task/run list refreshes every 5 seconds when a project is selected.

2. **Run status badges** — Update the run status display to show more informative badges and durations for completed/failed runs.

3. **Web UI: display run count and active count in task header** — When viewing task runs, show the count of total runs and how many are currently running.

## Context

Read these files first:
- `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md` — conventions, commit format
- `/Users/jonnyzzz/Work/conductor-loop/Instructions.md` — build commands
- `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js` — current implementation
- `/Users/jonnyzzz/Work/conductor-loop/web/src/index.html` — current HTML
- `/Users/jonnyzzz/Work/conductor-loop/web/src/styles.css` — current CSS

## What to implement

### Part 1: Auto-refresh task list when project is selected

Currently `refreshTimer` is set globally. When a project is selected:
- The `loadTasks()` function loads tasks once
- The `renderMainPanel()` shows tasks and runs

**Goal**: When `state.selectedProject` is set, refresh the task list + run list periodically.

**Implementation**:
- In `loadTasks()`, after successfully loading, schedule a refresh after `REFRESH_MS` (5000ms)
- When the project changes, cancel the previous timer and start fresh
- The timer should only be active when a project is selected
- Use `clearTimeout` + `setTimeout` pattern (similar to existing `refreshTimer`)
- When a task is selected, also refresh the run list for that task

### Part 2: Better run duration display

In `renderRunsSection()`, improve the duration display:
- For running tasks: show elapsed time since start (e.g., "3m 12s")
- For completed/failed tasks: show total duration (already implemented)
- Use `fmtDuration()` helper which already exists

Check if `fmtDuration` can handle "now" (open-ended duration) — if `end_time` is null/empty, pass `Date.now()` as the end.

### Part 3: Task run count in header

When a task is selected and expanded to show runs, update the task header to show:
- Count of total runs for that task
- Count of currently running runs

Example: "my-task [3 runs, 1 running]" or just "my-task (2 runs)"

This is a minor UI enhancement. Keep it simple.

## Quality Gates

1. `go build ./...` must pass (no Go changes expected, but verify)
2. `go test ./...` — all 18 packages must pass
3. Ensure no JS syntax errors (review carefully)

## Commit format

```
feat(ui): add task list auto-refresh and improved run display

- Auto-refresh task/run list every 5s when project is selected
- Show elapsed time for running tasks in run list
- Show run count in task header when expanded
```

## Output

Write your output summary to `$RUN_FOLDER/output.md` and create `$TASK_FOLDER/DONE` when complete.
