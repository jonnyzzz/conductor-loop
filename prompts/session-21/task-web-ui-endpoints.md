# Task: Update Web UI to Use Project-Scoped Message Endpoints

## Context

The web UI currently uses old-style `/api/v1/messages/stream?project_id=...` endpoints.
New cleaner project-scoped endpoints now exist. Update the web UI to use them.

Read these files first:
1. /Users/jonnyzzz/Work/conductor-loop/AGENTS.md (code style, commit format)
2. /Users/jonnyzzz/Work/conductor-loop/web/src/app.js (current web UI)
3. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_messages.go (new endpoints)
4. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go (routing)

## Available Project-Scoped Message Endpoints

These endpoints are already implemented and working:

- `GET /api/projects/{p}/messages` — list recent messages (JSON)
- `GET /api/projects/{p}/messages/stream` — SSE stream of project messages
- `POST /api/projects/{p}/messages` — post a message to project bus
- `GET /api/projects/{p}/tasks/{t}/messages` — list task messages
- `GET /api/projects/{p}/tasks/{t}/messages/stream` — SSE stream of task messages
- `POST /api/projects/{p}/tasks/{t}/messages` — post message to task bus

## Changes to Make in web/src/app.js

### 1. Project SSE (connectProjectSSE function, ~line 123)

Change from:
```js
const sseUrl = `${API_BASE}/api/v1/messages/stream?project_id=${enc(projectId)}`;
```
To:
```js
const sseUrl = `${API_BASE}/api/projects/${enc(projectId)}/messages/stream`;
```

### 2. Task message SSE (loadTabContent function, ~line 340)

Change from:
```js
const sseUrl = `${API_BASE}/api/v1/messages/stream?project_id=${enc(state.selectedProject)}&task_id=${enc(state.selectedTask)}`;
```
To:
```js
const sseUrl = `${API_BASE}/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}/messages/stream`;
```

### 3. Post message (postMessage function, ~line 616)

Change from:
```js
const resp = await fetch(API_BASE + '/api/v1/messages', {
  method: 'POST',
  ...
  body: JSON.stringify({
    project_id: state.selectedProject,
    task_id: state.selectedTask,
    type,
    body,
  }),
});
```
To use task-scoped endpoint:
```js
const url = state.selectedTask
  ? `${API_BASE}/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}/messages`
  : `${API_BASE}/api/projects/${enc(state.selectedProject)}/messages`;
const resp = await fetch(url, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ type, body }),
});
```

## Important Notes

1. The SSE message payload format is the SAME for both old and new endpoints (both use `content` field).
2. The POST request body for project-scoped endpoint only needs `type` and `body` (not `project_id`/`task_id`).
3. Do NOT break any existing functionality.

## Completion Criteria

- [ ] `connectProjectSSE()` uses `/api/projects/{p}/messages/stream`
- [ ] Task MESSAGES tab uses `/api/projects/{p}/tasks/{t}/messages/stream`
- [ ] `postMessage()` uses project-scoped POST endpoints
- [ ] Build still passes: `go build ./...`
- [ ] Tests pass: `go test ./...`
- [ ] Changes committed: `feat(web): use project-scoped message endpoints`

## Done File

When complete: `echo "done" > "$TASK_FOLDER/DONE"`
