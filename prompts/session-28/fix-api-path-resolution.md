# Sub-Agent Task: Fix API Path Resolution Bug (Session #28)

## Role
You are an implementation agent. Your CWD is /Users/jonnyzzz/Work/conductor-loop.

## Problem Description

Multiple API handlers in `internal/api/` use `filepath.Join(s.rootDir, projectID, taskID, ...)`
to construct paths to task files. This breaks when tasks are stored under a `runs/` subdirectory
relative to rootDir.

**Example bug:**
- Server started with `--root /Users/jonnyzzz/Work/conductor-loop`
- Task stored at: `runs/conductor-loop/task-20260220-235739-djv260/TASK-MESSAGE-BUS.md`
- Handler constructs: `conductor-loop/task-20260220-235739-djv260/TASK-MESSAGE-BUS.md`
- Result: file not found

The `allRunInfos()` function correctly walks the entire tree, but direct path construction fails.

## Affected Code

### In `internal/api/handlers_projects.go`:
1. **`serveTaskFile` (line ~341)**: `filepath.Join(s.rootDir, projectID, taskID, "TASK.md")`
2. **`handleProjectStats` (line ~813)**: `filepath.Join(s.rootDir, projectID)` - needs to find all task dirs for this project

### In `internal/api/handlers.go`:
3. **`handleMessages` (line ~252)**: `filepath.Join(s.rootDir, projectID, "PROJECT-MESSAGE-BUS.md")`
4. **`handleMessages` (line ~254)**: `filepath.Join(s.rootDir, projectID, taskID, "TASK-MESSAGE-BUS.md")`
5. **`handlePostMessage` (line ~322,324)**: Same bus path construction
6. **`handleTasks` (line ~398)**: `filepath.Join(s.rootDir, req.ProjectID, req.TaskID)` for task creation

## Storage Layout Reference

The storage layout is: `<root>/<project>/<task>/runs/<run_id>/`

When rootDir is set to the project root (e.g., `/Users/jonnyzzz/Work/conductor-loop`),
runs can be stored at various sub-paths like `runs/<project>/<task>/` or
`conductor-loop/<task>/` etc.

The `allRunInfos()` function handles this by recursively walking rootDir looking for
`run-info.yaml` files - each RunInfo contains the project_id and task_id fields.

## Required Fix

Add a helper function `findTaskDir` to `handlers_projects.go` that:
1. Takes `rootDir string, projectID string, taskID string`
2. Returns the actual task directory path and whether it was found
3. Strategy: walk the rootDir tree looking for directories matching `<projectID>/<taskID>` pattern
   OR use `allRunInfos()` to find a run belonging to this task, then derive the task dir

Alternatively, a simpler but effective approach:
1. Add `findProjectTaskDir(rootDir, projectID, taskID string) (string, bool)` that checks:
   - `filepath.Join(rootDir, projectID, taskID)` (direct)
   - `filepath.Join(rootDir, "runs", projectID, taskID)` (common runs/ subdirectory)
   - Walk rootDir looking for `<anything>/<projectID>/<taskID>` directory

2. Update all callers to use `findProjectTaskDir` instead of direct `filepath.Join`

## Expected Behavior After Fix

The following endpoints should work correctly regardless of where runs are stored:
- `GET /api/projects/{p}/tasks/{t}/file?name=TASK.md`
- `GET /api/projects/{p}/stats`
- `GET /api/v1/messages?project_id=p&task_id=t` (message bus reads)
- `POST /api/v1/messages` (message bus writes)

## Tests Required

Update/add tests in `internal/api/handlers_projects_test.go` or `internal/api/handlers_test.go`:
1. Test that `serveTaskFile` works when task is under `runs/` subdirectory
2. Test `handleProjectStats` correctly counts tasks and runs from `runs/<project>/<task>` structure

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./internal/api/... -race` must pass (no races)
3. `go test ./...` all green

## Commit Message Format

Use format: `fix(api): resolve task directory paths correctly across root structures`

When done, create a DONE file at the task directory to signal completion:
`touch <JRUN_TASK_FOLDER>/DONE`
