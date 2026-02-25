# Task: Add Per-Task Log Stream Endpoint + Wire React LogViewer

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

This is a Go-based multi-agent orchestration framework. The project has:
- A Go backend serving REST API at /api/...
- A React frontend in frontend/ (built to frontend/dist/)
- The React LogViewer component is fully built but BROKEN because `logStreamUrl = undefined`

## Current State

The React frontend (frontend/src/App.tsx) has:
```typescript
const logStreamUrl = undefined  // task-level log streaming not yet implemented
```

This makes the entire "Live logs" panel (LogViewer component) show "No log lines yet" and never update.
The LogViewer expects a URL for SSE log streaming with events: `log`, `run_start`, `run_end`.

The backend has `GET /api/v1/runs/stream/all` which streams ALL runs. But there is no per-task endpoint.

The backend SSE infrastructure is in `internal/api/sse.go`:
- `StreamManager.SubscribeRun(runID, cursor)` - subscribe to a single run's logs
- `streamAllRuns()` - fans in all runs (example to follow)

The project-scoped API router is in `internal/api/handlers_projects.go`:
- `handleProjectsRouter` dispatches on URL path segments
- `handleProjectTask` handles `/api/projects/{p}/tasks/{t}/...` sub-paths

## What To Implement

### 1. Backend: Add per-task log stream endpoint

Add support for `GET /api/projects/{p}/tasks/{t}/runs/stream` in handlers_projects.go.

The handler should:
1. Parse projectID and taskID from the path
2. List all runs for that project+task
3. Subscribe to each run's stream using StreamManager.SubscribeRun()
4. Fan them all together (like streamAllRuns does with fanIn)
5. Also start a RunDiscovery goroutine to pick up NEW runs started for this task during streaming
6. Stream the fan-in events to the SSE client

Looking at `handleProjectTask()`, the path parsing works like:
- parts[0]=projectID, parts[1]="tasks", parts[2]=taskID, parts[3]=...

So add a case for `parts[3] == "runs" && parts[4] == "stream"` (when len(parts)==5).

The implementation should closely follow `streamAllRuns()` but:
- Filter runs to only those matching projectID AND taskID
- Use `allRunInfos()` to get runs and filter appropriately

For filtering: the `RunInfo` structs from `s.allRunInfos()` have `ProjectID` and `TaskID` fields.

The `listRunIDs()` helper used in `streamAllRuns()` returns all run IDs. For task-scoped streaming,
we need to find run IDs matching the project+task. Look at how `handleProjectTask()` currently
finds runs to understand the data model.

**Important**: The `RunDiscovery` discovery goroutine in `streamAllRuns()` discovers all NEW runs.
For task-level streaming, we need to discover new runs for the specific project+task only.
The `RunDiscovery` type just polls for any new runIDs. Filter them using `allRunInfos()`.

### 2. React: Wire up logStreamUrl

Edit `frontend/src/App.tsx`:
- Change `const logStreamUrl = undefined  // task-level log streaming not yet implemented`
- To: `const logStreamUrl = effectiveProjectId && effectiveTaskId ? `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/runs/stream` : undefined`

### 3. Rebuild React frontend

After making changes to the React source:
```bash
cd /Users/jonnyzzz/Work/conductor-loop/frontend
npm run build
```

This updates frontend/dist/ which is served by the conductor.

### 4. Add tests

Add at least 2 unit tests in `internal/api/` for the new endpoint:
- Test that the endpoint returns 405 for non-GET methods
- Test that the endpoint returns 404 for unknown projects/tasks

Look at existing tests like `TestHandleAllRunsStreamMethodNotAllowed` for patterns.

## Files to Modify

- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` — add route + handler
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/App.tsx` — wire logStreamUrl
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go` — add tests
- Run `cd /Users/jonnyzzz/Work/conductor-loop/frontend && npm run build` after React changes

## Quality Gates

Before finishing:
1. `go build ./...` must pass
2. `go test ./internal/... ./cmd/...` must pass (all tests green)
3. `go test -race ./internal/... ./cmd/...` must pass (no data races)
4. React app rebuilt: `frontend/dist/index.html` must be fresh

## Completion

Create a `DONE` file in `$JRUN_TASK_FOLDER` when complete. Also write a brief summary to `$JRUN_RUN_FOLDER/output.md`.

Commit your changes with:
```
feat(api): add per-task log stream SSE endpoint and wire React LogViewer
```

Follow the project's commit convention from AGENTS.md.
