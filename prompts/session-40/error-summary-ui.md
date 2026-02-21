# Task: Surface ErrorSummary in Web UI run detail panel

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
This is a Go-based multi-agent orchestration framework with a React frontend.

## Goal

Show the `error_summary` field from run-info.yaml in the web UI's run detail panel
when a run has failed (ISSUE-010 deferred item).

## Background

ISSUE-010: The `ErrorSummary` field is stored in `run-info.yaml` when a run fails
(see `classifyExitCode()` and `tailFile()` in internal/runner/job.go). But this
field is NOT surfaced in the web UI. Users can't see why a run failed without
manually reading run-info.yaml.

The React frontend is in /Users/jonnyzzz/Work/conductor-loop/frontend/src/.
The API responses come from internal/api/handlers.go.

## Requirements

1. **API**: Add `error_summary` to the `RunResponse` struct in internal/api/handlers.go
   - Check if `RunInfo` already has `ErrorSummary string` field (it should)
   - Map it in `runInfoToResponse()` function
   - Ensure the JSON field is included in GET /api/v1/runs/:id response

2. **React Frontend**: Show error_summary in the run detail view
   - Find where run details are displayed in frontend/src/
   - When `status == "failed"` and `error_summary` is non-empty, show it in a red/warning box
   - Keep it simple: a styled `<div>` with the error text is sufficient
   - After editing frontend, run the build: `cd frontend && npm run build`

3. **Tests**: Add API handler test to verify `error_summary` appears in response
   - Look at internal/api/handlers_test.go for patterns

## Key Files to Read

- /Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go — RunInfo struct with ErrorSummary
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go — RunResponse, runInfoToResponse()
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_test.go — API test patterns
- /Users/jonnyzzz/Work/conductor-loop/frontend/src/ — React components (explore)
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go — classifyExitCode(), tailFile()

## Quality Gates (REQUIRED before writing DONE file)

1. `go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent`
2. `go test -count=1 ./internal/api/` — all tests pass
3. `cd /Users/jonnyzzz/Work/conductor-loop/frontend && npm run build` — builds without errors
4. Verify: GET /api/v1/runs/:id response for a failed run includes `"error_summary"` field

## Output

Write your findings and implementation summary to output.md in your RUN_FOLDER.
Create DONE file in your TASK_FOLDER when complete.

## CRITICAL: Task Folder Environment Variables

Your TASK_FOLDER and RUN_FOLDER are provided as environment variables. Use them:
- Write output to: $RUN_FOLDER/output.md
- Create DONE file at: $TASK_FOLDER/DONE
