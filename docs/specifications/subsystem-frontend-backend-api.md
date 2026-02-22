# Frontend-Backend API Contract Subsystem

## Overview
Defines the REST/JSON + SSE API contract between the React monitoring UI (frontend) and the Go run-agent serve backend. This API provides read access to project/task/run data, message bus streaming and posting, and task creation capabilities.

## Goals
- Provide a stable API for the monitoring UI.
- Enable TypeScript type safety through integration tests.
- Support efficient streaming of logs and message bus updates.
- Enable task creation and message posting from the UI.
- Support selecting a backend host (single active host at a time).

## Non-Goals
- Remote multi-user access (localhost only in MVP).
- Write operations beyond task creation and message posting.
- Authentication/authorization in MVP.
- Global search across all projects/tasks.
- Cross-host aggregation (user picks one host at a time).

## Technology Stack
- Protocol: REST (HTTP/1.1 or HTTP/2) + Server-Sent Events (SSE).
- Format: JSON for REST; text/event-stream for SSE.
- Base path: `/api/v1`.
- Host: localhost only for MVP, optional remote host for future.
- Port: configurable (default 14355).

## Host Selection
- UI can store multiple backend base URLs (local storage or config).
- UI uses `GET /api/v1/health` and `GET /api/v1/version` to validate a host.
- UI shows the active host label in the header.

## API Endpoints

### Health & Version

#### GET /api/v1/health

**Response**:
```json
{
  "status": "ok"
}
```

#### GET /api/v1/version

**Response**:
```json
{
  "version": "v1"
}
```

### Project Management

#### GET /api/v1/projects

List all projects.

**Response**:
```json
{
  "projects": [
    {
      "id": "swarm",
      "last_activity": "2026-02-04T17:31:55Z",
      "task_count": 3
    }
  ]
}
```

#### GET /api/v1/projects/:project_id

Get project details.

**Response**:
```json
{
  "id": "swarm",
  "last_activity": "2026-02-04T17:31:55Z",
  "home_folders": {
    "project_root": "/path/to/projects/swarm",
    "source_folders": ["/path/to/projects/swarm/src"],
    "additional_folders": []
  },
  "tasks": [
    {
      "id": "20260131-205800-planning",
      "status": "running",
      "last_activity": "2026-02-04T17:31:55Z"
    }
  ]
}
```

### Task Management

#### GET /api/v1/projects/:project_id/tasks

List all tasks for a project.

**Response**:
```json
{
  "tasks": [
    {
      "id": "20260131-205800-planning",
      "name": "planning",
      "status": "running",
      "last_activity": "2026-02-04T17:31:55Z",
      "run_count": 15
    }
  ]
}
```

#### GET /api/v1/projects/:project_id/tasks/:task_id

Get task details.

**Response**:
```json
{
  "id": "20260131-205800-planning",
  "name": "planning",
  "project_id": "swarm",
  "status": "running",
  "last_activity": "2026-02-04T17:31:55Z",
  "created_at": "2026-01-31T20:58:00Z",
  "done": false,
  "state": "Reviewing subsystem specifications...",
  "runs": [
    {
      "id": "20260204-173042-12345",
      "agent": "claude",
      "status": "completed",
      "exit_code": 0,
      "start_time": "2026-02-04T17:30:42Z",
      "end_time": "2026-02-04T17:31:55Z"
    }
  ]
}
```

#### POST /api/v1/projects/:project_id/tasks

Create a new task or restart an existing task.

**Request**:
```json
{
  "task_id": "20260204-180000-newfeature",
  "prompt": "Add feature X to the system...",
  "agent_type": "codex",
  "project_root": "/path/to/projects/myproject",
  "attach_mode": "restart",
  "config": {
    "JRUN_PARENT_ID": "20260204-170000-11111"
  }
}
```

**Response**:
```json
{
  "task_id": "20260204-180000-newfeature",
  "status": "started",
  "run_id": "20260204-180005-12400"
}
```

### Run Management

#### GET /api/v1/projects/:project_id/tasks/:task_id/runs/:run_id

Get run metadata.

**Response**:
```json
{
  "version": 1,
  "run_id": "20260204-173042-12345",
  "project_id": "swarm",
  "task_id": "20260131-205800-planning",
  "parent_run_id": "",
  "previous_run_id": "20260204-172000-12340",
  "agent": "claude",
  "pid": 12345,
  "pgid": 12345,
  "start_time": "2026-02-04T17:30:42.569Z",
  "end_time": "2026-02-04T17:31:55.789Z",
  "exit_code": 0,
  "cwd": "/path/to/projects/swarm",
  "backend_provider": "anthropic",
  "backend_model": "claude-sonnet-4-5"
}
```

### File Access

#### GET /api/v1/projects/:project_id/tasks/:task_id/file

Read task-level files (TASK.md, TASK_STATE.md).

Query parameters:
- `name`: file name (e.g., `TASK.md`, `TASK_STATE.md`).

**Response**:
```json
{
  "name": "TASK_STATE.md",
  "content": "Reviewing subsystem specifications...",
  "modified": "2026-02-04T17:31:55Z"
}
```

