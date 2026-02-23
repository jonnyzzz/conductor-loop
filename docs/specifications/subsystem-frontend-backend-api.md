# Frontend-Backend API Contract Subsystem

## Overview
This document defines the current API contract exposed by the Go backend (`internal/api`).
Source of truth:
- `internal/api/routes.go`
- `internal/api/handlers.go`
- `internal/api/handlers_projects.go`
- `internal/api/handlers_projects_messages.go`
- `internal/api/sse.go`

## Transport and Defaults
- Protocol: HTTP JSON + SSE (`text/event-stream`).
- Server default host: `0.0.0.0`.
- Server default port: `14355`.
- If default port is not explicitly pinned, server may probe up to `14355..14454`.
- UI is served from the same backend under `/` and `/ui/`.

## Top-Level Routes

### Core (`/api/v1` + metrics)
| Path | Methods | Notes |
| --- | --- | --- |
| `/metrics` | GET | Prometheus format. |
| `/api/v1/health` | GET | `{"status":"ok"}`. |
| `/api/v1/version` | GET | `{"version":"<server version>"}`. |
| `/api/v1/status` | GET | Runtime status summary. |
| `/api/v1/admin/self-update` | GET, POST | Self-update status/request. |
| `/api/v1/runs/stream/all` | GET | SSE fan-in for all runs. |
| `/api/v1/tasks` | GET, POST | Task list/create. |
| `/api/v1/tasks/{task_id}` | GET, DELETE | Task fetch/cancel. |
| `/api/v1/runs` | GET | Flat run list. |
| `/api/v1/runs/{run_id}` | GET | Run summary JSON. |
| `/api/v1/runs/{run_id}/info` | GET | Raw `run-info.yaml`. |
| `/api/v1/runs/{run_id}/stop` | POST | Stop run. |
| `/api/v1/runs/{run_id}/stream` | GET | SSE for one run logs/status. |
| `/api/v1/messages` | GET, POST | Message list/post by query/body scope. |
| `/api/v1/messages/stream` | GET | SSE message stream. |

### Project-centric (`/api/projects`)
| Path | Methods | Notes |
| --- | --- | --- |
| `/api/projects` | GET, POST | Project list/create. |
| `/api/projects/home-dirs` | GET | Recently used source dirs. |
| `/api/projects/{project_id}` | GET, DELETE | Project detail/delete. |
| `/api/projects/{project_id}/stats` | GET | Project aggregate stats. |
| `/api/projects/{project_id}/runs/flat` | GET | Flattened run graph view. |
| `/api/projects/{project_id}/tasks` | GET | Task list for project. |
| `/api/projects/{project_id}/tasks/{task_id}` | GET, DELETE | Task detail/delete. |
| `/api/projects/{project_id}/tasks/{task_id}/resume` | POST | Remove `DONE` marker. |
| `/api/projects/{project_id}/tasks/{task_id}/file` | GET | Task file endpoint (`TASK.md` only). |
| `/api/projects/{project_id}/tasks/{task_id}/runs` | GET | Paginated runs for task. |
| `/api/projects/{project_id}/tasks/{task_id}/runs/stream` | GET | SSE fan-in for all task runs. |
| `/api/projects/{project_id}/tasks/{task_id}/runs/{run_id}` | GET, DELETE | Run detail/delete. |
| `/api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/stop` | POST | Stop run. |
| `/api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/file` | GET | Run file content endpoint. |
| `/api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/stream` | GET | SSE file-tail stream. |
| `/api/projects/{project_id}/messages` | GET, POST | Project bus list/post. |
| `/api/projects/{project_id}/messages/stream` | GET | Project bus SSE. |
| `/api/projects/{project_id}/tasks/{task_id}/messages` | GET, POST | Task bus list/post. |
| `/api/projects/{project_id}/tasks/{task_id}/messages/stream` | GET | Task bus SSE. |
| `/api/projects/{project_id}/gc` | POST | Garbage collection for old runs. |

## Request and Response Contracts

### `GET /api/v1/status`
Response shape:
```json
{
  "active_runs_count": 1,
  "uptime_seconds": 123.45,
  "configured_agents": ["claude", "codex"],
  "version": "dev",
  "running_tasks": [
    {
      "project_id": "conductor-loop",
      "task_id": "task-...",
      "run_id": "20260223-...",
      "agent": "codex",
      "started": "2026-02-23T20:00:00Z"
    }
  ]
}
```

### `POST /api/v1/tasks`
Request shape:
```json
{
  "project_id": "conductor-loop",
  "task_id": "task-...",
  "agent_type": "codex",
  "prompt": "...",
  "config": {"KEY": "VALUE"},
  "project_root": "/abs/path",
  "attach_mode": "create",
  "depends_on": ["task-..."],
  "thread_parent": {
    "project_id": "...",
    "task_id": "...",
    "run_id": "...",
    "message_id": "..."
  },
  "thread_message_type": "USER_REQUEST"
}
```
Response (`201`):
```json
{
  "project_id": "conductor-loop",
  "task_id": "task-...",
  "run_id": "20260223-...",
  "status": "started",
  "queue_position": 0,
  "depends_on": ["task-..."]
}
```

