# API Reference

Complete REST API reference for Conductor Loop. The API provides endpoints for task management, run monitoring, log streaming, and message bus access.

## API Surfaces

There are two API surfaces:

1. **`/api/v1/...`** — The primary REST API documented in this file, used for task creation, listing, and message bus access.
2. **`/api/projects/...`** — A project-centric API used by the web UI. Provides endpoints like:
   - `GET /api/projects` — list projects
   - `POST /api/projects` — create a project and persist its home/work folder
   - `GET /api/projects/home-dirs` — list known project home/work folders for form suggestions
   - `GET /api/projects/{projectId}/tasks` — list tasks for a project
   - `GET /api/projects/{projectId}/tasks/{taskId}` — task detail with run list
   - `GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}` — run detail
   - `GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/file?name=output.md` — read run file (output.md, stdout, stderr, prompt)
   - `GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/stream?name=output.md` — SSE stream of growing file
   - `POST /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/stop` — stop a running run (202=SIGTERM sent, 409=not running)
   - `GET /api/projects/{projectId}/tasks/{taskId}/file?name=TASK.md` — read TASK.md from task directory
   - `GET /api/projects/{projectId}/tasks/{taskId}/runs/stream` — SSE stream that fans in live output from all runs of a task (used by the React LogViewer)
   - `DELETE /api/projects/{projectId}/tasks/{taskId}/runs/{runId}` — delete a completed or failed run directory (204 No Content on success; 409 Conflict if still running)
   - `DELETE /api/projects/{projectId}/tasks/{taskId}` — delete an entire task directory and all its runs (204 No Content; 409 Conflict if any run is still running; 404 Not Found)
   - `GET /api/projects/{projectId}/stats` — project statistics: task count, run counts by status, and total message bus bytes
   - `GET /api/projects/{projectId}/messages` — list project-level message bus messages; `POST` appends a new message
   - `GET /api/projects/{projectId}/messages/stream` — SSE stream of project-level message bus
   - `GET /api/projects/{projectId}/tasks/{taskId}/messages` — list task-level message bus messages; `POST` appends a new message
   - `GET /api/projects/{projectId}/tasks/{taskId}/messages/stream` — SSE stream of task-level message bus
   - `POST /api/projects/{projectId}/tasks/{taskId}/resume` — remove the task's `DONE` file so the Ralph Loop can restart it (200 OK on success; 404 if task not found; 400 if no DONE file)

## Base URL

```
http://localhost:14355/api/v1
```

Change the host and port in your config.yaml:

```yaml
api:
  host: 0.0.0.0
  port: 14355
```

## Form Submission Audit Log

Web UI form submissions are durably appended to a JSONL audit file:

```
<root>/_audit/form-submissions.jsonl
```

`<root>` is the API server root directory (`--root`, default `~/run-agent`).

Logged endpoints:

- `POST /api/v1/tasks`
- `POST /api/projects`
- `POST /api/projects/{project_id}/messages`
- `POST /api/projects/{project_id}/tasks/{task_id}/messages`
- `POST /api/v1/messages`

Each JSON line includes:

| Field | Description |
|------|-------------|
| `timestamp` | UTC timestamp of the accepted submission |
| `request_id` | Request correlation id (`X-Request-ID` if provided; generated otherwise) |
| `correlation_id` | Same value as `request_id` for cross-log joins |
| `method` | HTTP method |
| `path` | Request path |
| `endpoint` | Route pattern label |
| `remote_addr` | Client remote address when available |
| `project_id` | Project identifier (if applicable) |
| `task_id` | Task identifier (if applicable) |
| `run_id` | Allocated run id (task creation only) |
| `message_id` | Appended message id (message posting only) |
| `payload` | Sanitized request body |

Redaction and safety rules:

- Fields with sensitive key names (`token`, `secret`, `password`, `api_key`, `authorization`, etc.) are replaced with `[REDACTED]`.
- Token-like substrings in free-text fields (for example bearer tokens and JWT-like values) are masked.
- Large text fields are truncated to keep log lines bounded.
- Audit write failures are logged as warnings only; primary API request handling continues.

## Authentication

By default the conductor API is unauthenticated. To enable API key authentication, set an API key using one of the methods below. When a key is configured, all requests to protected endpoints must supply it.

### Enabling authentication

