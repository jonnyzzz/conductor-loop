# Frontend-Backend API Contract Subsystem

## Overview

Defines the REST/JSON + SSE API contract between the React monitoring UI (frontend) and the Go run-agent serve backend. This API provides read-only access to project/task/run data, message bus streaming, and task creation capabilities.

## Goals

- Provide a stable API for the monitoring UI
- Enable TypeScript type safety through integration tests
- Support efficient streaming of logs and message bus updates
- Enable task creation and management from the UI

## Non-Goals

- Remote multi-user access (localhost only in MVP)
- Write operations beyond task creation and message posting
- Authentication/authorization (localhost assumes trust)
- Global search across all projects/tasks

## Technology Stack

- **Protocol**: REST (HTTP/1.1 or HTTP/2) + Server-Sent Events (SSE)
- **Format**: JSON for REST; text/event-stream for SSE
- **Host**: localhost only (no remote access in MVP)
- **Port**: Configurable (default 8080)

## API Endpoints

### Project Management

#### GET /api/projects

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

#### GET /api/projects/:project_id

Get project details.

**Response**:
```json
{
  "id": "swarm",
  "last_activity": "2026-02-04T17:31:55Z",
  "home_folders": {
    "project_root": "/Users/user/projects/swarm",
    "source_folders": ["/Users/user/projects/swarm/src"],
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

#### GET /api/projects/:project_id/tasks

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

#### GET /api/projects/:project_id/tasks/:task_id

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

#### POST /api/projects/:project_id/tasks

Create a new task or restart existing task.

**Request**:
```json
{
  "task_id": "20260204-180000-newfeature",
  "prompt": "Add feature X to the system...",
  "project_root": "/Users/user/projects/myproject",
  "attach_mode": "restart"  // or "new"
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

#### GET /api/projects/:project_id/tasks/:task_id/runs/:run_id

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
  "cwd": "/Users/user/projects/swarm",
  "backend_provider": "anthropic",
  "backend_model": "claude-sonnet-4-5"
}
```

### File Access

#### GET /api/projects/:project_id/tasks/:task_id/file

Read task-level files (TASK.md, TASK_STATE.md).

**Query Parameters**:
- `name`: File name (e.g., "TASK.md", "TASK_STATE.md")

**Response**:
```json
{
  "name": "TASK_STATE.md",
  "content": "Reviewing subsystem specifications...",
  "modified": "2026-02-04T17:31:55Z"
}
```

#### GET /api/projects/:project_id/tasks/:task_id/runs/:run_id/file

Read run-level files (prompt.md, output.md, agent-stdout.txt, agent-stderr.txt).

**Query Parameters**:
- `name`: File name (e.g., "output.md", "agent-stdout.txt")
- `tail`: Optional, number of lines to tail (default: all)

**Response**:
```json
{
  "name": "output.md",
  "content": "...",
  "modified": "2026-02-04T17:31:55Z",
  "size_bytes": 12345
}
```

**Security**: Backend validates that `name` parameter is one of the allowed files and prevents path traversal.

### Message Bus

#### POST /api/projects/:project_id/bus

Post a message to project message bus.

**Request**:
```json
{
  "type": "USER",
  "message": "Please clarify the requirements...",
  "parents": ["MSG-20260204-173000-1"]
}
```

**Response**:
```json
{
  "msg_id": "MSG-20260204-173100-2"
}
```

#### POST /api/projects/:project_id/tasks/:task_id/bus

Post a message to task message bus.

(Same format as project bus post)

#### GET /api/projects/:project_id/bus/stream

Stream project message bus via SSE.

**Query Parameters**:
- `since`: Optional ISO-8601 timestamp to start streaming from

**SSE Events**:
```
event: message
data: {"msg_id": "MSG-20260204-173100-2", "ts": "2026-02-04T17:31:00Z", "type": "USER", "message": "..."}

event: message
data: {"msg_id": "MSG-20260204-173200-3", "ts": "2026-02-04T17:32:00Z", "type": "ANSWER", "parents": ["MSG-20260204-173100-2"], "message": "..."}
```

#### GET /api/projects/:project_id/tasks/:task_id/bus/stream

Stream task message bus via SSE.

(Same format as project bus stream)

### Log Streaming

#### GET /api/projects/:project_id/tasks/:task_id/logs/stream

Stream all run logs (stdout/stderr) for a task via SSE.

**Query Parameters**:
- `since`: Optional ISO-8601 timestamp
- `run_id`: Optional run ID filter (default: all runs)

**SSE Events**:
```
event: run_start
data: {"run_id": "20260204-173042-12345", "agent": "claude", "start_time": "2026-02-04T17:30:42Z"}

event: log
data: {"run_id": "20260204-173042-12345", "stream": "stdout", "line": "Starting agent..."}

event: log
data: {"run_id": "20260204-173042-12345", "stream": "stderr", "line": "[Warning] ..."}

event: run_end
data: {"run_id": "20260204-173042-12345", "exit_code": 0, "end_time": "2026-02-04T17:31:55Z"}
```

**Behavior**:
- Streams from all runs in chronological order
- New runs automatically included
- Each log line tagged with run_id and stream (stdout/stderr)
- Client-side filtering by run_id or stream

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

- `200 OK`: Success
- `201 Created`: Task created
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Backend error

## TypeScript Type Generation

### Integration Tests

Create Node.js or browser-based integration tests that:
1. Start a test instance of run-agent serve
2. Make API requests using fetch or axios
3. Validate response structure against TypeScript interfaces
4. Fail if types don't match

### Type Definition Example

```typescript
interface Project {
  id: string;
  last_activity: string;  // ISO-8601
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
  start_time: string;  // ISO-8601
  end_time: string;    // ISO-8601
  exit_code: number;
  cwd: string;
  backend_provider?: string;
  backend_model?: string;
  backend_endpoint?: string;
  commandline?: string;
}
```

### OpenAPI Consideration

Future: Generate OpenAPI 3.0 specification and use openapi-generator or similar to auto-generate TypeScript types.

## Performance Considerations

### Caching

- File content responses: no caching (files may change)
- Metadata responses: short-lived cache (2s) for repeated requests

### Streaming

- SSE connections kept alive with periodic ping (every 30s)
- Reconnection supported via `since` parameter

### Rate Limiting

- No rate limiting in MVP (localhost only)

## CORS

- Disabled (localhost only; same-origin)
- Enable CORS if remote access added post-MVP

## Security

### Path Traversal Prevention

- Backend validates file names against allowed list
- No user-specified file paths accepted
- All file access rooted at ~/run-agent

### Input Validation

- All JSON payloads validated against schemas
- String length limits: message body 64KB (same as message bus)
- Reject invalid UTF-8

## Related Files

- subsystem-monitoring-ui.md (frontend consumer)
- subsystem-message-bus-tools.md (message bus format)
- subsystem-storage-layout.md (data sources)
- subsystem-runner-orchestration.md (run-agent serve implementation)
