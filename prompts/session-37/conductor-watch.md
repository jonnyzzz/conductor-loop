# Task: Add conductor watch command

## Context

You are a sub-agent working on the conductor-loop project. The `run-agent` CLI has a `watch`
command (`cmd/run-agent/watch.go`) that polls tasks until they complete. The `conductor` CLI
should have an equivalent `conductor watch` command that polls via the conductor server API.

## What to Implement

Add `conductor watch` command to `cmd/conductor/`:

### Behavior

```
conductor watch --project <p> [--task <t>] [--server URL] [--timeout 30m] [--json] [--interval 5s]
```

- Poll conductor server every `--interval` (default 5s) for task status
- `--task` can be specified multiple times (or comma-separated)
- If no `--task` given, watch ALL tasks in the project
- Exit 0 when all watched tasks reach a terminal state (completed/failed/done)
- Exit 1 on `--timeout` exceeded (default 30m)
- Print status table each poll cycle (like run-agent watch)
- `--json` outputs structured JSON with final task statuses

### API calls

Uses existing conductor server APIs:
- `GET /api/projects/{p}/tasks` — list tasks
- `GET /api/projects/{p}/tasks/{t}` — task detail (via the existing project-scoped API)

Note: conductor task list uses GET /api/projects/{p}/tasks (paginated response with `items` field).
The taskListAPIResponse struct is in task.go.

### Reference implementation

Read `cmd/run-agent/watch.go` and `cmd/run-agent/watch_test.go` as reference for:
- Poll loop structure
- Status table printing
- Timeout handling
- --task repeatable flag
- Test structure

### Output format (table mode)

```
Watching 2 task(s) in project 'conductor-loop'...
[2026-02-21 03:30:05] Poll #1
TASK ID                              STATUS     RUNS
task-20260221-012529-m711tm          completed   2
task-20260221-012532-fm8c1w          running     1

All tasks completed.
```

### Output format (JSON mode with --json)

On completion, print JSON array of task statuses:
```json
{"tasks":[{"task_id":"...","status":"completed","run_count":2},...],"all_done":true}
```

## Instructions

1. Read `cmd/run-agent/watch.go` and `cmd/run-agent/watch_test.go` for reference
2. Read `cmd/conductor/task.go` for existing task list/status functions to reuse
3. Read `cmd/conductor/main.go` to understand how commands are registered
4. Create `cmd/conductor/watch.go` with the watch command implementation
5. Create `cmd/conductor/watch_test.go` with at least 6 tests:
   - TestWatchAllTasksComplete (all tasks complete immediately)
   - TestWatchTimeout (tasks don't complete within timeout)
   - TestWatchSpecificTask (single task filtered)
   - TestWatchJSONOutput (JSON output format)
   - TestWatchEmptyProject (no tasks found)
   - TestWatchCmdHelp (command appears in help)
6. Register the watch command in `cmd/conductor/main.go`
7. Run: `go build -o bin/conductor ./cmd/conductor && go test ./cmd/conductor/`

## Quality

- Follow existing code style in cmd/conductor/
- Use testify assertions (github.com/stretchr/testify/assert or require)
- All tests must pass; no data races
- The watch command must appear in `./bin/conductor --help` output

## Completion

Create the DONE file at: $TASK_FOLDER/DONE
