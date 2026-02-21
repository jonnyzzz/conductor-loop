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
  port: 14355
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
# conductor 2026/02/05 10:00:00 API listening on http://0.0.0.0:14355
# conductor 2026/02/05 10:00:00 Root directory: /Users/you/conductor-loop
```

Leave this terminal running and open a new terminal for the next steps.

## Step 3: Test the Server

Verify the server is running:

```bash
# Check health
curl http://localhost:14355/api/v1/health

# Response:
# {"status":"ok","version":"dev"}

# Check version
curl http://localhost:14355/api/v1/version

# Response:
# {"version":"dev"}
```

## Step 4: Run Your First Task

### Simple "Hello World" Task

Create a task that prints "Hello World":

```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-20260205-100000-hello-world",
    "agent_type": "codex",
    "prompt": "Write a Python script that prints Hello World",
    "project_root": "/tmp"
  }'

# Response:
# {
#   "project_id": "my-project",
#   "task_id": "task-20260205-100000-hello-world",
#   "status": "created"
# }
```

Use the `task_id` from the response to monitor the task.

### Check Task Status

```bash
curl "http://localhost:14355/api/projects/my-project/tasks/task-20260205-100000-hello-world"

# Response includes the list of runs with their statuses.
```

Status values:
- `created`: Task created, not started yet
- `running`: Task is executing
- `success`: Task completed successfully
- `failed`: Task failed
- `restarted`: Task was restarted by Ralph Loop

### Check Running Task Runs (Latest Per Task)

Use `run-agent status` to scan each task's latest run and show only active (`status: running`) runs:

```bash
run-agent status --root /Users/you/run-agent --project my-project --status running
```

For automation-friendly output, add `--concise`:

```bash
run-agent status --root /Users/you/run-agent --project my-project --status running --concise
```

When no rows match, the command prints an explicit no-match message (instead of silent empty output).

### View Task Logs

Stream the live logs for all runs of a task (SSE):

```bash
curl -N "http://localhost:14355/api/projects/my-project/tasks/task-20260205-100000-hello-world/runs/stream"

# Output (Server-Sent Events):
# data: Starting task...
# data: Agent: codex
# data: Executing prompt...
# ...
```

Or stream a specific run file:

```bash
RUN_ID="20260205-1000000000-12345"
curl -N "http://localhost:14355/api/projects/my-project/tasks/task-20260205-100000-hello-world/runs/$RUN_ID/stream?name=output.md"
```

## Step 5: List All Tasks

See all tasks in a project:

```bash
curl http://localhost:14355/api/projects/my-project/tasks
```

## Step 6: Use the Web UI

Open the web UI in your browser:

```bash
# Open in browser (macOS)
open http://localhost:14355/ui/

# Linux
xdg-open http://localhost:14355/ui/

# Or just navigate to: http://localhost:14355/ui/
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
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-20260205-110000-ralph-explain",
    "agent_type": "claude",
    "prompt": "Explain what the Ralph Loop is in one paragraph",
    "project_root": "/tmp"
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
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-20260205-120000-build-project",
    "agent_type": "codex",
    "prompt": "Create a Python project with multiple files, using child tasks for each file",
    "project_root": "/tmp/myproject"
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

Example with explicit restart settings via `run-agent task --max-restarts 3`:

```bash
./bin/run-agent task \
  --project my-project \
  --task task-20260205-130000-flaky-test \
  --root ./runs \
  --config config.yaml \
  --agent codex \
  --prompt "Run a flaky test that might fail" \
  --max-restarts 3
```

If the task fails, it will restart up to 3 times before giving up.

## Step 10: Work with the Message Bus

The bus has two scopes:
- **Task scope**: `TASK-MESSAGE-BUS.md` for one task
- **Project scope**: `PROJECT-MESSAGE-BUS.md` for cross-task coordination

For local runs, use `run-agent bus` (direct file access):

