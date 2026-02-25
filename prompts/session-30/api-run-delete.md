# Task: API - Add DELETE endpoint for runs and task archive

## Context

You are a sub-agent working on the conductor-loop project. This is a Go-based multi-agent
orchestration framework with a REST API served by the conductor binary.

**Working directory**: /Users/jonnyzzz/Work/conductor-loop

## Background

The `run-agent gc` CLI command can delete old runs from disk, but there's no REST API
endpoint to delete individual runs. Users monitoring via the web UI or integrating via
API have no way to remove individual runs without using the CLI.

## What to implement

### 1. DELETE /api/projects/{project}/tasks/{task}/runs/{run}

Add an endpoint to delete a specific run directory and all its contents.

**Behavior:**
- Only allow deleting completed/failed runs (status != "running")
- Return 409 Conflict if run is still running
- Delete the run directory: `<root>/<project>/<task>/runs/<run_id>/`
- Return 204 No Content on success
- Return 404 if run not found

**Implementation location**: `internal/api/handlers_projects.go`

Add handler `handleRunDelete`:
```go
func (s *Server) handleRunDelete(w http.ResponseWriter, r *http.Request) *apiError {
    projectID, taskID, runID := extractProjectTaskRun(r)
    // Use findProjectTaskDir helper (already exists in the file)
    // Verify run exists and is not running (read run-info.yaml)
    // Delete the run directory
    // Return 204
}
```

Add route in `internal/api/routes.go`:
```
mux.Handle("DELETE /api/projects/", s.wrap(s.handleProjectsRouter))
```

Note: The existing `handleProjectsRouter` dispatches based on path segments. You'll need
to add DELETE routing for the runs path pattern. Look at how `handleProjectsRouter` works
and add a case for DELETE method on `/api/projects/{p}/tasks/{t}/runs/{r}`.

### 2. DELETE /api/projects/{project}/tasks/{task}/runs/{run} - Frontend button

Also add a "Delete" button in the RunDetail view (frontend) that calls DELETE on the run.

In `frontend/src/components/RunDetail.tsx`:
- Add a "Delete" button next to the "Stop" button
- Only show for non-running runs
- Show confirmation dialog before deleting
- After deletion, navigate back to task list

In `frontend/src/api/client.ts`:
```typescript
async deleteRun(projectId: string, taskId: string, runId: string): Promise<void> {
  await this.request<void>(
    `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs/${encodeURIComponent(runId)}`,
    { method: 'DELETE' }
  )
}
```

### 3. Tests

Add tests to `internal/api/handlers_projects_test.go` (or appropriate test file):
- Test DELETE returns 204 for completed run
- Test DELETE returns 409 for running run
- Test DELETE returns 404 for non-existent run

## Files to modify

1. `internal/api/handlers_projects.go` - add handleRunDelete
2. `internal/api/routes.go` - add DELETE route
3. `internal/api/handlers_projects_test.go` - add tests
4. `frontend/src/api/client.ts` - add deleteRun method
5. `frontend/src/components/RunDetail.tsx` - add Delete button
6. Rebuild frontend: `cd /Users/jonnyzzz/Work/conductor-loop/frontend && npm run build`

## After making code changes

1. Build Go: `cd /Users/jonnyzzz/Work/conductor-loop && go build ./...`
2. Run tests: `cd /Users/jonnyzzz/Work/conductor-loop && go test ./internal/api/... ./cmd/...`
3. Build frontend: `cd /Users/jonnyzzz/Work/conductor-loop/frontend && npm run build`
4. Run all tests: `cd /Users/jonnyzzz/Work/conductor-loop && go test ./...`

## Quality gates

- `go build ./...` passes
- `go test ./internal/api/...` passes (all new tests pass)
- No TypeScript errors in frontend build
- Existing tests still pass

## Commit format

```
feat(api,ui): add DELETE endpoint for run deletion

- DELETE /api/projects/{p}/tasks/{t}/runs/{r} removes completed/failed runs
- Returns 409 Conflict if run is still running
- Frontend RunDetail component gets Delete button for non-running runs
- 3 new API tests for delete endpoint behavior
```

Write "done" to: the DONE file in JRUN_TASK_FOLDER (env var will be set in your environment)
