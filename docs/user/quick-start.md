# Quick Start Guide

Get up and running with Conductor Loop in 5 minutes. This tutorial assumes you've already [installed](installation.md) the binaries.

## Prerequisites

- Conductor Loop installed (see [Installation Guide](installation.md))
- An API token for at least one agent (Claude, Codex, Gemini, etc.)
- Basic terminal/command-line knowledge

## Step 1: Set Up Configuration

Create a minimal configuration file:

```bash
# Create config directory
mkdir -p ~/.conductor/tokens

# Create config.yaml
cat > config.yaml <<EOF
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token
    timeout: 300

defaults:
  agent: codex
  timeout: 300

api:
  host: 0.0.0.0
  port: 8080
  cors_origins:
    - http://localhost:3000

storage:
  runs_dir: ./runs
EOF

# Add your API token
echo "your-codex-token-here" > ~/.conductor/tokens/codex.token
chmod 600 ~/.conductor/tokens/codex.token
```

## Step 2: Start the Server

Start the Conductor server:

```bash
# Start the server
./conductor --config config.yaml --root $(pwd)

# You should see:
# conductor 2026/02/05 10:00:00 Starting Conductor Loop server
# conductor 2026/02/05 10:00:00 API listening on http://0.0.0.0:8080
# conductor 2026/02/05 10:00:00 Root directory: /Users/you/conductor-loop
```

Leave this terminal running and open a new terminal for the next steps.

## Step 3: Test the Server

Verify the server is running:

```bash
# Check health
curl http://localhost:8080/api/v1/health

# Response:
# {"status":"ok","version":"dev"}

# Check version
curl http://localhost:8080/api/v1/version

# Response:
# {"version":"dev"}
```

## Step 4: Run Your First Task

### Simple "Hello World" Task

Create a task that prints "Hello World":

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "codex",
    "prompt": "Write a Python script that prints Hello World",
    "working_dir": "/tmp"
  }'

# Response:
# {
#   "run_id": "run_20260205_100001_abc123",
#   "status": "created",
#   "message": "Task created successfully"
# }
```

Save the `run_id` from the response - you'll need it to check status and view logs.

### Check Task Status

```bash
# Replace RUN_ID with your actual run ID
RUN_ID="run_20260205_100001_abc123"

curl http://localhost:8080/api/v1/runs/$RUN_ID

# Response:
# {
#   "run_id": "run_20260205_100001_abc123",
#   "status": "running",
#   "agent": "codex",
#   "start_time": "2026-02-05T10:00:01Z",
#   "working_dir": "/tmp"
# }
```

Status values:
- `created`: Task created, not started yet
- `running`: Task is executing
- `success`: Task completed successfully
- `failed`: Task failed
- `restarted`: Task was restarted by Ralph Loop

### View Task Logs

Stream the live logs:

```bash
# Stream logs with curl (SSE)
curl -N http://localhost:8080/api/v1/runs/$RUN_ID/stream

# Output (Server-Sent Events):
# data: {"type":"log","timestamp":"2026-02-05T10:00:01Z","line":"Starting task..."}
# data: {"type":"log","timestamp":"2026-02-05T10:00:02Z","line":"Agent: codex"}
# data: {"type":"log","timestamp":"2026-02-05T10:00:03Z","line":"Executing prompt..."}
# ...
```

## Step 5: List All Runs

See all your task executions:

```bash
curl http://localhost:8080/api/v1/runs

# Response:
# {
#   "runs": [
#     {
#       "run_id": "run_20260205_100001_abc123",
#       "status": "success",
#       "agent": "codex",
#       "start_time": "2026-02-05T10:00:01Z",
#       "end_time": "2026-02-05T10:00:45Z"
#     }
#   ]
# }
```

## Step 6: Use the Web UI

Open the web UI in your browser:

```bash
# Open in browser (macOS)
open http://localhost:8080

# Linux
xdg-open http://localhost:8080

# Or just navigate to: http://localhost:8080
```

The web UI provides:
- **Task List**: View all running and completed tasks
- **Run Details**: Click a run to see detailed logs
- **Live Streaming**: Logs update in real-time
- **Message Bus**: View cross-task communication
- **Run Tree**: Visualize parent-child task relationships

Screenshot: [The web UI shows a list of runs with status indicators, timestamps, and navigation to detailed log views]

## Step 7: Try Different Agents

### Using Claude Agent

First, add Claude configuration:

```yaml
# Add to config.yaml agents section
agents:
  claude:
    type: claude
    token_file: ~/.conductor/tokens/claude.token
    timeout: 300
