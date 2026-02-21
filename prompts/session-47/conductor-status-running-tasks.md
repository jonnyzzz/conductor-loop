# Task: Enhance `conductor status` to Show Running Tasks

## Context

You are working on the conductor-loop project (Go-based multi-agent orchestration framework).
CWD: /Users/jonnyzzz/Work/conductor-loop

**Current state:**
- `go build ./...` passes, all 15 test packages green
- `conductor status` exists and shows counts of active/completed/failed runs + server uptime
- The `/api/v1/status` endpoint returns `active_runs`, `completed_runs`, `failed_runs`, `uptime`, `configured_agents`
- Missing: When `active_runs > 0`, there's no way to quickly see WHICH tasks are running

## Goal

Enhance the `/api/v1/status` endpoint and `conductor status` CLI command to show a list of currently-running tasks when there are active runs.

## What to Implement

### 1. Enhance the `/api/v1/status` API Response

File: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go`

In `handleStatus()`, add a `running_tasks` field to the response when there are active runs.

The new response shape should include:
```json
{
  "uptime": "1h23m",
  "active_runs": 2,
  "completed_runs": 45,
  "failed_runs": 3,
  "api_requests_total": 156,
  "configured_agents": ["claude", "codex"],
  "running_tasks": [
    {
      "project_id": "my-project",
      "task_id": "task-20260221-123456-my-task",
      "run_id": "20260221-1234560000-12345",
      "agent": "claude",
      "started": "2026-02-21T12:34:56Z"
    }
  ]
}
```

`running_tasks` should be:
- An empty array `[]` when `active_runs == 0`
- A list of running runs when `active_runs > 0`

To get running runs, scan `allRunInfos()` and filter for `status == "running"`.

The `runningTaskItem` struct:
```go
type runningTaskItem struct {
    ProjectID string    `json:"project_id"`
    TaskID    string    `json:"task_id"`
    RunID     string    `json:"run_id"`
    Agent     string    `json:"agent"`
    Started   time.Time `json:"started"`
}
```

Sort the running tasks by `Started` time (oldest first).

### 2. Enhance the `conductor status` CLI Output

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/status.go`

Currently the CLI displays:
```
Status: running
Uptime: 1h23m
Active runs: 2
Completed runs: 45
Failed runs: 3
Agents: claude, codex
```

Enhance it to show running tasks when present:
```
Status: running
Uptime: 1h23m
Active runs: 2
Completed runs: 45
Failed runs: 3
Agents: claude, codex

Running tasks:
  PROJECT          TASK                              RUN                     AGENT    STARTED
  my-project       task-20260221-123456-my-task      20260221-12345...       claude   12:34:56
  other-project    task-20260221-100000-other-task   20260221-10000...       claude   10:00:00
```

Implementation:
1. Parse the `running_tasks` field from the status JSON response
2. If `running_tasks` is non-empty, print a blank line then a `Running tasks:` header
3. Use `text/tabwriter` to format the table
4. Truncate run IDs to 20 chars in the table (they're long)
5. Format `Started` time as `15:04:05` (time only, local timezone)
6. If `started` is zero, show `-`

Update the status response struct to include the new field:
```go
type statusResponse struct {
    // ... existing fields ...
    RunningTasks []runningTaskItem `json:"running_tasks,omitempty"`
}

type runningTaskItem struct {
    ProjectID string    `json:"project_id"`
    TaskID    string    `json:"task_id"`
    RunID     string    `json:"run_id"`
    Agent     string    `json:"agent"`
    Started   time.Time `json:"started"`
}
```

### 3. Update Tests

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/status.go` (test file if one exists)
or create `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/status_test.go`

Check if there's an existing test file. If not, create one with basic tests for the status command parsing.

File: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_test.go` (or similar)

Add or update the test for `handleStatus` to verify that `running_tasks` appears in the response when there are active runs, and is empty when there are no active runs.

## Implementation Notes

- Look at the existing `handleStatus()` in `internal/api/handlers.go` to understand the current response structure
- The `allRunInfos()` function in `handlers.go` already scans all runs - use it to find running ones
- The `statusResponse` struct may be in the same file - update it
- The `conductor status` command is in `cmd/conductor/status.go` - update parsing and display there
- Be careful not to break `--json` output for scripts that parse it (adding new fields is OK)
- The `running_tasks` field should always be present in the JSON (even if empty array), but showing the table in text mode only when non-empty is fine

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./cmd/conductor/...` must pass
3. `go test ./internal/api/...` must pass
4. `go test -race ./...` must pass (check for data races)
5. `conductor status --json` must include `running_tasks` field

## What NOT to do

- Do NOT modify anything unrelated to the status feature
- Do NOT add extra flags or features beyond what's specified
- Do NOT refactor existing code
- Keep changes minimal and focused
