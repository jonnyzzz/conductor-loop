# Task: Add Explicit "All Jobs Finished" Semantics to Run Status

## Context

- **ID**: `task-20260223-155230-run-status-finish-criteria`
- **Priority**: P0
- **Source**: `docs/dev/todos.md`

Currently, run/task status has no first-class distinction between:
- **running**: at least one run is actively executing
- **queued**: runs pending agent pickup
- **blocked**: all pending runs have unfulfilled dependencies
- **failed**: latest run exited non-zero and task is not DONE
- **all-finished**: all runs have completed and DONE is present

The CLI output (`run-agent list`, `run-agent status`) collapses these into coarse states,
making it hard for automated monitoring to determine if a task is truly finished vs.
temporarily idle.

## Requirements

1. **New status values**: Extend `internal/storage` status enum with:
   - `all_finished` (DONE file present, no active runs)
   - `queued` (runs scheduled but not started)
   - `blocked` (dependencies unmet)
   - `partial_failure` (some runs failed, task still active)

2. **CLI flags**:
   - `run-agent status --status` → returns one of the new canonical status strings
   - `run-agent list --status failed` → filter by status
   - `run-agent list --status all_finished` → show only DONE tasks

3. **API response**: `GET /api/projects/{project}/tasks/{task}` includes `"status"` field
   with the new canonical value.

4. **Web UI**: Status badge in task detail reflects new states with distinct colors:
   - `all_finished` → green
   - `queued` → gray
   - `blocked` → orange
   - `partial_failure` → red/yellow
   - `failed` → red

5. **Tests**: Unit tests for status derivation logic; integration test for CLI flag filters.

## Acceptance Criteria

- `run-agent status --task <id> --root runs --status` outputs one of:
  `running`, `queued`, `blocked`, `all_finished`, `partial_failure`, `failed`, `unknown`
- `run-agent list --status all_finished --root runs` lists only tasks with DONE marker.
- API `/tasks/{id}` response includes `"status": "all_finished"` when DONE.
- `go test ./internal/storage ./internal/api -count=1` passes.
- `go build ./...` passes.

## Verification

```bash
go build -o bin/run-agent ./cmd/run-agent
./bin/run-agent status --root runs --project conductor-loop --task <task-id> --status
./bin/run-agent list --root runs --project conductor-loop --status all_finished
go test ./internal/storage ./internal/api -run 'TestStatus|TestFinished' -count=1
```

## Reference Files

- `internal/storage/task.go` — task status derivation
- `cmd/run-agent/status.go` — status command
- `cmd/run-agent/list.go` — list command with filters
- `internal/api/tasks.go` — task API handler
- `frontend/src/` — task status display components