**Via config file (`config.yaml`):**
```yaml
api:
  auth_enabled: true
  api_key: "your-secret-key"
```

**Via environment variable (overrides config; also enables auth):**
```bash
export CONDUCTOR_API_KEY="your-secret-key"
```

**Via CLI flag (overrides config):**
```bash
./conductor --api-key "your-secret-key"
```

If `auth_enabled: true` is set without an `api_key`, a warning is logged and authentication is disabled.

### Sending the API key

Include the key in one of these headers on every request:

```bash
# Authorization: Bearer
curl -H "Authorization: Bearer your-secret-key" http://localhost:14355/api/v1/tasks

# X-API-Key
curl -H "X-API-Key: your-secret-key" http://localhost:14355/api/v1/tasks
```

### Exempt paths

The following paths never require authentication:

| Path | Description |
|------|-------------|
| `/api/v1/health` | Health check |
| `/api/v1/version` | Version info |
| `/metrics` | Prometheus metrics |
| `/ui/` | Web UI static files |

### Unauthorized response

Requests without a valid key receive:

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="conductor"
Content-Type: application/json

{"error":"unauthorized","message":"valid API key required"}
```

For production deployments without API key auth, consider:
- Running behind a reverse proxy with authentication (nginx, Caddy)
- Using network isolation (VPN, private network)

## Response Format

All responses are JSON with appropriate HTTP status codes.

**Success Response:**
```json
{
  "field": "value"
}
```

**Error Response:**
```json
{
  "error": "error message"
}
```

## Endpoints

### Metrics

#### GET /metrics

Returns server metrics in Prometheus text format (content-type: `text/plain; version=0.0.4`).
Suitable for scraping by Prometheus, Grafana, or any compatible monitoring tool.

**Request:**
```bash
curl http://localhost:14355/metrics
```

**Response:** `200 OK` (`text/plain; version=0.0.4`)
```
# HELP conductor_uptime_seconds Server uptime in seconds
# TYPE conductor_uptime_seconds gauge
conductor_uptime_seconds 42.5

# HELP conductor_active_runs_total Currently active (running) agent runs
# TYPE conductor_active_runs_total gauge
conductor_active_runs_total 3

# HELP conductor_completed_runs_total Total completed agent runs since startup
# TYPE conductor_completed_runs_total counter
conductor_completed_runs_total 47

# HELP conductor_failed_runs_total Total failed agent runs since startup
# TYPE conductor_failed_runs_total counter
conductor_failed_runs_total 2

# HELP conductor_messagebus_appends_total Total message bus append operations
# TYPE conductor_messagebus_appends_total counter
conductor_messagebus_appends_total 1234

# HELP conductor_api_requests_total Total API requests by method and status
# TYPE conductor_api_requests_total counter
conductor_api_requests_total{method="GET",status="200"} 100
conductor_api_requests_total{method="POST",status="201"} 5
```

**Metrics exposed:**

| Metric | Type | Description |
|---|---|---|
| `conductor_uptime_seconds` | gauge | Server uptime in seconds |
| `conductor_active_runs_total` | gauge | Currently active (running) agent runs |
| `conductor_completed_runs_total` | counter | Total completed agent runs since startup |
| `conductor_failed_runs_total` | counter | Total failed agent runs since startup |
| `conductor_messagebus_appends_total` | counter | Total message bus append operations |
| `conductor_api_requests_total` | counter | Total API requests, labeled by `method` and `status` |

**Notes:**
- No external dependencies: implemented using Go's `sync/atomic` and Prometheus text format manually.
- The `/metrics` endpoint itself is not counted in `conductor_api_requests_total` (it bypasses the logging middleware).

---

### Health and Version

#### GET /api/v1/health

Check if the server is running.

**Request:**
```bash
curl http://localhost:14355/api/v1/health
```

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

#### GET /api/v1/version

Get the server version.

**Request:**
```bash
curl http://localhost:14355/api/v1/version
```

**Response:** `200 OK`
```json
{
  "version": "dev"
}
```

#### GET /api/v1/admin/self-update

Get safe self-update state.

**Request:**
```bash
curl http://localhost:14355/api/v1/admin/self-update
```

**Response:** `200 OK`
```json
{
  "state": "idle",
  "active_runs_now": 0
}
```

`state` values:
- `idle`: no update is pending.
- `deferred`: update request accepted and waiting for active root runs to finish (new root-run starts are blocked).
- `applying`: binary replacement and handoff in progress.
- `failed`: update attempt failed (see `last_error`).

#### POST /api/v1/admin/self-update

Request a safe self-update to a candidate `run-agent` binary.

**Request Body:**
```json
{
  "binary_path": "/absolute/path/to/run-agent"
}
```

**Request:**
```bash
curl -X POST http://localhost:14355/api/v1/admin/self-update \
  -H "Content-Type: application/json" \
  -d '{"binary_path":"/tmp/run-agent-new"}'
