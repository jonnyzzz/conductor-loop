# Task: Implement Task Deletion Endpoint and CLI Command

## Context

This is a Go-based multi-agent orchestration framework. The project has a REST API
for managing projects, tasks, and runs. Currently:
- `DELETE /api/projects/{p}/tasks/{t}/runs/{r}` - deletes a single completed run (Session #30)
- `run-agent gc` - cleans up old runs in bulk

There is NO way to delete an individual TASK (with all its runs) through the API or UI.
This feature would be valuable because:
1. Users can remove test/scratch tasks without running gc
2. The UI can surface task cleanup without CLI access
3. Complements the existing run-level delete button

## Your Task

### 1. Implement `DELETE /api/projects/{p}/tasks/{t}` REST endpoint

Read the existing handler code:
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`
- Look at `handleDeleteRun` for the pattern to follow
- Look at how routes are registered in the router

The endpoint should:
- Return 409 Conflict if the task has any currently RUNNING runs
- Return 404 if the task does not exist
- Return 204 No Content on success after removing the task directory
- The task directory is at: `<root>/<projectID>/<taskID>/`
- Before deleting, scan for running runs (check run-info.yaml files for status=running)

Add the route in the router (look at how other routes are registered).

### 2. Add test for the new endpoint

Add tests in the test file for the API handlers. Follow the pattern in:
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/` (look for existing test files)
- Use the same httptest pattern as existing tests

Test cases:
- Success: task exists with only completed runs → 204
- Conflict: task has a running run → 409
- Not found: task does not exist → 404

### 3. Add `run-agent task delete` subcommand

Read:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go` (where `task` subcommands are defined)
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/stop.go` (similar pattern to follow)

Add a new `delete` subcommand under `task`:
```
run-agent task delete --project <p> --task <t> [--root <dir>] [--force]
```

- `--project` and `--task` are required
- `--root` defaults to `./runs` or `$RUNS_DIR`
- Default behavior: refuse to delete if any run is currently running (show error)
- `--force` flag: skip the running check and delete anyway

Implementation:
- Scan the task directory for running runs (check run-info.yaml status fields)
- Without `--force`: return error if any runs are running
- Delete the task directory using `os.RemoveAll`
- Print confirmation: "Deleted task: <task-id>"

Add a `delete_test.go` (or similar) with tests:
- Missing project: returns error
- Missing task: returns error
- Task not found: returns error
- Task with running run (no --force): returns error
- Task with only completed runs: succeeds, directory removed
- Task with running run + --force: succeeds despite running run

### 4. Add UI Delete Task button (optional, if time permits)

If implementing the frontend part:
- Read `/Users/jonnyzzz/Work/conductor-loop/frontend/src/`
- Look at how the "Delete run" button was implemented in `RunDetail.tsx`
- Add a "Delete task" button in the task list item or task header
- Call `DELETE /api/projects/{p}/tasks/{t}`
- On success: refresh the task list and navigate back to the project view
- On 409 Conflict: show a message "Cannot delete task with running runs"

Note: The frontend uses React+TypeScript. After editing, run:
```bash
cd /Users/jonnyzzz/Work/conductor-loop/frontend
npm run build
```
to rebuild the frontend.

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop
go build ./...              # must pass
go test ./...               # must pass
go test -race ./internal/... ./cmd/...  # no races
```

## Output

Write a summary to the output.md file in your run directory.

Commit your changes:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
git add .
git commit -m "feat(api,cli): add task deletion endpoint and run-agent task delete command"
```

Follow AGENTS.md commit format.

## Important Notes

- Task directories are at: `<root>/<projectID>/<taskID>/`
- Run directories are nested inside: `<taskDir>/runs/<runID>/`
- Run status is in: `<runDir>/run-info.yaml` (field: `status`)
- Never delete a running task without --force
- Use `os.RemoveAll` for directory deletion
- The root dir can be found from `--root` flag or `RUNS_DIR` env var
