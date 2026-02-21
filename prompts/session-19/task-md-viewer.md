# Task: Add TASK.md Viewer to Web UI

## Context

This is the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
You are a Go and JavaScript developer implementing a new feature.

## Goal

Add the ability to view TASK.md content in the web monitoring UI.
Currently the web UI shows output.md, stdout, stderr, and prompt files
but there is no way to see the TASK.md (task description/brief).

## What to implement

### 1. Backend: Add task-level file endpoint

File: `internal/api/handlers_projects.go`

The `handleProjectTask` function routes requests to
`/api/projects/{p}/tasks/{t}/runs/{r}/...` (run-scoped).

Add a NEW route pattern for task-scoped files:
`GET /api/projects/{p}/tasks/{t}/file?name=TASK.md`

In `handleProjectTask`, before checking for `parts[5]` (run-level sub-paths),
check if `len(parts) == 4 && parts[3] == "file"` (or similar), and
serve the task-level file from the task directory.

The task directory is: `<rootDir>/<projectId>/<taskId>/`
Only allow `name=TASK.md` for security. Return the file content as text/plain.
If the file doesn't exist, return 404.

Add a test case in `internal/api/handlers_projects_test.go` for this new endpoint.

### 2. Frontend: Add TASK.MD tab in task detail

File: `web/src/app.js`

Currently when a run is selected, there are tabs: output.md, stdout, stderr, prompt, messages.
Add a new tab called "task.md" that fetches and displays the TASK.md content.

The tab should call:
`GET /api/projects/{p}/tasks/{t}/file?name=TASK.md`

This is a task-scoped read (not run-scoped), so it stays the same regardless
of which run is selected within a task.

The tab should be added BEFORE "output.md" as the first tab (it's the task description).
If the file is not found (404), show a message "No TASK.md found".

Look at how other tabs are rendered (look for the tab rendering logic in app.js)
and follow the same pattern.

## Technical notes

- Read the current `handlers_projects.go` to understand the path parsing logic
- Read `web/src/app.js` to understand how tabs work (look for tab switching, SSE, etc.)
- Use `go test ./internal/api/...` to verify backend tests pass
- Use `go build ./...` to verify compilation

## Quality gates (MUST pass before writing DONE file)

1. `go build ./...` must pass
2. `go test ./internal/api/...` must pass
3. `go test -race ./internal/api/...` must pass
4. The new endpoint must return 200 with TASK.md content when the file exists
5. The new endpoint must return 404 when TASK.md is missing

## Completion

When done, write a DONE file to the TASK_FOLDER directory.
Commit all changes with message: `feat(api,ui): add task.md viewer endpoint and web UI tab`
