# Task: Add Pagination to Task and Run Listing APIs

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

The project serves a REST API from `internal/api/`. Currently, these endpoints return ALL results:
- `GET /api/projects/{projectID}/tasks` — returns all tasks for a project
- `GET /api/projects/{projectID}/tasks/{taskID}/runs` — returns all runs for a task

As tasks and runs accumulate (the project already has 100+ tasks), returning all results without pagination degrades performance and usability.

## Your Task

Add `limit` and `offset` pagination query parameters to both list endpoints.

### Requirements

1. **Query parameters** (both endpoints):
   - `?limit=N` — return at most N items (default: 50, max: 500)
   - `?offset=M` — skip M items (default: 0)

2. **Response envelope** — wrap existing response in pagination metadata:
   ```json
   {
     "items": [...],
     "total": 150,
     "limit": 50,
     "offset": 0,
     "has_more": true
   }
   ```
   For backward compatibility: if no `limit` or `offset` params are specified, still return the envelope (not raw array).

3. **Sort order** — tasks sorted by creation time (newest first), runs sorted by start time (newest first).

4. **Frontend compatibility** — update the React frontend (`frontend/src/`) to handle the new paginated response format. The frontend currently reads the raw arrays directly.

### Implementation Steps

#### Step 1: Read the existing API code

```
/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go
/Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go
/Users/jonnyzzz/Work/conductor-loop/internal/api/api_types.go (if it exists)
```

Run the server locally to see current response format:
```bash
# Check existing task list response structure by reading the handler
```

#### Step 2: Read the frontend code

```
/Users/jonnyzzz/Work/conductor-loop/frontend/src/
```
Focus on components that call the tasks/runs APIs.

#### Step 3: Implement pagination in the API

In `internal/api/handlers_projects.go`:
1. Add a helper `parsePagination(r *http.Request) (limit, offset int)` that parses and validates query params
2. Add a `PaginatedResponse` struct to the project handlers file or a shared types file
3. Update `handleProjectTasks` to apply pagination
4. Update `handleProjectTaskRuns` to apply pagination

#### Step 4: Add tests

In `internal/api/` add test cases for:
- Default pagination (no params → limit=50, offset=0)
- Custom limit and offset
- Limit clamped to max (500)
- Total count correct even when results are paginated

#### Step 5: Update frontend

In `frontend/src/`:
- Update TypeScript types for the paginated response
- Update fetch calls to handle the new response format
- The current behavior (load all) should still work by default (no UI pagination controls needed — just make it compatible)
- Rebuild frontend: `cd frontend && npm run build`

#### Step 6: Build and test

```bash
go build ./...
go test ./internal/api/...
go test -race ./internal/api/...
```

### Quality Requirements

- Follow existing code style in `internal/api/`
- Add tests that cover the new pagination behavior
- Do NOT break existing tests
- Do NOT add frontend pagination UI controls — just make the format compatible
- Create DONE file in TASK_FOLDER when complete

## Done Criteria

- [ ] `GET /api/projects/{p}/tasks?limit=10&offset=0` returns paginated response
- [ ] `GET /api/projects/{p}/tasks/{t}/runs?limit=5&offset=0` returns paginated response
- [ ] Response includes `total`, `limit`, `offset`, `has_more` fields
- [ ] Frontend compiles and is compatible with new response format
- [ ] New tests added and all tests pass
- [ ] `go build ./...` passes
- [ ] DONE file created in TASK_FOLDER
