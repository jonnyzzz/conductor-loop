# Task: Add API stop endpoint + Web UI stop button

## Context

You are working in the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

This is a Go-based multi-agent orchestration framework. The project already has:
- `run-agent stop` CLI command in cmd/run-agent/stop.go that sends SIGTERM to a running process
- `runner.TerminateProcessGroup()`, `runner.KillProcessGroup()`, `runner.IsProcessAlive()` in internal/runner/
- Project-centric API at /api/projects/{project}/tasks/{task}/runs/{runID}/...
- Web UI at web/src/index.html + web/src/app.js

## Goal

Implement two things:

### 1. API endpoint: POST /api/projects/{p}/tasks/{t}/runs/{r}/stop

Add this endpoint to `internal/api/handlers_projects.go`.

**Behavior:**
- Read `run-info.yaml` for the given run directory
- If status != "running": return 409 Conflict with `{"error": "run is not running", "status": "..."}`
- If status == "running":
  - Call `runner.TerminateProcessGroup(pgid)` (use PGID if > 0, else PID)
  - Return immediately with 202 Accepted: `{"run_id": "...", "message": "SIGTERM sent"}`
  - Do NOT wait for the process to stop — that's async
- If run not found: return 404

**Route:** Add to `/api/projects/{p}/tasks/{t}/runs/{r}/stop` — the router is in the `handleProjectsRouter` function in `handlers_projects.go`. The pattern for routing is already there (see how `/stream` is handled).

**Import needed:** `"github.com/jonnyzzz/conductor-loop/internal/runner"`

### 2. Web UI: Stop button in run detail panel

Modify `web/src/index.html` and `web/src/app.js`.

**Changes to index.html:**
- In the run-detail header div, add a Stop button:
  ```html
  <button id="stop-run-btn" class="btn-stop hidden" onclick="stopCurrentRun()" title="Stop">■ Stop</button>
  ```
  Place it before the close button.

**Changes to app.js:**
- After `loadRunMeta()` completes successfully, show/hide the stop button based on `run.status === 'running'`
- Add `stopCurrentRun()` function:
  - POSTs to `${runPrefix()}/runs/${enc(state.selectedRun)}/stop`
  - On success: shows toast "Stop signal sent", calls loadRunMeta()
  - On error: shows toast with error message

**Changes to styles.css:**
- Add `.btn-stop` style: red/orange button, small, matching existing button styles

## File Locations

- API routes: `/Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go`
- Project handlers: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`
- Project handlers tests: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go`
- Runner package: `/Users/jonnyzzz/Work/conductor-loop/internal/runner/`
- Storage: `/Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go`
- Web UI: `/Users/jonnyzzz/Work/conductor-loop/web/src/`

## How to find the run directory from the API

The Server struct has a `Root` field. The run directory path is:
`<root>/<project>/<task>/runs/<runID>/`

Look at how `handleRunFile` or `handleRunStream` in `handlers_projects.go` constructs the run directory path.

## Tests Required

Add at least 2 tests to `handlers_projects_test.go`:
1. `TestStopRun_Success` — mock a running run-info.yaml, verify 202 returned
2. `TestStopRun_NotRunning` — mock a completed run-info.yaml, verify 409 returned

For testing, you can create a temp run directory with a fake run-info.yaml. Look at existing tests in handlers_projects_test.go for patterns.

**Note:** The test should NOT need a real process running. Just mock the run-info.yaml with `status: running` and `pid: 99999999` (very high PID that doesn't exist). The handler should call TerminateProcessGroup, which will fail with "no such process" but that should be acceptable for the test. Or you can make the handler not return an error if TerminateProcessGroup fails (just log it and return 202 anyway since SIGTERM was best-effort).

## Quality Gates

Before creating DONE file:
1. `go build ./...` — must pass
2. `go test ./internal/api/...` — must pass
3. `go test -race ./internal/api/...` — must pass

## Output

Write a summary to output.md describing what you implemented, what files you changed, and the test results.

After quality gates pass, create a DONE file in $TASK_FOLDER to signal completion.