```

**Response:** `202 Accepted`
```json
{
  "state": "deferred",
  "binary_path": "/tmp/run-agent-new",
  "active_runs_at_request": 2,
  "active_runs_now": 2
}
```

Safety behavior:
- If root runs are active, the update is deferred automatically.
- While update state is `deferred` or `applying`, `POST /api/v1/tasks` returns `409 Conflict`.
- Handoff starts only when active root runs drop to zero.
- Update admission and root-run launch admission use a shared gate so new root starts are not admitted after drain begins.
- On failure, rollback is attempted, state changes to `failed`, and queued planner launches deferred by drain mode are resumed automatically.

---

### Tasks

#### POST /api/v1/tasks

Create a new task.

**Request Body:**
```json
{
  "project_id": "my-project",
  "task_id": "task-001",
  "agent_type": "codex",
  "prompt": "Write a hello world script",
  "attach_mode": "create",
  "config": {
    "key": "value"
  },
  "thread_parent": {
    "project_id": "my-project",
    "task_id": "task-20260222-160000-parent",
    "run_id": "20260222-1601000000-12345-1",
    "message_id": "MSG-20260222-160200-000000000-PID12345-0001"
  },
  "thread_message_type": "USER_REQUEST"
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `task_id` | string | Yes | Task identifier |
| `agent_type` | string | Yes | Agent to use (codex, claude, etc.) |
| `prompt` | string | Yes | Task prompt/instructions |
| `attach_mode` | string | No | `create`, `attach`, or `resume` |
| `config` | object | No | Additional configuration |
| `depends_on` | string[] | No | Task dependencies |
| `thread_parent` | object | No | Parent message reference for threaded answer workflow |
| `thread_parent.project_id` | string | Yes* | Parent project id (*required when `thread_parent` is set*) |
| `thread_parent.task_id` | string | Yes* | Parent task id (*required when `thread_parent` is set*) |
| `thread_parent.run_id` | string | Yes* | Parent run id (*required when `thread_parent` is set*) |
| `thread_parent.message_id` | string | Yes* | Parent message id (*required when `thread_parent` is set*) |
| `thread_message_type` | string | No | For threaded flow, must be exactly `USER_REQUEST` |

**Identifier Rules:**
- Must contain only alphanumeric, dash, underscore
- No spaces allowed
- Case-sensitive

**Request:**
```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-001",
    "agent_type": "codex",
    "prompt": "Write a Python script that prints hello world"
  }'
```

**Response:** `201 Created`
```json
{
  "project_id": "my-project",
  "task_id": "task-001",
  "run_id": "20260222-1700000000-12345-1",
  "status": "started",
  "queue_position": 0,
  "depends_on": []
}
```

`status` values for task creation:

- `started`: planner started this root task immediately.
- `queued`: planner accepted the task but postponed execution due `max_concurrent_root_tasks`.

When `status` is `queued`, `queue_position` is `1..N` (oldest queued task is `1`).

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 400 | Bad Request | Invalid JSON, missing required fields, invalid identifiers |
| 404 | Not Found | Threaded parent task/message reference not found |
| 409 | Conflict | Invalid threaded parent type (only `QUESTION` and `FACT` are allowed sources) |
| 500 | Internal Server Error | Failed to create task directory or files |
| 503 | Service Unavailable | Task execution disabled (`--disable-task-start`) |

**Threaded Answer Workflow Notes:**
- Set `thread_parent` to create a child task linked to an existing task message.
- Parent source message types currently supported: `QUESTION`, `FACT`.
- `thread_message_type` is restricted to `USER_REQUEST` for this workflow.
- The server persists linkage in:
  - child task file: `TASK-THREAD-LINK.yaml`
  - child task bus: `TASK-MESSAGE-BUS.md` (`USER_REQUEST` with parent metadata)
  - parent task bus: `TASK-MESSAGE-BUS.md` backlink message for child-task navigation

#### GET /api/v1/tasks

List all tasks.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | No | Filter by project |

**Request:**
```bash
# List all tasks
curl http://localhost:14355/api/v1/tasks

# List tasks for a project
curl http://localhost:14355/api/v1/tasks?project_id=my-project
```

**Response:** `200 OK`
```json
{
  "tasks": [
    {
      "project_id": "my-project",
      "task_id": "task-001",
      "status": "queued",
      "queue_position": 1,
      "last_activity": "2026-02-05T10:00:00Z",
      "runs": [
        {
          "run_id": "run_20260205_100001_abc123",
          "project_id": "my-project",
          "task_id": "task-001",
          "status": "running",
          "start_time": "2026-02-05T10:00:01Z"
        }
      ]
    }
  ]
}
```

#### GET /api/v1/tasks/:taskId

Get task details.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |

**Request:**
```bash
curl "http://localhost:14355/api/v1/tasks/task-001?project_id=my-project"
```

**Response:** `200 OK`
```json
{
  "project_id": "my-project",
  "task_id": "task-001",
  "status": "queued",
  "queue_position": 1,
  "last_activity": "2026-02-05T10:00:00Z",
  "runs": [
    {
      "run_id": "run_20260205_100001_abc123",
      "project_id": "my-project",
      "task_id": "task-001",
      "status": "running",
      "start_time": "2026-02-05T10:00:01Z"
    }
  ]
}
```

Task status values include `queued`, `running`, `blocked`, `completed`, `failed`, and `idle`.

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 400 | Bad Request | Missing project_id |
| 404 | Not Found | Task does not exist |

#### DELETE /api/v1/tasks/:taskId

Cancel a running task.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |

**Request:**
```bash
curl -X DELETE "http://localhost:14355/api/v1/tasks/task-001?project_id=my-project"
```

**Response:** `200 OK`
```json
{
  "status": "cancelled"
}
```

---

### Runs

#### GET /api/v1/runs

List all runs across all tasks.

**Request:**
```bash
curl http://localhost:14355/api/v1/runs
```

**Response:** `200 OK`
```json
{
  "runs": [
    {
      "run_id": "run_20260205_100001_abc123",
      "project_id": "my-project",
      "task_id": "task-001",
      "status": "success",
      "start_time": "2026-02-05T10:00:01Z",
      "end_time": "2026-02-05T10:01:30Z",
      "exit_code": 0
    },
    {
      "run_id": "run_20260205_100002_def456",
      "project_id": "my-project",
      "task_id": "task-002",
      "status": "running",
      "start_time": "2026-02-05T10:02:00Z"
    }
  ]
}
```

**Run Status Values:**

| Status | Description |
|--------|-------------|
| `created` | Run created, not started |
| `running` | Run is executing |
| `success` | Run completed successfully (exit code 0) |
| `failed` | Run failed (non-zero exit code) |
| `stopped` | Run was stopped by user |

#### GET /api/v1/runs/:runId

Get run details (includes metadata and full logs).

**Request:**
```bash
curl http://localhost:14355/api/v1/runs/run_20260205_100001_abc123
```

**Response:** `200 OK`
```json
{
  "run_id": "run_20260205_100001_abc123",
  "project_id": "my-project",
  "task_id": "task-001",
  "status": "success",
  "start_time": "2026-02-05T10:00:01Z",
  "end_time": "2026-02-05T10:01:30Z",
  "exit_code": 0,
  "agent_version": "2.1.50",
  "error_summary": "",
  "output": "Starting agent...\nExecuting prompt...\nCompleted successfully.\n"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | string | Unique run identifier |
| `project_id` | string | Project this run belongs to |
| `task_id` | string | Task this run belongs to |
| `status` | string | Run status (`created`, `running`, `success`, `failed`, `stopped`) |
| `start_time` | string | ISO 8601 start timestamp |
| `end_time` | string | ISO 8601 end timestamp (omitted if still running) |
| `exit_code` | int | Process exit code (omitted if still running) |
| `agent_version` | string | Version of the agent CLI that executed the run (e.g. `"2.1.50"`) |
| `error_summary` | string | Human-readable description of the exit code (e.g. `"Process killed (OOM or external signal)"`) |

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 404 | Not Found | Run does not exist |

#### GET /api/v1/runs/:runId/info

Get run metadata (without full logs).

**Request:**
```bash
curl http://localhost:14355/api/v1/runs/run_20260205_100001_abc123/info
```

**Response:** `200 OK`
```json
{
  "run_id": "run_20260205_100001_abc123",
  "project_id": "my-project",
  "task_id": "task-001",
  "status": "success",
  "start_time": "2026-02-05T10:00:01Z",
  "end_time": "2026-02-05T10:01:30Z",
  "exit_code": 0,
  "agent_version": "2.1.50",
  "error_summary": ""
}
```

Use this endpoint when you only need metadata, not the full output.

#### GET /api/v1/runs/:runId/stream

Stream run logs in real-time using Server-Sent Events (SSE).

**Request:**
```bash
curl -N http://localhost:14355/api/v1/runs/run_20260205_100001_abc123/stream
```

**Response:** `200 OK` (text/event-stream)
```
data: {"type":"log","timestamp":"2026-02-05T10:00:01Z","line":"Starting agent..."}

data: {"type":"log","timestamp":"2026-02-05T10:00:02Z","line":"Executing prompt..."}

data: {"type":"log","timestamp":"2026-02-05T10:00:05Z","line":"Agent output..."}

data: {"type":"status","status":"running"}

data: {"type":"log","timestamp":"2026-02-05T10:01:30Z","line":"Completed successfully."}

data: {"type":"status","status":"success","exit_code":0}

data: {"type":"done"}
```

**Event Types:**

| Type | Description | Fields |
|------|-------------|--------|
| `log` | Log line | `timestamp`, `line` |
| `status` | Status update | `status`, `exit_code` (optional) |
| `done` | Stream complete | None |

**JavaScript Example:**
```javascript
const eventSource = new EventSource(
  'http://localhost:14355/api/v1/runs/run_20260205_100001_abc123/stream'
);

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);

  if (data.type === 'log') {
    console.log(`[${data.timestamp}] ${data.line}`);
  } else if (data.type === 'status') {
    console.log(`Status: ${data.status}`);
  } else if (data.type === 'done') {
    console.log('Stream complete');
    eventSource.close();
  }
};