#### GET /api/v1/projects/:project_id/tasks/:task_id/runs/:run_id/file

Read run-level files (prompt.md, output.md, agent-stdout.txt, agent-stderr.txt).

Query parameters:
- `name`: file name (e.g., `output.md`, `agent-stdout.txt`).
- `tail`: optional number of lines to tail (default: all).

**Response**:
```json
{
  "name": "output.md",
  "content": "...",
  "modified": "2026-02-04T17:31:55Z",
  "size_bytes": 12345
}
```

Security: backend validates that `name` is one of the allowed files and prevents path traversal.

### Message Bus

#### GET /api/v1/projects/:project_id/bus

Read the project message bus.

Query parameters:
- `after`: optional message id to start after.

#### GET /api/v1/projects/:project_id/tasks/:task_id/bus

Read the task message bus.

Query parameters:
- `after`: optional message id to start after.

#### POST /api/v1/projects/:project_id/bus

Post a message to the project message bus.

**Request**:
```json
{
  "type": "USER",
  "body": "Please clarify the requirements...",
  "parents": ["MSG-20260204-173000-1"]
}
```

**Response**:
```json
{
  "msg_id": "MSG-20260204-173100-2"
}
```

#### POST /api/v1/projects/:project_id/tasks/:task_id/bus

Post a message to the task message bus (same payload as project bus).

#### GET /api/v1/projects/:project_id/bus/stream

Stream project message bus via SSE.

Query parameters:
- `after`: optional message id to start after.

SSE events:
```
event: message
data: {"msg_id": "MSG-20260204-173100-2", "ts": "2026-02-04T17:31:00Z", "type": "USER", "project_id": "swarm", "body": "..."}
```

#### GET /api/v1/projects/:project_id/tasks/:task_id/bus/stream

Stream task message bus via SSE (same payload as project bus stream).

### Log Streaming

#### GET /api/v1/runs/stream/all

Stream stdout/stderr for all runs via SSE.

#### GET /api/v1/runs/:run_id/stream

Stream stdout/stderr for a single run via SSE.

SSE events:
```
event: log
data: {"run_id": "20260204-173042-12345", "stream": "stdout", "line": "Starting agent...", "timestamp": "2026-02-04T17:30:43Z"}

event: status
data: {"run_id": "20260204-173042-12345", "status": "completed", "exit_code": 0}

event: heartbeat
data: {}
```

SSE cursor behavior:
- Clients may send `Last-Event-ID` with a cursor value to resume.
- Cursor format is `s=<stdout_lines>;e=<stderr_lines>` (single integer applies to both).

## Error Handling

### Standard Error Response

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Project 'foo' not found",
    "details": {}
  }
}
```

### HTTP Status Codes

- `200 OK`: success.
- `201 Created`: task created.
- `202 Accepted`: async stop accepted.
- `400 Bad Request`: invalid request parameters.
- `401 Unauthorized`: missing/invalid auth (if enabled).
- `404 Not Found`: resource not found.
- `405 Method Not Allowed`: wrong HTTP method.
- `409 Conflict`: ambiguous identifiers or already-finished runs.
- `500 Internal Server Error`: backend error.

## TypeScript Type Generation

### Integration Tests

Create Node.js or browser-based integration tests that:
1. Start a test instance of run-agent serve.
2. Make API requests using fetch or axios.
3. Validate response structure against TypeScript interfaces.
4. Fail if types don't match.

### Type Definition Example

```typescript
interface Project {
  id: string;
  last_activity: string;
  task_count: number;
}

interface ProjectsResponse {
  projects: Project[];
}

interface RunInfo {
  version: number;
  run_id: string;
  project_id: string;
  task_id: string;
  parent_run_id: string;
  previous_run_id: string;
  agent: string;
  pid: number;
  pgid: number;
  start_time: string;
  end_time: string;
  exit_code: number;
  cwd: string;
  backend_provider?: string;
  backend_model?: string;
  backend_endpoint?: string;
  commandline?: string;
}
```

### OpenAPI Consideration

Future: generate an OpenAPI 3.0 specification and use openapi-generator (or similar) to auto-generate TypeScript types.

## Performance Considerations

### Caching

- File content responses: no caching (files may change).
- Metadata responses: short-lived cache (2s) for repeated requests.

### Streaming

- SSE connections kept alive with periodic ping (every 30s).
- Reconnection supported via `Last-Event-ID`.

### Rate Limiting

- No rate limiting in MVP (localhost only).

## CORS

- Disabled by default (localhost only; same-origin).
- Enable CORS if remote access is added post-MVP.

## Security

### Path Traversal Prevention

- Backend validates file names against allowed list.
- No user-specified file paths accepted.
- All file access rooted at `~/run-agent`.

### Input Validation

- All JSON payloads validated against schemas.
- String length limits: message body 64KB (same as message bus).
- Reject invalid UTF-8.

## Related Files

- subsystem-monitoring-ui.md (frontend consumer)
- subsystem-message-bus-tools.md (message bus format)
- subsystem-storage-layout.md (data sources)
- subsystem-runner-orchestration.md (run-agent serve implementation)
