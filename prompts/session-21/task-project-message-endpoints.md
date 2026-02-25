# Task: Add Project-Scoped Message Bus Endpoints

## Context

You are implementing improvements to the Conductor Loop REST API. Read the following files
first to understand the codebase:

1. /Users/jonnyzzz/Work/conductor-loop/AGENTS.md (code style, commit format)
2. /Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go (current routes)
3. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go (project handlers)
4. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go (message handlers)
5. /Users/jonnyzzz/Work/conductor-loop/internal/api/sse.go (SSE streaming - streamMessages func)
6. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go (existing tests)

## Goal

Add project-scoped and task-scoped message bus endpoints to the project API router so users
have a clean, RESTful way to access message buses without query parameters.

## New Endpoints to Add

All under the `/api/projects/` router in `handlers_projects.go`:

### Project-level message endpoints
- `GET /api/projects/{p}/messages` — list recent messages from PROJECT-MESSAGE-BUS.md
  - Query params: `limit` (default 50), `since` (msg_id for pagination)
  - Response: `{"messages": [...]}`
- `GET /api/projects/{p}/messages/stream` — SSE stream of PROJECT-MESSAGE-BUS.md
  - Same format as existing `/api/v1/messages/stream?project_id=...`
  - Supports `Last-Event-ID` header for resumable clients
- `POST /api/projects/{p}/messages` — post a message to PROJECT-MESSAGE-BUS.md
  - Body: `{"type": "USER", "body": "message text"}`
  - Response 201: `{"msg_id": "...", "timestamp": "..."}`

### Task-level message endpoints
- `GET /api/projects/{p}/tasks/{t}/messages` — list recent messages from TASK-MESSAGE-BUS.md
- `GET /api/projects/{p}/tasks/{t}/messages/stream` — SSE stream of TASK-MESSAGE-BUS.md
- `POST /api/projects/{p}/tasks/{t}/messages` — post a message to TASK-MESSAGE-BUS.md

## Implementation Guidelines

1. **Reuse existing logic**: The `streamMessages` func in sse.go already has the SSE logic.
   Create thin wrapper methods for the new routes that call shared logic with the appropriate
   bus path.

2. **Add routes to handleProjectsRouter**: Update the routing in `handleProjectsRouter` to
   dispatch to the new handlers. Current path structure:
   - `/api/projects/{p}` → handleProjectDetail
   - `/api/projects/{p}/tasks` → handleProjectTasks
   - `/api/projects/{p}/tasks/{t}` → handleProjectTask
   - NEW: `/api/projects/{p}/messages` → handleProjectMessages
   - NEW: `/api/projects/{p}/messages/stream` → handleProjectMessagesStream
   - NEW: `/api/projects/{p}/tasks/{t}/messages` → handled in handleProjectTask router
   - NEW: `/api/projects/{p}/tasks/{t}/messages/stream` → same

3. **Test coverage**: Add tests for all new endpoints in handlers_projects_test.go.
   Use the existing test patterns (httptest.NewRecorder, httptest.NewRequest).

4. **Do NOT break existing tests**: Run `go test ./...` before committing.

5. **Follow code style**: No new packages. Keep files under 500 lines. Use existing error
   helpers (apiErrorNotFound, apiErrorBadRequest, apiErrorInternal).

## Routing Fix

The `handleProjectsRouter` currently dispatches to `handleProjectTask` for all paths with
3+ segments after `/api/projects/`. You need to also handle the `messages` sub-path at the
project level (2 segments: `{p}/messages`).

Current `parts` structure after `splitPath(r.URL.Path, "/api/projects/")`:
- 0 segments: not found
- 1 segment: `[{p}]` → handleProjectDetail
- 2+ segments with `parts[1]=="tasks"`: → handleProjectTasks or handleProjectTask
- NEW: 2+ segments with `parts[1]=="messages"`: → handleProjectMessages*

## Bus Path Logic

```go
// Project-level messages
busPath := filepath.Join(s.rootDir, projectID, "PROJECT-MESSAGE-BUS.md")

// Task-level messages
busPath := filepath.Join(s.rootDir, projectID, taskID, "TASK-MESSAGE-BUS.md")
```

Always create the parent directory if it doesn't exist (os.MkdirAll).

## Completion Criteria

- [ ] `go build ./...` passes
- [ ] `go test ./...` — all packages green
- [ ] `go test -race ./internal/... ./cmd/...` — no races
- [ ] New endpoints respond correctly (verified by tests)
- [ ] Changes committed with format: `feat(api): add project-scoped message bus endpoints`

## Done File

When complete, write "DONE" to the file path in the JRUN_TASK_FOLDER environment variable
(use `$JRUN_TASK_FOLDER/DONE` path).

Example: `echo "done" > "$JRUN_TASK_FOLDER/DONE"`