eventSource.onerror = (error) => {
  console.error('Stream error:', error);
  eventSource.close();
};
```

#### POST /api/v1/runs/:runId/stop

Stop a running task.

**Request:**
```bash
curl -X POST http://localhost:14355/api/v1/runs/run_20260205_100001_abc123/stop
```

**Response:** `200 OK`
```json
{
  "status": "stopped"
}
```

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 404 | Not Found | Run does not exist |
| 500 | Internal Server Error | Failed to stop process |

#### GET /api/projects

List projects visible to the web UI.

Projects are assembled from run metadata (`run-info.yaml`) plus persisted project markers (`PROJECT-ROOT.txt`) created by `POST /api/projects`.

**Request:**

```bash
curl "http://localhost:14355/api/projects"
```

**Response:** `200 OK`

```json
{
  "projects": [
    {
      "id": "my-project",
      "last_activity": "2026-02-22T17:42:10Z",
      "task_count": 3,
      "project_root": "/Users/alice/Work/my-project"
    }
  ]
}
```

**Fields:**

| Field | Type | Description |
|------|------|-------------|
| `id` | string | Project identifier |
| `last_activity` | string | Most recent run or project marker timestamp |
| `task_count` | int | Number of known tasks with runs |
| `project_root` | string | Saved home/work folder path (when configured) |

#### POST /api/projects

Create a project and persist its default home/work folder path.

This creates:

- project directory: `<root>/<project_id>/`
- project root marker file: `<root>/<project_id>/PROJECT-ROOT.txt`

`<root>` is the conductor API root (`--root`, default `~/run-agent`).

**Request Body:**

```json
{
  "project_id": "my-project",
  "project_root": "/Users/alice/Work/my-project"
}
```

**Validation rules:**

- `project_id` is required and must pass identifier validation
- `project_id` must be unique (duplicate IDs return `409 Conflict`)
- `project_root` is required
- `project_root` must resolve to an absolute, existing directory
- `project_root` must be outside the conductor storage root

**Request:**

```bash
curl -X POST "http://localhost:14355/api/projects" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "project_root": "/Users/alice/Work/my-project"
  }'