```

Add your Claude token:

```bash
echo "your-claude-token" > ~/.conductor/tokens/claude.token
chmod 600 ~/.conductor/tokens/claude.token
```

Restart the server and run a task:

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "claude",
    "prompt": "Explain what the Ralph Loop is in one paragraph",
    "working_dir": "/tmp"
  }'
```

### Supported Agents

- **codex**: OpenAI Codex
- **claude**: Anthropic Claude
- **gemini**: Google Gemini
- **perplexity**: Perplexity AI
- **xai**: xAI (Grok)

See [Configuration](configuration.md) for agent-specific setup.

## Step 8: Parent-Child Tasks

Conductor Loop supports hierarchical tasks where a parent task can spawn child tasks.

The agent can create child tasks by writing to the message bus:

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "codex",
    "prompt": "Create a Python project with multiple files, using child tasks for each file",
    "working_dir": "/tmp/myproject"
  }'
```

When an agent writes a task request to the message bus, the Ralph Loop automatically:
1. Detects the child task request
2. Spawns a new run-agent process
3. Monitors the child task
4. Restarts the parent if the child fails

View the task tree in the Web UI to see parent-child relationships.

## Step 9: Understanding the Ralph Loop

The **Ralph Loop** is Conductor Loop's automatic restart mechanism. When you run a task (not a job), the Ralph Loop:

1. Executes the agent
2. Monitors for child tasks
3. Waits for all children to complete
4. Restarts on failure (up to max_restarts)
5. Exits with the final status

Example with explicit restart settings:

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "codex",
    "prompt": "Run a flaky test that might fail",
    "working_dir": "/tmp",
    "max_restarts": 3
  }'
```

If the task fails, it will restart up to 3 times before giving up.

## Step 10: View the Message Bus

The message bus enables cross-task communication:

```bash
# View all messages
curl http://localhost:8080/api/v1/messages

# Response:
# {
#   "messages": [
#     {
#       "timestamp": "2026-02-05T10:00:01Z",
#       "run_id": "run_20260205_100001_abc123",
#       "type": "task_start",
#       "data": {"agent": "codex", "prompt": "..."}
#     },
#     ...
#   ]
# }

# Stream messages in real-time (SSE)
curl -N http://localhost:8080/api/v1/messages/stream
```

Tasks can write to the message bus for coordination:
- Requesting child tasks
- Reporting progress
- Sharing data between tasks
- Signaling completion

## Common Use Cases

### Use Case 1: Code Generation

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "codex",
    "prompt": "Create a REST API server in Go with /health and /users endpoints",
    "working_dir": "/tmp/myapi"
  }'
```

### Use Case 2: Code Review

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "claude",
    "prompt": "Review the code in ./src and provide feedback on best practices",
    "working_dir": "/path/to/project"
  }'
```

### Use Case 3: Multi-Step Workflow

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "codex",
    "prompt": "1. Clone github.com/example/repo 2. Run tests 3. Generate coverage report 4. Create a summary",
    "working_dir": "/tmp/workflow"
  }'
```

## Next Steps

Now that you've completed the quick start, explore:

- **[Configuration Guide](configuration.md)**: Learn all configuration options
- **[CLI Reference](cli-reference.md)**: Master the command-line interface
- **[API Reference](api-reference.md)**: Deep dive into the REST API
- **[Web UI Guide](web-ui.md)**: Explore the web interface features
- **[Troubleshooting](troubleshooting.md)**: Solve common issues

## Tips & Best practices

1. **Use meaningful working directories**: Each task should have its own directory
2. **Set appropriate timeouts**: Complex tasks may need longer timeouts
3. **Monitor the message bus**: Use it to understand task coordination
4. **Check logs regularly**: Logs contain valuable debugging information
5. **Use the Web UI**: It's easier than curl for monitoring tasks

## Getting Help

- Check the [FAQ](faq.md) for common questions
- Read the [Troubleshooting Guide](troubleshooting.md)
- Open an issue on [GitHub](https://github.com/jonnyzzz/conductor-loop/issues)