### `GET /api/v1/tasks`
Response:
```json
{
  "tasks": [
    {
      "project_id": "conductor-loop",
      "task_id": "task-...",
      "status": "running",
      "queue_position": 0,
      "last_activity": "2026-02-23T20:00:00Z",
      "depends_on": [],
      "blocked_by": []
    }
  ]
}
```

### `GET /api/v1/tasks/{task_id}`
Supports optional `project_id` query filter.
Response includes embedded runs (`run_id/project_id/task_id/status/...`).

### `DELETE /api/v1/tasks/{task_id}`
Marks task done and attempts to stop active runs.
Response (`202`):
```json
{"stopped_runs": 1}
```

### `GET /api/v1/runs`
Response:
```json
{
  "runs": [
    {
      "run_id": "20260223-...",
      "project_id": "conductor-loop",
      "task_id": "task-...",
      "status": "completed",
      "process_ownership": "managed",
      "start_time": "2026-02-23T20:00:00Z",
      "end_time": "2026-02-23T20:01:00Z",
      "exit_code": 0,
      "agent_version": "...",
      "error_summary": ""
    }
  ]
}
```

### `GET /api/v1/runs/{run_id}`
Returns the same `RunResponse` shape as list items.

### `GET /api/v1/runs/{run_id}/info`
- Content-Type: `application/x-yaml`.
- Body: raw `run-info.yaml`.

### `POST /api/v1/runs/{run_id}/stop`
Response (`202`):
```json
{"status":"stopping"}
```

### `GET /api/v1/messages`
Query:
- `project_id` (required)
- `task_id` (optional)
- `after` (optional msg_id cursor)

Response:
```json
{
  "messages": [
    {
      "msg_id": "MSG-...",
      "timestamp": "2026-02-23T20:00:00Z",
      "type": "FACT",
      "project_id": "conductor-loop",
      "task_id": "task-...",
      "run_id": "20260223-...",
      "issue_id": "MSG-...",
      "parents": [{"msg_id": "MSG-...", "kind": "depends_on"}],
      "links": [{"url": "https://example.invalid", "label": "ref", "kind": "reference"}],
      "meta": {"source": "runner"},
      "body": "..."
    }
  ]
}
```

### `POST /api/v1/messages`
Request:
```json
{
  "project_id": "conductor-loop",
  "task_id": "task-...",
  "run_id": "20260223-...",
  "type": "USER",
  "body": "..."
}
```
Response (`201`):
```json
{"msg_id":"MSG-...","timestamp":"2026-02-23T20:00:00Z"}
```
If `type` is omitted, default is `USER`.

### Project/task message list and post endpoints
- `GET /api/projects/{project_id}/messages`
- `GET /api/projects/{project_id}/tasks/{task_id}/messages`
- `POST /api/projects/{project_id}/messages`
- `POST /api/projects/{project_id}/tasks/{task_id}/messages`

List query:
- `since` (msg_id)
- `limit` (max `5000`)

POST body:
```json
{"type":"USER","body":"..."}
```
Response (`201`) matches `PostMessageResponse` (`msg_id`, `timestamp`).

### Project and task metadata endpoints
`GET /api/projects`:
```json
{
  "projects": [
    {
      "id": "conductor-loop",
      "last_activity": "2026-02-23T20:00:00Z",
      "task_count": 3,
      "project_root": "/abs/path"
    }
  ]
}
```

`POST /api/projects` request:
```json
{"project_id":"conductor-loop","project_root":"/abs/path"}
```
Response (`201`) is one `projectSummary` object.

`GET /api/projects/{project_id}` response:
```json
{
  "id": "conductor-loop",
  "last_activity": "2026-02-23T20:00:00Z",
  "task_count": 3,
  "project_root": "/abs/path"
}
```

`GET /api/projects/{project_id}/tasks` returns paginated envelope:
```json
{
  "items": [
    {
      "id": "task-...",
      "project_id": "conductor-loop",
      "status": "running",
      "last_activity": "2026-02-23T20:00:00Z",
      "run_count": 4,
      "run_counts": {"running": 1, "completed": 3},
      "depends_on": [],
      "blocked_by": [],
      "thread_parent": null,
      "done": false,
      "last_run_status": "running",
      "last_run_exit_code": 0,
      "last_run_output_size": 1234,
      "queue_position": 0
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0,
  "has_more": false
}
```

`GET /api/projects/{project_id}/tasks/{task_id}` returns `projectTask` shape with `runs` array of `projectRun` entries.