```

**Response:** `201 Created`

```json
{
  "id": "my-project",
  "last_activity": "2026-02-22T17:42:10Z",
  "task_count": 0,
  "project_root": "/Users/alice/Work/my-project"
}
```

**Errors:**

| Status | Cause |
|--------|-------|
| 400 Bad Request | Missing/invalid `project_id` or `project_root`; non-absolute path; non-existent directory |
| 409 Conflict | Project ID already exists |
| 500 Internal Server Error | Filesystem write/stat failure |

#### GET /api/projects/home-dirs

Return deduplicated project home/work folders used by the UI for datalist suggestions.

Sources include:

- run `cwd` values from existing runs
- persisted `PROJECT-ROOT.txt` markers created by `POST /api/projects`

**Request:**

```bash
curl "http://localhost:14355/api/projects/home-dirs"
```

**Response:** `200 OK`

```json
{
  "dirs": [
    "/Users/alice/Work/my-project",
    "/Users/alice/Work/other-project"
  ]
}
```

#### DELETE /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}

Delete a completed or failed run directory from disk. This permanently removes all files for the specified run (agent output, stdout, stderr, prompt, run-info.yaml).

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `project_id` | Project identifier |
| `task_id` | Task identifier |
| `run_id` | Run identifier |

**Request:**

```bash
curl -X DELETE \
  "http://localhost:14355/api/projects/my-project/tasks/task-20260220-140000-hello/runs/20260220-1400000000-abc12345"
