# Task: Add `run-agent watch` command

## Context

You are a sub-agent working on the conductor-loop project. This is a Go-based multi-agent
orchestration framework. The CLI tool is `run-agent`.

**Working directory**: /Users/jonnyzzz/Work/conductor-loop

## Background

When orchestrating multiple parallel sub-agents using `./bin/run-agent job`, the parent
orchestrator needs to monitor all running tasks. Currently the only way is to:
1. Check run-info.yaml files manually
2. Use `./bin/run-agent list --project <p> --task <t>` for each task
3. Use the web UI

A `run-agent watch` command would continuously monitor one or more tasks and exit when
they all complete, making it easy to synchronize parallel agent runs.

## What to implement

### `run-agent watch` command

Add a new `watch` subcommand to `cmd/run-agent/`:

```
watch --project <project> --task <task1> [--task <task2>] ... --root <root>
```

**Behavior:**
- Polls run status for specified tasks every 2 seconds
- Prints a one-line status table every 2 seconds:
  ```
  [2026-02-21 00:01:00] task-20260221-... running (1m23s elapsed)
  [2026-02-21 00:01:00] task-20260221-... completed (0m45s)
  [2026-02-21 00:01:00] Waiting for 1 task to complete...
  ```
- Exits 0 when ALL specified tasks are completed (status=completed or status=failed)
- `--timeout` flag (default 30m): exit with code 1 if tasks don't finish within timeout
- `--project` is required; `--task` can be specified multiple times (or comma-separated)
- `--root` defaults to RUNS_DIR env var or `./runs`

**Output format (one status line per task, refreshed each poll):**
```
Watching 3 tasks for project 'conductor-loop':
  task-abc... [running]  elapsed: 0m42s
  task-def... [completed] duration: 1m15s
  task-ghi... [running]  elapsed: 0m42s
Waiting for 2 running tasks... (timeout in 29m18s)
```

Or JSON output with `--json`:
```json
{"tasks": [{"task_id": "...", "status": "running", "elapsed": 42}], "all_done": false}
```

**Implementation location**: `cmd/run-agent/watch.go`

Use `internal/storage.ReadRunInfo()` to read run status from run-info.yaml files.
Use `./bin/run-agent list` logic to find the latest run for each task.

### Structure

```go
package main

import (
    "github.com/spf13/cobra"
    // ...
)

func newWatchCmd() *cobra.Command {
    var projectID string
    var taskIDs []string
    var rootDir string
    var timeout time.Duration
    var jsonOutput bool

    cmd := &cobra.Command{
        Use:   "watch",
        Short: "Watch tasks until completion",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runWatch(rootDir, projectID, taskIDs, timeout, jsonOutput)
        },
    }
    cmd.Flags().StringVar(&projectID, "project", "", "project id (required)")
    cmd.Flags().StringArrayVar(&taskIDs, "task", nil, "task id(s) to watch (can repeat)")
    cmd.Flags().StringVar(&rootDir, "root", defaultRunsDir(), "runs root directory")
    cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "max wait time")
    cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
    cmd.MarkFlagRequired("project")
    return cmd
}
```

Add to `cmd/run-agent/main.go` (or wherever commands are registered).

### Tests

Create `cmd/run-agent/watch_test.go` with at least 5 tests:
- Watch single completed task exits immediately
- Watch single running task waits and exits when done
- Watch multiple tasks exits when all done
- Timeout causes exit code 1
- Empty task list returns error

## Files to create/modify

1. `cmd/run-agent/watch.go` - new watch command (create new file)
2. `cmd/run-agent/watch_test.go` - tests (create new file)
3. `cmd/run-agent/main.go` - register watch command

## After making code changes

1. Build: `cd /Users/jonnyzzz/Work/conductor-loop && go build ./...`
2. Test: `cd /Users/jonnyzzz/Work/conductor-loop && go test ./cmd/run-agent/...`
3. Verify: `./bin/run-agent watch --help` shows correct help text
4. Run all tests: `cd /Users/jonnyzzz/Work/conductor-loop && go test ./...`

## Quality gates

- `go build ./...` passes
- `go test ./cmd/run-agent/...` passes (all watch tests pass)
- `./bin/run-agent watch --help` works correctly
- No data races: `go test -race ./cmd/run-agent/...`

## Commit format

```
feat(cli): add run-agent watch command for monitoring task completion

- watch --project <p> --task <t> polls task status until completion
- Supports multiple tasks, --timeout flag (default 30m), --json output
- Exits 0 when all tasks complete, 1 on timeout
- 5 new tests covering single/multi-task and timeout scenarios
```

Write "done" to the DONE file in TASK_FOLDER env var when complete.