`GET /api/projects/{project_id}/tasks/{task_id}/runs` returns paginated `projectRun` items.

`GET /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}` returns one `projectRun`.

### File access endpoints
Task file:
- `GET /api/projects/{project_id}/tasks/{task_id}/file?name=TASK.md`
- Only `TASK.md` is supported.

Run file:
- `GET /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/file?name=stdout|stderr|prompt|output.md`
- Default `name=stdout` when omitted.

File response shape:
```json
{
  "name": "stdout",
  "content": "...",
  "modified": "2026-02-23T20:00:00Z",
  "size_bytes": 1234
}
```

### Destructive project/task/run endpoints
- `DELETE /api/projects/{project_id}` (supports `?force=true`)
- `DELETE /api/projects/{project_id}/tasks/{task_id}`
- `DELETE /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}`
- `POST /api/projects/{project_id}/gc?older_than=168h&dry_run=true|false&keep_failed=true|false`

`GET /api/projects/{project_id}/stats` and `GET /api/projects/home-dirs` provide aggregate stats and home-dir hints.

`GET /api/projects/{project_id}/runs/flat` supports:
- `active_only`
- `selected_task_id`
- `selected_task_limit` (clamped)

### Self-update endpoint
`GET /api/v1/admin/self-update` response shape:
```json
{
  "state": "idle|deferred|applying|failed",
  "binary_path": "/path/to/candidate",
  "requested_at": "...",
  "started_at": "...",
  "finished_at": "...",
  "active_runs_at_request": 0,
  "active_runs_now": 0,
  "active_runs_error": "",
  "last_error": "",
  "last_note": ""
}
```

`POST /api/v1/admin/self-update` request:
```json
{"binary_path":"/abs/path/to/new/binary"}
```
Successful request returns `202` with the same status object.

## SSE Contracts

### Message bus SSE
Endpoints:
- `/api/v1/messages/stream`
- `/api/projects/{project_id}/messages/stream`
- `/api/projects/{project_id}/tasks/{task_id}/messages/stream`

Events:
- `message` with `id: <msg_id>` and payload fields:
  - `msg_id`, `timestamp`, `type`, `project_id`, `task_id`, `run_id`, `issue_id`, `parents` (msg_id list), `meta`, `body`
- `heartbeat` with `{}`

Resumption:
- Use `Last-Event-ID: <msg_id>`.

### Run log SSE (`StreamManager`)
Endpoints:
- `/api/v1/runs/{run_id}/stream`
- `/api/v1/runs/stream/all`
- `/api/projects/{project_id}/tasks/{task_id}/runs/stream`

Events:
- `log` payload:
  - `run_id`, optional `project_id`, optional `task_id`, `stream`, `line`, `timestamp`
- `status` payload:
  - `run_id`, optional `project_id`, optional `task_id`, `status`, `exit_code`
- `heartbeat` payload `{}`

Cursor:
- SSE ID for `log` events uses `s=<stdout>;e=<stderr>`.
- Clients may pass this value in `Last-Event-ID`.

### Run file-tail SSE
Endpoint:
- `/api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/stream?name=stdout|stderr|prompt|output.md`

Behavior:
- Emits raw `data:` lines as file grows.
- Emits `event: done` once run is finished and file tail is exhausted.
- Emits `event: error` on read failures.

## Errors and Status Codes
General error envelope for wrapped handlers:
```json
{
  "error": {
    "code": "BAD_REQUEST|NOT_FOUND|CONFLICT|METHOD_NOT_ALLOWED|FORBIDDEN|INTERNAL",
    "message": "...",
    "details": {}
  }
}
```

Common statuses:
- `200 OK`
- `201 Created`
- `202 Accepted`
- `204 No Content`
- `400 Bad Request`
- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found`
- `405 Method Not Allowed`
- `409 Conflict`
- `429 Too Many Requests` (per-run SSE max clients)
- `500 Internal Server Error`

Auth middleware unauthorized response (special case):
```json
{"error":"unauthorized","message":"valid API key required"}
```

## Security and Validation
- Identifier validation rejects separators and `..` (including URL-decoded forms).
- File and bus paths are confined to configured root via `requirePathWithinRoot`.
- Browser/UI-origin destructive operations are rejected with `403` (`rejectUIDestructiveAction`).
- API key auth supports:
  - `Authorization: Bearer <key>`
  - `X-API-Key: <key>`
- Auth-exempt paths:
  - `/api/v1/health`
  - `/api/v1/version`
  - `/metrics`
  - `/ui/*`

## Drift Corrections Applied
- Canonical default port is `14355` (not `8080`).
- Project routes are under `/api/projects` (not `/api/v1/projects`).
- Message bus endpoints use `/messages`, not `/bus`.
- Message stream includes full payload and SSE `id`.