```

**Response:** `204 No Content`

No response body on success.

**Errors:**

| Status | Cause |
|--------|-------|
| 403 Forbidden | Request came from web UI/browser context (destructive actions disabled in UI pathways) |
| 404 Not Found | Run or task directory does not exist |
| 409 Conflict | Run is still in `running` status; stop it first |
| 500 Internal Server Error | Filesystem error removing the run directory |

**Notes:**

- Only completed or failed runs can be deleted. Attempting to delete a running run returns `409 Conflict`.
- Use `POST .../stop` first if you need to terminate a running run before deleting it.
- Deleting a run is permanent and cannot be undone.
- The web UI does not expose this action; use CLI or explicit API clients.

#### DELETE /api/projects/{project_id}/tasks/{task_id}

Delete an entire task directory and all its runs from disk. This permanently removes the task prompt, all run directories, output files, and the task-level message bus.

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `project_id` | Project identifier |
| `task_id` | Task identifier |

**Request:**

```bash
curl -X DELETE \
  "http://localhost:14355/api/projects/my-project/tasks/task-20260220-140000-hello"
```

**Response:** `204 No Content`

No response body on success.

**Errors:**

| Status | Cause |
|--------|-------|
| 403 Forbidden | Request came from web UI/browser context (destructive actions disabled in UI pathways) |
| 404 Not Found | Task directory does not exist |
| 409 Conflict | At least one run is still in `running` status; stop all runs first |
| 500 Internal Server Error | Filesystem error removing the task directory |

**Notes:**

- Use `run-agent task delete` or `POST .../stop` on each running run before deleting.
- Deleting a task is permanent and removes all associated runs, output files, and the task message bus.
- The CLI counterpart is `run-agent task delete --project <p> --task <t>` (use `--force` to skip the running-run check).

---

#### GET /api/projects/{project_id}/stats

Return aggregate statistics for a project: task count, run counts by status, and message bus totals. The web UI uses this endpoint to populate the stats bar at the top of the task list.

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `project_id` | Project identifier |

**Request:**

```bash
curl "http://localhost:14355/api/projects/my-project/stats"
```

**Response:** `200 OK`
```json
{
  "project_id": "my-project",
  "total_tasks": 12,
  "total_runs": 47,
  "running_runs": 2,
  "completed_runs": 41,
  "failed_runs": 3,
  "crashed_runs": 1,
  "message_bus_files": 13,
  "message_bus_total_bytes": 524288
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `project_id` | string | Project identifier |
| `total_tasks` | int | Total number of task directories found |
| `total_runs` | int | Total number of run directories across all tasks |
| `running_runs` | int | Runs currently in `running` status |
| `completed_runs` | int | Runs in `completed` status |
| `failed_runs` | int | Runs in `failed` status |
| `crashed_runs` | int | Runs in any other terminal status (e.g. `crashed`) |
| `message_bus_files` | int | Number of message bus files (task + project level) |
| `message_bus_total_bytes` | int64 | Total size in bytes of all message bus files |

**Errors:**

| Status | Cause |
|--------|-------|
| 404 Not Found | Project directory does not exist |
| 500 Internal Server Error | Filesystem error reading the project directory |

---

#### GET /api/v1/runs/stream/all

Stream all run updates in real-time (SSE).

**Request:**
```bash
curl -N http://localhost:14355/api/v1/runs/stream/all
```

**Response:** `200 OK` (text/event-stream)
```
data: {"type":"run_created","run_id":"run_20260205_100001_abc123","project_id":"my-project","task_id":"task-001"}

data: {"type":"run_started","run_id":"run_20260205_100001_abc123","status":"running"}

data: {"type":"run_updated","run_id":"run_20260205_100001_abc123","status":"success","exit_code":0}
```

Use this endpoint to monitor all runs in real-time.

---

### Messages

#### GET /api/v1/messages

Get messages from the message bus.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `task_id` | string | No | Filter by task (optional) |
| `after` | string | No | Get messages after this ID |

**Request:**
```bash
# Get all project messages
curl "http://localhost:14355/api/v1/messages?project_id=my-project"

# Get task-specific messages
curl "http://localhost:14355/api/v1/messages?project_id=my-project&task_id=task-001"

# Get messages after a specific message ID
curl "http://localhost:14355/api/v1/messages?project_id=my-project&after=msg_123"
```

**Response:** `200 OK`
```json
{
  "messages": [
    {
      "msg_id": "msg_001",
      "timestamp": "2026-02-05T10:00:01Z",
      "type": "task_start",
      "project_id": "my-project",
      "task_id": "task-001",
      "run_id": "run_20260205_100001_abc123",
      "body": "Task started"
    },
    {
      "msg_id": "msg_002",
      "timestamp": "2026-02-05T10:00:05Z",
      "type": "progress",
      "project_id": "my-project",
      "task_id": "task-001",
      "run_id": "run_20260205_100001_abc123",
      "parents": ["msg_001"],
      "body": "Processing..."
    }
  ]
}
```

**Message Types:**

| Type | Description |
|------|-------------|
| `task_start` | Task started |
| `task_complete` | Task completed |
| `task_failed` | Task failed |
| `progress` | Progress update |
| `child_request` | Request to spawn child task |
| `custom` | Custom message |

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 400 | Bad Request | Missing project_id |
| 404 | Not Found | Message ID not found (after parameter) |

#### GET /api/v1/messages/stream

Stream message bus updates in real-time (SSE).

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `task_id` | string | No | Filter by task (optional) |

**Request:**
```bash
# Stream all project messages
curl -N "http://localhost:14355/api/v1/messages/stream?project_id=my-project"

# Stream task-specific messages
curl -N "http://localhost:14355/api/v1/messages/stream?project_id=my-project&task_id=task-001"
```

**Response:** `200 OK` (text/event-stream)
```
data: {"msg_id":"msg_001","timestamp":"2026-02-05T10:00:01Z","type":"task_start","body":"Task started"}

data: {"msg_id":"msg_002","timestamp":"2026-02-05T10:00:05Z","type":"progress","body":"Processing..."}
```

---

### POST /api/projects/{project_id}/messages

Post a message to the project-level message bus.

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `project_id` | Project identifier |

**Request Body:**
```json
{
  "type": "USER",
  "body": "Please prioritize the auth subsystem"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Message type (default: `USER`). Typical values: `USER`, `FACT`, `PROGRESS`, `DECISION`, `ERROR`, `QUESTION` |
| `body` | string | Yes | Message body text |

**Request:**
```bash
curl -X POST \
  "http://localhost:14355/api/projects/my-project/messages" \
  -H "Content-Type: application/json" \
  -d '{"type": "USER", "body": "Focus on auth module"}'
```

**Response:** `201 Created`
```json
{
  "msg_id": "2026-02-21T10:00:01.000000000Z-001",
  "timestamp": "2026-02-21T10:00:01Z"
}
```

---

### POST /api/projects/{project_id}/tasks/{task_id}/messages

Post a message to the task-level message bus.

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `project_id` | Project identifier |
| `task_id` | Task identifier |

**Request Body:** Same as `POST /api/projects/{project_id}/messages`

**Request:**
```bash
curl -X POST \
  "http://localhost:14355/api/projects/my-project/tasks/task-20260221-100000-my-task/messages" \
  -H "Content-Type: application/json" \
  -d '{"type": "DECISION", "body": "Using approach B for the refactor"}'
```

**Response:** `201 Created`
```json
{
  "msg_id": "2026-02-21T10:00:02.000000000Z-001",
  "timestamp": "2026-02-21T10:00:02Z"
}
```

---

### POST /api/projects/{project_id}/tasks/{task_id}/resume

Remove the task's `DONE` file so the Ralph Loop can restart the task.

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `project_id` | Project identifier |
| `task_id` | Task identifier |

**Request:**
```bash
curl -X POST \
  "http://localhost:14355/api/projects/my-project/tasks/task-20260221-100000-my-task/resume"
```

**Response:** `200 OK`
```json
{
  "project_id": "my-project",
  "task_id": "task-20260221-100000-my-task",
  "resumed": true
}
```

**Errors:**

| Status | Cause |
|--------|-------|
| 400 Bad Request | Task has no `DONE` file (nothing to resume) |
| 404 Not Found | Task directory does not exist |

---

## Common Patterns

### Create and Monitor a Task

```bash
# 1. Create task
RESPONSE=$(curl -s -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-001",
    "agent_type": "codex",
    "prompt": "Write hello world"
  }')

# 2. Get the latest run ID
RUN_ID=$(curl -s "http://localhost:14355/api/v1/tasks/task-001?project_id=my-project" | \
  jq -r '.runs[0].run_id')

# 3. Stream logs
curl -N "http://localhost:14355/api/v1/runs/$RUN_ID/stream"
```

### Poll for Task Completion

```bash
# Poll every 5 seconds
while true; do
  STATUS=$(curl -s "http://localhost:14355/api/v1/runs/$RUN_ID/info" | \
    jq -r '.status')

  echo "Status: $STATUS"

  if [[ "$STATUS" == "success" || "$STATUS" == "failed" ]]; then
    break
  fi

  sleep 5
done
```

### Monitor All Runs

```bash
# Stream all run updates
curl -N http://localhost:14355/api/v1/runs/stream/all | \
  while IFS= read -r line; do
    if [[ "$line" == data:* ]]; then
      echo "${line#data: }" | jq .
    fi
  done
```

### Get Message Bus Updates

```bash
# Get latest message ID
LAST_MSG=$(curl -s "http://localhost:14355/api/v1/messages?project_id=my-project" | \
  jq -r '.messages[-1].msg_id')

# Poll for new messages
while true; do
  curl -s "http://localhost:14355/api/v1/messages?project_id=my-project&after=$LAST_MSG" | \
    jq '.messages[]'
  sleep 5
done
```

## Error Handling

### HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | OK | Success |
| 400 | Bad Request | Check request body and parameters |
| 404 | Not Found | Resource doesn't exist |
| 405 | Method Not Allowed | Use correct HTTP method |
| 500 | Internal Server Error | Check server logs |
| 503 | Service Unavailable | Task execution disabled |

### Error Response Format

```json
{
  "error": "detailed error message"
}
```

### Example Error Handling (Bash)

```bash
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"project_id":"test","task_id":"task-001","agent_type":"codex","prompt":"test"}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -eq 200 ]; then
  echo "Success: $BODY"
else
  echo "Error ($HTTP_CODE): $BODY"
fi
```

## CORS Configuration

To access the API from a web frontend on a different origin, configure CORS in config.yaml:

```yaml
api:
  cors_origins:
    - http://localhost:3000
    - https://app.example.com
```

The API will respond with appropriate CORS headers:
```
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

## Rate Limiting

Currently, there is no built-in rate limiting. For production use, consider:
- Reverse proxy with rate limiting (nginx, Caddy)
- API gateway (Kong, Tyk)
- Custom middleware

## Next Steps

- [CLI Reference](cli-reference.md) - Command-line interface
- [Web UI Guide](web-ui.md) - Using the web interface
- [Configuration](configuration.md) - Configure the server
- [Examples](../examples/basic-usage.md) - API usage examples
