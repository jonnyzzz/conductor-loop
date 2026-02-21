# Task: Add --status filter to conductor task list

## Context

You are implementing a feature improvement for the Conductor Loop project at /Users/jonnyzzz/Work/conductor-loop.

The project is a Go-based multi-agent orchestration framework. The conductor CLI has commands to manage projects and tasks via a REST API server.

## Current State

`conductor task list --project <project-id>` lists all tasks in a project, showing TASK ID, STATUS, RUNS, and LAST ACTIVITY. It calls `GET /api/projects/{id}/tasks`.

**Problem:** When a project has many tasks (e.g., dozens of dog-food sub-agent tasks), you can't filter to see only running tasks or only done tasks. You have to scan through all of them visually.

## Your Task

Add a `--status` filter flag to `conductor task list` and support it in the API.

### Part 1: API Query Parameter

Add `?status=<filter>` query parameter support to `handleProjectTasks` in `internal/api/handlers_projects.go`.

**Filter values:**
- `running` — tasks that have at least one run with no end_time (i.e., currently running)
- `done` — tasks where the DONE file exists at `<taskDir>/DONE`
- `failed` — tasks that have no DONE file and no running runs, and at least one failed run (exit_code != 0)
- `active` — tasks with running runs (same as `running`)
- empty/unset — all tasks (current behavior)

**Implementation:**
- In `handleProjectTasks`, read `r.URL.Query().Get("status")`
- After building the task list, apply the filter before returning
- Status values are case-insensitive

**Task status determination** (for filtering):
- A task is "running" if any run in `listTaskRunInfos()` has `EndTime.IsZero()`
- A task is "done" if the file `<taskDir>/DONE` exists
- A task is "failed" if not done AND not running AND at least one run has `ExitCode != 0`

**Note:** The existing `taskInfo` struct already has a `Status` field computed by `buildTaskInfo()`. Use that status string for the filter:
- Status "running" → matches `running` or `active` filter
- Status "done" → matches `done` filter
- Status "failed" → matches `failed` filter

### Part 2: CLI Flag

Add `--status` flag to `conductor task list` in `cmd/conductor/task.go`.

**Usage:**
```
conductor task list --project <project-id> [--status running|done|failed|active] [--server URL] [--json]
```

**Example:**
```bash
# Show only running tasks
conductor task list --project conductor-loop --status running

# Show only done tasks
conductor task list --project conductor-loop --status done
```

Pass the `--status` value as `?status=<value>` query parameter to the API.

### Part 3: Tests

Add tests in `internal/api/handlers_projects_test.go` for the status filter:
- Test `?status=running` returns only running tasks
- Test `?status=done` returns only done tasks
- Test `?status=failed` returns only failed tasks
- Test empty status returns all tasks
- Test unknown status value returns all tasks (graceful degradation)

Add tests in `cmd/conductor/commands_test.go` for the CLI flag:
- Test `--status running` passes `?status=running` to the API
- Test `--status done` passes `?status=done` to the API
- Test without `--status` makes request without status parameter

### Important Implementation Notes

1. Read the existing `handleProjectTasks` function carefully in `internal/api/handlers_projects.go` to understand how tasks are listed and returned.

2. Look at the `buildTaskInfo` function to understand what the `Status` field contains for different task states.

3. The filter should be applied AFTER the task list is built (post-processing), to keep the filtering logic separate from the listing logic.

4. For the "running" status determination, check the `Status` field in `taskInfo` — the existing code already computes this correctly.

5. Do NOT change the existing behavior when no status filter is provided.

## Quality Requirements

1. `go build ./...` must pass
2. `go test -race ./internal/... ./cmd/...` must pass (no new test failures)
3. Follow existing code style
4. Commit with message: `feat(api+cli): add --status filter to conductor task list`

## How to Work

1. Read relevant files first:
   - `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` (look at `handleProjectTasks` and `buildTaskInfo`)
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go` (look at `taskList` and `newTaskListCmd`)
   - `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go` (look at existing tests for `handleProjectTasks`)
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/commands_test.go`

2. Implement changes
3. Build and test
4. Commit

Working directory: /Users/jonnyzzz/Work/conductor-loop
