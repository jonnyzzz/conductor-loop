# Task: Implement conductor project delete command

## Context

You are implementing a new feature for the Conductor Loop project at /Users/jonnyzzz/Work/conductor-loop.

The project is a Go-based multi-agent orchestration framework. The conductor CLI has commands to manage projects and tasks via a REST API server.

## Current State

The conductor CLI has `conductor project list`, `conductor project stats`, and `conductor project gc` commands in `cmd/conductor/project.go`. There is also `conductor task delete <task-id>` that deletes individual tasks.

What's MISSING: there is no way to delete an ENTIRE project (all tasks + all runs). You can only delete tasks one by one.

## Your Task

Implement `conductor project delete <project-id>` that deletes an entire project via the API+CLI.

### Part 1: API Endpoint

Add `DELETE /api/projects/{id}` to the server in `internal/api/handlers_projects.go`.

**Behavior:**
1. Check if any tasks have running runs. If yes, return `409 Conflict` with message `{"error": "project has running tasks; stop them first or use --force"}`.
2. If no running tasks (or force=true via `?force=true` query param), delete all task directories under the project directory and the project directory itself.
3. Return `200 OK` with JSON:
   ```json
   {"project_id": "...", "deleted_tasks": N, "freed_bytes": N}
   ```
4. If project not found, return `404 Not Found`.

**How to identify running runs:** A run is "running" if its `run-info.yaml` has `status: running` (or no end_time). Use the existing `listTaskRunInfos()` helper to get runs.

**How to compute freed_bytes:** Use `dirSize(projectDir)` — walk the directory and sum file sizes before deletion. You can implement a simple `dirSize(path string) int64` helper that uses `filepath.WalkDir`.

**Wire it up:** In `internal/api/handlers_projects.go`'s `handleProjectsRouter` function, add a case for `DELETE /api/projects/{id}` (method=DELETE, parts length=1).

### Part 2: CLI Command

Add `conductor project delete <project-id>` to `cmd/conductor/project.go`.

**Usage:**
```
conductor project delete <project-id> [flags]

Flags:
  --force         stop running tasks and delete anyway
  --server URL    conductor server URL (default: http://localhost:8080)
  --json          output response as JSON
```

**Response display (text mode):**
```
Project conductor-loop deleted (N tasks, X.XX MB freed).
```

**If 409 (running tasks):**
```
Error: project has running tasks; stop them first or use --force
```

Register the new command in `newProjectCmd()`.

### Part 3: Tests

Add tests in `internal/api/handlers_projects_test.go` for the DELETE endpoint:
- Test deleting an empty project → 200 OK, deleted_tasks=0
- Test deleting a project with tasks → 200 OK, deleted_tasks=N
- Test deleting a non-existent project → 404
- Test deleting a project with running tasks → 409
- Test force=true deletes project even with running tasks → 200

Add tests in `cmd/conductor/commands_test.go` for the CLI command:
- Test successful deletion (mock server returns 200)
- Test 409 error message
- Test --json flag

### Important Implementation Notes

1. Read the existing `handleProjectsRouter` in `internal/api/handlers_projects.go` to understand the routing pattern — add DELETE handling alongside existing GET/POST cases.

2. Use the existing `splitPath` and helper functions. Look at `handleTaskDelete` as a pattern for deletion logic.

3. Use `dirSize` to compute freed bytes BEFORE deletion.

4. To check for running tasks: iterate through task dirs, call `listTaskRunInfos` for each, check if any run has no `EndTime`.

5. For the `--force` flag: if force=true, first stop all running tasks (send SIGTERM to their PIDs) then delete. You can use the existing `stopTaskRuns()` helper to stop running tasks.

6. After deletion, call `os.RemoveAll(projectDir)` to delete the project directory.

## Quality Requirements

1. `go build ./...` must pass
2. `go test -race ./internal/... ./cmd/...` must pass (no new test failures)
3. Follow existing code style (tabwriter for output, error wrapping, etc.)
4. Commit with message: `feat(api+cli): add conductor project delete command`

## How to Work

1. Read relevant files first:
   - `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/project.go`
   - `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go`
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/commands_test.go`

2. Implement changes
3. Build and test
4. Commit

Working directory: /Users/jonnyzzz/Work/conductor-loop