```bash
# Follow task-scoped messages while a task is running
./bin/run-agent bus read \
  --project my-project \
  --task task-20260205-100000-hello-world \
  --root ./runs \
  --follow

# Read recent project-scoped messages
./bin/run-agent bus read --project my-project --root ./runs --tail 50

# Post a task-scoped progress update
./bin/run-agent bus post \
  --project my-project \
  --task task-20260205-100000-hello-world \
  --root ./runs \
  --type PROGRESS \
  --body "started auth module refactor"

# Post a project-scoped decision
./bin/run-agent bus post \
  --project my-project \
  --root ./runs \
  --type DECISION \
  --body "standardizing on OAuth2"
```

### Read/Post/Follow Patterns (`run-agent bus`)

Use these three patterns in day-to-day task operations:

```bash
# 1) Read recent context before touching a task
./bin/run-agent bus read \
  --project my-project \
  --task task-20260205-100000-hello-world \
  --root ./runs \
  --tail 30

# 2) Follow live events while the task is executing
./bin/run-agent bus read \
  --project my-project \
  --task task-20260205-100000-hello-world \
  --root ./runs \
  --follow

# 3) Post explicit lifecycle updates
./bin/run-agent bus post \
  --project my-project \
  --task task-20260205-100000-hello-world \
  --root ./runs \
  --type PROGRESS \
  --body "starting integration test batch"

./bin/run-agent bus post \
  --project my-project \
  --task task-20260205-100000-hello-world \
  --root ./runs \
  --type FACT \
  --body "integration tests passed: 48/48"
```

You can also read and post through the server API:

```bash
# Read all project-level messages
curl "http://localhost:14355/api/v1/messages?project_id=my-project"

# Stream task-level messages
curl -N "http://localhost:14355/api/projects/my-project/tasks/task-20260205-100000-hello-world/messages/stream"

# Post to task-level message bus
curl -X POST \
  "http://localhost:14355/api/projects/my-project/tasks/task-20260205-100000-hello-world/messages" \
  -H "Content-Type: application/json" \
  -d '{"type": "PROGRESS", "body": "starting test run"}'
```

### Recommended Message Types

- `PROGRESS`: start/end of major steps
- `FACT`: concrete outcomes (tests passed, files changed, run IDs)
- `DECISION`: chosen approach and short rationale
- `ERROR`: blocker and attempted remediation
- `QUESTION`: explicit request for input
- `INFO`: neutral status updates

`RUN_START`, `RUN_STOP`, and `RUN_CRASH` are emitted by the runner automatically.

### Typical Local Orchestration Workflow

1. Start a root task (`run-agent task` or `conductor job submit`).
2. Follow task messages with `run-agent bus read --project ... --task ... --follow`.
3. Post `PROGRESS` before major steps and `FACT` after concrete outcomes.
4. Post `DECISION` for strategy changes and `ERROR` for blockers.
5. Use project-scoped messages to communicate cross-task coordination points.

The Web UI Message Bus panels read and write these same task/project bus files via API. If UI compose is unavailable or feed context looks stale after switching scope/task, use `run-agent bus read/post` as the source of truth.

## Common Use Cases

### Use Case 1: Code Generation

```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-20260205-140000-code-gen",
    "agent_type": "codex",
    "prompt": "Create a REST API server in Go with /health and /users endpoints",
    "project_root": "/tmp/myapi"
  }'
```

### Use Case 2: Code Review

```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-20260205-141500-code-review",
    "agent_type": "claude",
    "prompt": "Review the code in ./src and provide feedback on best practices",
    "project_root": "/path/to/project"
  }'
```

### Use Case 3: Multi-Step Workflow

```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-20260205-143000-workflow",
    "agent_type": "codex",
    "prompt": "1. Clone github.com/example/repo 2. Run tests 3. Generate coverage report 4. Create a summary",
    "project_root": "/tmp/workflow"
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
