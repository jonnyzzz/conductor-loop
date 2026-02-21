# Task: Add `conductor task resume` Command

You are a software engineer working on the Conductor Loop project. Your task is to add a `conductor task resume` client command that calls a new API endpoint to resume an exhausted task.

## Working Directory
/Users/jonnyzzz/Work/conductor-loop

## Background

The `run-agent resume` command (added in session #40) resets an exhausted task by removing the DONE file so the Ralph loop can run again. However, there is no server-side equivalent for users of the `conductor` client CLI.

Currently, `conductor task` has:
- `conductor task list`
- `conductor task status`
- `conductor task stop`
- `conductor task delete`

We need to add `conductor task resume`.

## Implementation Plan

### Step 1: Add API endpoint `POST /api/v1/tasks/{project}/{task}/resume`

In `internal/api/`, add a handler that:
1. Accepts `POST /api/v1/projects/{projectId}/tasks/{taskId}/resume` (via the projects router)
   OR adds a new route for `POST /api/v1/tasks/{taskId}/resume` with project query param

   **PREFERRED APPROACH**: Look at how `handleTaskByID` handles task stop (look in handlers.go for handleTaskStop or similar). The existing task routes use `/api/v1/tasks/` with project/task path segments OR query params. Check the existing `conductor task stop` implementation to see how it calls the API, then mirror that pattern.

2. The handler should:
   - Find the task directory: `<root>/<project>/<task>/`
   - Delete the DONE file if it exists: `<root>/<project>/<task>/DONE`
   - Return 200 OK with `{"project_id": "...", "task_id": "...", "resumed": true}`
   - Return 404 if the task directory doesn't exist
   - Return 400 if the task has no DONE file (nothing to resume)

### Step 2: Add `conductor task resume` client command

In `cmd/conductor/`, create `task_resume.go` following the same pattern as `task_stop.go`:
1. Command: `conductor task resume <task-id> --project <project> [--server URL]`
2. Calls the new API endpoint
3. Prints result or error
4. Handles --json flag for machine-readable output

### Step 3: Wire up the command

In `cmd/conductor/main.go`, add the resume command to the task sub-command (like task stop is wired).

## Reading Before Coding

Read these files FIRST to understand the existing patterns:
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go` - Find handleTaskStop or handleTaskByID for the stop operation
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go` - See how routes are organized
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task_stop.go` - Mirror this pattern
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go` - See how to wire up a command
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/resume.go` - See the existing resume logic (for reference on what to do at the file level)

## Tests

Add tests in:
- `internal/api/handlers_test.go` - Test the new resume endpoint (POST with DONE file → 200, no DONE file → 400, no task → 404)
- `cmd/conductor/task_resume_test.go` - Integration test for the CLI command (optional but preferred)

## Build and Verify

After implementation:
```bash
go build ./...
go test ./internal/api/... ./cmd/conductor/...
./bin/conductor task resume --help  # verify command exists
```

## Commit

If all tests pass, commit with:
```
feat(api): add task resume endpoint and conductor task resume command
```

## Output

Write output.md to $RUN_FOLDER/output.md with:
- Summary of what was implemented
- Which files were created/modified
- Test results
- Any issues encountered
