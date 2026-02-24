# REST API Usage Example

Demonstrates how to interact with Conductor Loop entirely via the REST API using curl commands.

## What This Example Demonstrates

- Creating tasks via API
- Polling for task completion
- Streaming logs via Server-Sent Events (SSE)
- Error handling
- All available API endpoints

## Prerequisites

- Conductor Loop server running
- curl installed
- jq installed (for JSON parsing)

## API Endpoints

### Health and Version
- `GET /api/v1/health` - Health check
- `GET /api/v1/version` - Server version

### Tasks
- `GET /api/v1/tasks` - List all tasks
- `GET /api/v1/tasks/:id` - Get task details
- `POST /api/v1/tasks` - Create new task

### Runs
- `GET /api/v1/runs` - List all runs
- `GET /api/v1/runs/:id` - Get run details
- `GET /api/v1/runs/stream/all` - SSE stream all runs
- `GET /api/v1/runs/:id/stream` - SSE stream single run

### Messages
- `GET /api/v1/messages` - Get message bus entries
- `GET /api/v1/messages/stream` - SSE stream messages

## Scripts

All scripts are in the `scripts/` directory:

1. `01-health-check.sh` - Verify API is running
2. `02-create-task.sh` - Create a new task
3. `03-poll-status.sh` - Poll for task completion
4. `04-stream-logs.sh` - Stream logs via SSE
5. `05-list-runs.sh` - List all runs
6. `06-get-run.sh` - Get specific run details
7. `07-message-bus.sh` - Read message bus
8. `08-stream-messages.sh` - Stream message bus via SSE

## Quick Start

### 1. Start the Server

```bash
run-agent serve --config ../hello-world/config.yaml
```

### 2. Run Health Check

```bash
./scripts/01-health-check.sh
```

Expected output:
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### 3. Create a Task

```bash
./scripts/02-create-task.sh
```

This creates a task and returns:
```json
{
  "task_id": "api-demo-task",
  "project_id": "api-demo",
  "run_id": "run_20260205-115154-12345",
  "status": "pending"
}
```

### 4. Poll for Completion

```bash
./scripts/03-poll-status.sh run_20260205-115154-12345
```

### 5. Stream Logs (Alternative)

```bash
./scripts/04-stream-logs.sh run_20260205-115154-12345
```

## Detailed Examples

### Creating a Task

**Request:**
```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "my-task",
    "agent_type": "codex",
    "prompt": "Write a hello world program in Python",
    "config": {
      "timeout": "300"
    }
  }'
```

**Response:**
```json
{
  "run_id": "run_20260205-115154-12345",
  "project_id": "my-project",
  "task_id": "my-task",
  "agent_type": "codex",
  "status": "pending",
  "start_time": "2026-02-05T11:51:54Z"
}
```

### Polling for Status

**Request:**
```bash
curl http://localhost:14355/api/v1/runs/run_20260205-115154-12345
```

**Response (running):**
```json
{
  "run_id": "run_20260205-115154-12345",
  "project_id": "my-project",
  "task_id": "my-task",
  "agent_type": "codex",
  "status": "running",
  "start_time": "2026-02-05T11:51:54Z",
  "pid": 12345,
  "pgid": 12345
}
```

**Response (completed):**
```json
{
  "run_id": "run_20260205-115154-12345",
  "project_id": "my-project",
  "task_id": "my-task",
  "agent_type": "codex",
  "status": "completed",
  "start_time": "2026-02-05T11:51:54Z",
  "end_time": "2026-02-05T11:52:10Z",
  "exit_code": 0,
  "output": "# Hello World\n\nprint('Hello, World!')\n"
}
```

### Streaming Logs with SSE

**Request:**
```bash
curl -N http://localhost:14355/api/v1/runs/run_20260205-115154-12345/stream
```

**Response (Server-Sent Events):**
```
event: status
data: {"status": "running", "timestamp": "2026-02-05T11:51:54Z"}

event: log
data: {"line": "Agent started", "timestamp": "2026-02-05T11:51:55Z"}

event: log
data: {"line": "Processing prompt...", "timestamp": "2026-02-05T11:51:56Z"}

event: status
data: {"status": "completed", "exit_code": 0, "timestamp": "2026-02-05T11:52:10Z"}
```

### Listing Runs

**Request:**
```bash
curl "http://localhost:14355/api/v1/runs?project_id=my-project&status=completed&limit=10"
```

**Response:**
```json
{
  "runs": [
    {
      "run_id": "run_20260205-115154-12345",
      "project_id": "my-project",
      "task_id": "my-task",
      "status": "completed",
      "start_time": "2026-02-05T11:51:54Z",
      "end_time": "2026-02-05T11:52:10Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

### Reading Message Bus

**Request:**
```bash
curl "http://localhost:14355/api/v1/messages?project_id=my-project&type=FACT"
```

**Response:**
```json
{
  "messages": [
    {
      "msg_id": "20260205115154000000000-12345-1",
      "timestamp": "2026-02-05T11:51:54Z",
      "type": "FACT",
      "project_id": "my-project",
      "task_id": "my-task",
      "run_id": "run_20260205-115154-12345",
      "body": "Task completed successfully"
    }
  ]
}
```

## Error Handling

### Task Creation Failure

**Request:**
```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "test",
    "task_id": "test",
    "agent_type": "nonexistent-agent",
    "prompt": "test"
  }'
