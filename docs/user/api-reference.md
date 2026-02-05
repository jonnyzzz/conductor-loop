# API Reference

Complete REST API reference for Conductor Loop. The API provides endpoints for task management, run monitoring, log streaming, and message bus access.

## Base URL

```
http://localhost:8080/api/v1
```

Change the host and port in your config.yaml:

```yaml
api:
  host: 0.0.0.0
  port: 8080
```

## Authentication

Currently, the API does not require authentication. This is suitable for local development and trusted environments.

For production deployments, consider:
- Running behind a reverse proxy with authentication (nginx, Caddy)
- Using network isolation (VPN, private network)
- Implementing custom authentication middleware

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

### Health and Version

#### GET /api/v1/health

Check if the server is running.

**Request:**
```bash
curl http://localhost:8080/api/v1/health
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
curl http://localhost:8080/api/v1/version
```

**Response:** `200 OK`
```json
{
  "version": "dev"
}
```

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
  "config": {
    "key": "value"
  }
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `task_id` | string | Yes | Task identifier |
| `agent_type` | string | Yes | Agent to use (codex, claude, etc.) |
| `prompt` | string | Yes | Task prompt/instructions |
| `config` | object | No | Additional configuration |

**Identifier Rules:**
- Must contain only alphanumeric, dash, underscore
- No spaces allowed
- Case-sensitive

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-001",
    "agent_type": "codex",
    "prompt": "Write a Python script that prints hello world"
  }'
```

**Response:** `200 OK`
```json
{
  "project_id": "my-project",
  "task_id": "task-001",
  "status": "created"
}
```

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 400 | Bad Request | Invalid JSON, missing required fields, invalid identifiers |
| 500 | Internal Server Error | Failed to create task directory or files |
| 503 | Service Unavailable | Task execution disabled (`--disable-task-start`) |

#### GET /api/v1/tasks

List all tasks.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | No | Filter by project |

**Request:**
```bash
# List all tasks
curl http://localhost:8080/api/v1/tasks

# List tasks for a project
curl http://localhost:8080/api/v1/tasks?project_id=my-project
```

**Response:** `200 OK`
```json
{
  "tasks": [
    {
      "project_id": "my-project",
      "task_id": "task-001",
      "status": "running",
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
curl "http://localhost:8080/api/v1/tasks/task-001?project_id=my-project"
```

**Response:** `200 OK`
```json
{
  "project_id": "my-project",
  "task_id": "task-001",
  "status": "running",
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
curl -X DELETE "http://localhost:8080/api/v1/tasks/task-001?project_id=my-project"
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
curl http://localhost:8080/api/v1/runs
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
curl http://localhost:8080/api/v1/runs/run_20260205_100001_abc123
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
  "output": "Starting agent...\nExecuting prompt...\nCompleted successfully.\n"
}
```

**Errors:**

| Status | Error | Cause |
|--------|-------|-------|
| 404 | Not Found | Run does not exist |

#### GET /api/v1/runs/:runId/info

Get run metadata (without full logs).

**Request:**
```bash
curl http://localhost:8080/api/v1/runs/run_20260205_100001_abc123/info
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
  "exit_code": 0
}
```

Use this endpoint when you only need metadata, not the full output.

#### GET /api/v1/runs/:runId/stream

Stream run logs in real-time using Server-Sent Events (SSE).

**Request:**
```bash
curl -N http://localhost:8080/api/v1/runs/run_20260205_100001_abc123/stream
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
  'http://localhost:8080/api/v1/runs/run_20260205_100001_abc123/stream'
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
curl -X POST http://localhost:8080/api/v1/runs/run_20260205_100001_abc123/stop
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

#### GET /api/v1/runs/stream/all

Stream all run updates in real-time (SSE).

**Request:**
```bash
curl -N http://localhost:8080/api/v1/runs/stream/all
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
curl "http://localhost:8080/api/v1/messages?project_id=my-project"

# Get task-specific messages
curl "http://localhost:8080/api/v1/messages?project_id=my-project&task_id=task-001"

# Get messages after a specific message ID
curl "http://localhost:8080/api/v1/messages?project_id=my-project&after=msg_123"
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
curl -N "http://localhost:8080/api/v1/messages/stream?project_id=my-project"

# Stream task-specific messages
curl -N "http://localhost:8080/api/v1/messages/stream?project_id=my-project&task_id=task-001"
```

**Response:** `200 OK` (text/event-stream)
```
data: {"msg_id":"msg_001","timestamp":"2026-02-05T10:00:01Z","type":"task_start","body":"Task started"}

data: {"msg_id":"msg_002","timestamp":"2026-02-05T10:00:05Z","type":"progress","body":"Processing..."}
```

---

## Common Patterns

### Create and Monitor a Task

```bash
# 1. Create task
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-001",
    "agent_type": "codex",
    "prompt": "Write hello world"
  }')

# 2. Get the latest run ID
RUN_ID=$(curl -s "http://localhost:8080/api/v1/tasks/task-001?project_id=my-project" | \
  jq -r '.runs[0].run_id')

# 3. Stream logs
curl -N "http://localhost:8080/api/v1/runs/$RUN_ID/stream"
```

### Poll for Task Completion

```bash
# Poll every 5 seconds
while true; do
  STATUS=$(curl -s "http://localhost:8080/api/v1/runs/$RUN_ID/info" | \
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
curl -N http://localhost:8080/api/v1/runs/stream/all | \
  while IFS= read -r line; do
    if [[ "$line" == data:* ]]; then
      echo "${line#data: }" | jq .
    fi
  done
```

### Get Message Bus Updates

```bash
# Get latest message ID
LAST_MSG=$(curl -s "http://localhost:8080/api/v1/messages?project_id=my-project" | \
  jq -r '.messages[-1].msg_id')

# Poll for new messages
while true; do
  curl -s "http://localhost:8080/api/v1/messages?project_id=my-project&after=$LAST_MSG" | \
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
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/api/v1/tasks \
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
