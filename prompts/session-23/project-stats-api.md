# Task: Add Project Statistics to API (Session #23)

## Context

You are improving the REST API of the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

The project is a Go-based multi-agent orchestration framework. It has a REST API defined in `internal/api/handlers_projects.go` and `internal/api/handlers.go`.

**Current state:**
- `GET /api/projects/{p}/tasks` lists all tasks for a project
- `GET /api/projects/{p}/tasks/{t}/runs` lists all runs for a task
- `GET /api/v1/status` returns server uptime, configured agents, active run count

**What to implement:**
Add a project-level statistics endpoint that gives useful operational metrics.

## Requirements

### 1. New endpoint: `GET /api/projects/{p}/stats`

Returns statistics for a project:
```json
{
  "project_id": "conductor-loop",
  "total_tasks": 49,
  "total_runs": 78,
  "running_runs": 2,
  "completed_runs": 70,
  "failed_runs": 6,
  "crashed_runs": 0,
  "message_bus_files": 12,
  "message_bus_total_bytes": 184320
}
```

**How to compute:**
- Walk `<root>/<project>/` directory
- Count task directories (valid task IDs matching `task-<YYYYMMDD>-<HHMMSS>-<slug>`)
- For each task, count run directories and read `run-info.yaml` for status
- Count `*MESSAGE-BUS.md` files and sum their sizes
- Use `os.ReadDir` for directory walks
- If the project doesn't exist, return 404

### 2. Register the endpoint

In `internal/api/handlers_projects.go`, add the new handler. Register it in `setupRoutes()` or wherever project routes are registered.

Pattern: `GET /api/projects/{p}/stats`

The `{p}` variable is captured via path splitting (look at how existing project handlers work).

### 3. Tests

Add tests in `internal/api/handlers_projects_test.go`:
- Test that stats returns correct counts for a project with known tasks/runs
- Test that a non-existent project returns 404
- At least 3 new tests

## Implementation Steps

1. Read and understand:
   - `internal/api/handlers_projects.go` — existing project handlers
   - `internal/api/handlers_projects_test.go` — test patterns
   - `internal/storage/storage.go` and `internal/storage/runinfo.go` — how to read run info
2. Add the stats struct and handler function
3. Register the route
4. Add tests using `httptest`
5. Run: `go build ./... && go test ./internal/api/...`

## Quality Gates

- `go build ./...` passes
- `go test ./internal/api/...` passes
- `go test -race ./internal/api/...` passes
- All 13 test packages still pass: `go test ./internal/... ./cmd/...`

## When Done

Create a `DONE` file in your task directory (`$JRUN_TASK_FOLDER/DONE`) to signal completion.

Write your output summary to `$JRUN_RUN_FOLDER/output.md`.