```

**Response (400 Bad Request):**
```json
{
  "error": "agent not configured: nonexistent-agent",
  "code": "INVALID_AGENT"
}
```

### Run Not Found

**Request:**
```bash
curl http://localhost:14355/api/v1/runs/nonexistent-run
```

**Response (404 Not Found):**
```json
{
  "error": "run not found: nonexistent-run",
  "code": "NOT_FOUND"
}
```

## Advanced Usage

### Streaming All Runs

Monitor all running tasks in real-time:

```bash
curl -N http://localhost:14355/api/v1/runs/stream/all
```

Use case: Dashboard showing all active tasks

### Filtering Messages

Get only errors from the last hour:

```bash
SINCE=$(date -u -d '1 hour ago' +%Y%m%d%H%M%S)
curl "http://localhost:14355/api/v1/messages?type=ERROR&since_id=${SINCE}"
```

### Creating Tasks Programmatically

**Python example:**
```python
import requests
import time

# Create task
response = requests.post('http://localhost:14355/api/v1/tasks', json={
    'project_id': 'automation',
    'task_id': f'task-{int(time.time())}',
    'agent_type': 'codex',
    'prompt': 'Generate unit tests for app.py'
})

run_id = response.json()['run_id']

# Poll for completion
while True:
    run = requests.get(f'http://localhost:14355/api/v1/runs/{run_id}').json()
    if run['status'] in ['completed', 'failed']:
        break
    time.sleep(5)

# Get output
print(run['output'])
```

### Concurrent Task Creation

Launch multiple tasks in parallel:

```bash
for i in {1..10}; do
    curl -X POST http://localhost:14355/api/v1/tasks \
      -H "Content-Type: application/json" \
      -d "{
        \"project_id\": \"parallel-demo\",
        \"task_id\": \"task-$i\",
        \"agent_type\": \"codex\",
        \"prompt\": \"Process item $i\"
      }" &
done
wait
```

## Performance Considerations

### Polling vs Streaming

**Polling:**
- Simple to implement
- Works with any HTTP client
- Higher latency (poll interval)
- More network traffic

**SSE Streaming:**
- Real-time updates
- Lower latency
- Efficient (push-based)
- Requires SSE client support

**Recommendation:** Use SSE for real-time monitoring, polling for simple scripts.

### Rate Limiting

Be mindful of:
- API rate limits (if configured)
- Agent concurrency limits (default: 16)
- Network bandwidth
- Server capacity

## Integration Examples

### CI/CD Integration

```bash
#!/bin/bash
# .github/workflows/ci.sh

# Create test task
RESPONSE=$(curl -s -X POST http://conductor:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "ci",
    "task_id": "test-'$GITHUB_SHA'",
    "agent_type": "codex",
    "prompt": "Run all tests and report results"
  }')

RUN_ID=$(echo $RESPONSE | jq -r '.run_id')

# Wait for completion
while true; do
    STATUS=$(curl -s http://conductor:14355/api/v1/runs/$RUN_ID | jq -r '.status')
    if [ "$STATUS" = "completed" ]; then
        EXIT_CODE=$(curl -s http://conductor:14355/api/v1/runs/$RUN_ID | jq -r '.exit_code')
        exit $EXIT_CODE
    elif [ "$STATUS" = "failed" ]; then
        exit 1
    fi
    sleep 5
done
```

### Webhook Trigger

```bash
#!/bin/bash
# webhook-handler.sh

# Triggered by external webhook
PROJECT_ID=$1
TASK_TYPE=$2

curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": \"$PROJECT_ID\",
    \"task_id\": \"webhook-$(date +%s)\",
    \"agent_type\": \"claude\",
    \"prompt\": \"Handle $TASK_TYPE event for $PROJECT_ID\"
  }"
```

## Troubleshooting

### Connection Refused
```bash
# Check if server is running
curl http://localhost:14355/api/v1/health
# If fails, start server: run-agent serve
```

### Timeout Errors
```bash
# Increase timeout
curl --max-time 300 http://localhost:14355/api/v1/runs/$RUN_ID
```

### JSON Parse Errors
```bash
# Use jq to validate JSON
echo '$JSON' | jq .
```

### CORS Issues (from browser)
```yaml
# In config.yaml
api:
  cors_origins:
    - http://localhost:3000
```

## Next Steps

After mastering the REST API:

1. Build a custom dashboard using the streaming endpoints
2. Integrate with your existing tools and workflows
3. Create automation scripts for common tasks
4. Deploy with [Docker](../docker-deployment/) for production-ready setup

## Related Examples

- [docker-deployment](../docker-deployment/) - Production deployment
