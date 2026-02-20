# CLI Reference

Complete command-line reference for Conductor Loop binaries: `conductor` and `run-agent`.

## `conductor` - Main CLI

The main server and task orchestration CLI.

### Usage

```bash
conductor [flags]
conductor [command] [flags]
```

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | "" | Path to config file |
| `--root` | string | "" | Root directory for run-agent |
| `--host` | string | "0.0.0.0" | HTTP listen host (overrides config) |
| `--port` | int | 8080 | HTTP listen port (overrides config) |
| `--disable-task-start` | bool | false | Disable task execution (API-only mode) |
| `--version` | bool | false | Print version and exit |
| `--help` | bool | false | Show help and exit |

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CONDUCTOR_CONFIG` | Path to config file | `/etc/conductor/config.yaml` |
| `CONDUCTOR_ROOT` | Root directory | `/opt/conductor` |
| `CONDUCTOR_DISABLE_TASK_START` | Disable task execution | `true`, `1`, `yes` |

### Commands

#### Default Command (Server Mode)

Start the Conductor server.

```bash
conductor [flags]
```

**Examples:**

```bash
# Start with config file
conductor --config config.yaml

# Start with environment variable
export CONDUCTOR_CONFIG=/etc/conductor/config.yaml
conductor

# Start with custom root directory
conductor --config config.yaml --root /opt/conductor

# Start on a custom port (overrides config file value)
conductor --config config.yaml --port 9090

# Start in API-only mode (no task execution)
conductor --config config.yaml --disable-task-start

# Start with all options
conductor \
  --config /etc/conductor/config.yaml \
  --root /opt/conductor \
  --host 127.0.0.1 \
  --port 8080 \
  --disable-task-start
```

**Behavior:**

1. Loads configuration from `--config` or `CONDUCTOR_CONFIG`
2. Starts REST API server on configured host:port
3. Serves static frontend files (if available)
4. Listens for task creation requests
5. Spawns run-agent processes for tasks
6. Provides SSE streaming for logs
7. Runs until interrupted (Ctrl+C)

**Output:**

```
conductor 2026/02/05 10:00:00 Starting Conductor Loop server
conductor 2026/02/05 10:00:00 Config loaded from: config.yaml
conductor 2026/02/05 10:00:00 Root directory: /Users/you/conductor-loop
conductor 2026/02/05 10:00:00 API listening on http://0.0.0.0:8080
conductor 2026/02/05 10:00:00 Task execution: enabled
```

#### `conductor task`

Manage tasks via the conductor server API.

```bash
conductor task <subcommand> [flags]
```

**Subcommands:**

##### `conductor task status <task-id>`

Get the status of a task.

```bash
conductor task status <task-id> [--server URL] [--project PROJECT] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:8080" | Conductor server URL |
| `--project` | string | "" | Project ID (optional filter) |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor task status task-20260220-140000-my-task --project my-project
```

**Output:**
```
Task:   my-project/task-20260220-140000-my-task
Status: success
Last activity: 2026-02-20T14:01:30Z

RUN ID                          STATUS   START TIME             END TIME               EXIT CODE
20260220-1400000000-abc12345    success  2026-02-20T14:00:00Z   2026-02-20T14:01:30Z   0
```

#### `conductor job`

Manage jobs via the conductor server API.

```bash
conductor job <subcommand> [flags]
```

**Subcommands:**

##### `conductor job submit`

Submit a new job to the conductor server.

```bash
conductor job submit --project PROJECT --task TASK --agent AGENT --prompt PROMPT [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:8080" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--task` | string | "" | Task ID (required) |
| `--agent` | string | "" | Agent type, e.g. claude (required) |
| `--prompt` | string | "" | Task prompt (required) |
| `--project-root` | string | "" | Working directory for the task |
| `--attach-mode` | string | "create" | Attach mode: create, attach, or resume |
| `--wait` | bool | false | Wait for task completion by polling |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor job submit \
  --project my-project \
  --task task-20260220-140000-hello \
  --agent claude \
  --prompt "Write hello world" \
  --wait
```

##### `conductor job list`

List tasks on the conductor server.

```bash
conductor job list [--server URL] [--project PROJECT] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:8080" | Conductor server URL |
| `--project` | string | "" | Filter by project ID |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor job list --project my-project
```

**Output:**
```
PROJECT     TASK                              STATUS   LAST ACTIVITY
my-project  task-20260220-140000-hello        success  2026-02-20T14:01:30Z
my-project  task-20260220-150000-analysis     running  2026-02-20T15:02:00Z
```

#### `conductor version`

Print version information.

```bash
conductor version
```

**Output:**
```
dev
```

**Alternative:**
```bash
conductor --version
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (config load failed, server failed, etc.) |

### Common Usage Patterns

#### Development

```bash
# Local development with live reload
conductor --config config.yaml --root $(pwd)
```

#### Production

```bash
# Run as systemd service
ExecStart=/usr/local/bin/conductor \
  --config /etc/conductor/config.yaml \
  --root /var/lib/conductor
```

#### Docker

```bash
# Docker container
docker run -d \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/runs:/data/runs \
  conductor:latest \
  --config /app/config.yaml \
  --root /data
```

#### Testing/Development Mode

```bash
# API-only mode (no task execution)
conductor --config config.yaml --disable-task-start
```

Use this mode for:
- Testing the API without executing tasks
- Frontend development
- Configuration validation
- Performance testing

---

## `run-agent` - Agent Runner CLI

Low-level agent execution binary. Usually called by the conductor server, not directly by users.

### Usage

```bash
run-agent <command> [flags]
```

### Commands

#### `run-agent task`

Run a task with the Ralph Loop (automatic restart).

```bash
run-agent task --project <id> --task <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--project` | string | Project ID |
| `--task` | string | Task ID |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | "" | Root directory |
| `--config` | string | "" | Config file path |
| `--agent` | string | "" | Agent type |
| `--prompt` | string | "" | Prompt override |
| `--cwd` | string | "" | Working directory |
| `--message-bus` | string | "" | Message bus path |
| `--max-restarts` | int | 0 | Max restarts (0 = infinite) |
| `--child-wait-timeout` | duration | 0 | Child wait timeout |
| `--child-poll-interval` | duration | 0 | Child poll interval |
| `--restart-delay` | duration | 1s | Delay between restarts |

**Examples:**

```bash
# Basic task execution
run-agent task \
  --project proj_abc123 \
  --task task-20260220-140000-hello-world \
  --root /data \
  --config config.yaml \
  --agent codex \
  --prompt "Print hello world" \
  --cwd /tmp

# Task with restart limits
run-agent task \
  --project proj_abc123 \
  --task task-20260220-141500-flaky-test \
  --root /data \
  --config config.yaml \
  --agent codex \
  --prompt "Flaky test" \
  --max-restarts 3 \
  --restart-delay 5s

# Task with child monitoring
run-agent task \
  --project proj_abc123 \
  --task task-20260220-143000-child-monitor \
  --root /data \
  --config config.yaml \
  --agent codex \
  --message-bus /data/message-bus.jsonl \
  --child-wait-timeout 300s \
  --child-poll-interval 1s
```

**Behavior (Ralph Loop):**

1. Start agent execution
2. Monitor agent output
3. Detect child task requests in message bus
4. Wait for child tasks to complete
5. Check exit status
6. If failed and max-restarts not reached: restart (go to 1)
7. Exit with final status

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Failed (after all restarts) |
| 2 | Configuration error |

##### `run-agent task resume`

Resume a stopped or failed task from its existing task directory.

```bash
run-agent task resume --project <id> --task <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--project` | string | Project ID |
| `--task` | string | Task ID (must already have a task directory with TASK.md) |

**Optional Flags:** Same as `run-agent task`, except `--max-restarts` defaults to 3.

**Example:**

```bash
run-agent task resume \
  --project my-project \
  --task task-20260220-140000-hello-world \
  --root /data \
  --config config.yaml
```

**Behavior:**
- Validates that `<root>/<project>/<task>/TASK.md` exists
- Resumes Ralph Loop execution without re-creating the task directory
- Useful for re-running a task that previously stopped or failed

#### `run-agent job`

Run a single agent job (no restart logic).

```bash
run-agent job --project <id> --task <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--project` | string | Project ID |
| `--task` | string | Task ID |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | "" | Root directory |
| `--config` | string | "" | Config file path |
| `--agent` | string | "" | Agent type |
| `--prompt` | string | "" | Prompt text |
| `--prompt-file` | string | "" | Prompt file path |
| `--cwd` | string | "" | Working directory |
| `--message-bus` | string | "" | Message bus path |
| `--parent-run-id` | string | "" | Parent run ID |
| `--previous-run-id` | string | "" | Previous run ID |

**Examples:**

```bash
# Basic job execution
run-agent job \
  --project proj_abc123 \
  --task task-20260220-150000-explain-ralph-loop \
  --root /data \
  --config config.yaml \
  --agent claude \
  --prompt "Explain the Ralph Loop"

# Job with prompt from file
run-agent job \
  --project proj_abc123 \
  --task task-20260220-151500-file-processor \
  --root /data \
  --config config.yaml \
  --agent codex \
  --prompt-file /tmp/prompt.txt \
  --cwd /tmp/workspace

# Child job
run-agent job \
  --project proj_abc123 \
  --task task-20260220-152000-child-task \
  --root /data \
  --config config.yaml \
  --agent codex \
  --prompt "Child task" \
  --parent-run-id run_001 \
  --message-bus /data/message-bus.jsonl
```

**Behavior:**

1. Load configuration
2. Initialize agent
3. Execute prompt
4. Stream output to logs
5. Write to message bus (if specified)
6. Exit with agent status

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Agent failed |
| 2 | Configuration error |

#### `run-agent serve`

Start a read-only HTTP server exposing the runs API and web UI. Task execution is disabled.
Useful for monitoring runs without a running conductor server.

```bash
run-agent serve --root ./runs
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--host` | string | "127.0.0.1" | HTTP server host |
| `--port` | int | 14355 | HTTP server port |
| `--root` | string | "" | Root directory |
| `--config` | string | "" | Config file path |

**Examples:**

```bash
# Start on default port 14355
run-agent serve --root ./runs

# Start on custom port
run-agent serve --root ./runs --port 8090

# Listen on all interfaces
run-agent serve --root ./runs --host 0.0.0.0 --port 14355
```

**Note:** When opening the web UI via `file://`, the `API_BASE` in `web/src/app.js` defaults to
`http://localhost:8080` (conductor's port). If using `run-agent serve`, either open the UI
through the server at `http://127.0.0.1:14355/ui/` or update `API_BASE` manually.

#### `run-agent stop`

Stop a running task by sending SIGTERM to its process group.

```bash
run-agent stop [flags]
```

**Flags (choose one approach):**

| Flag | Type | Description |
|------|------|-------------|
| `--run-dir` | string | Path to run directory (alternative to --root/--project/--task) |
| `--root` | string | Run-agent root directory |
| `--project` | string | Project ID |
| `--task` | string | Task ID |
| `--run` | string | Run ID (optional, defaults to latest running run) |
| `--force` | bool | Send SIGKILL if process does not stop within 30s timeout |

**Examples:**

```bash
# Stop by project/task (stops the latest running run)
run-agent stop --root ./runs --project my-project --task task-20260220-140000-hello

# Stop by run directory
run-agent stop --run-dir ./runs/my-project/task-20260220-140000-hello/runs/20260220-1400000000-abc12345

# Force kill if SIGTERM doesn't work
run-agent stop --root ./runs --project my-project --task task-20260220-140000-hello --force
```

**Behavior:**
1. Resolves run directory (by `--run-dir` or latest running run under `--project/--task`)
2. Reads `run-info.yaml` to get PGID/PID
3. Sends SIGTERM to process group
4. Polls for up to 30s for the process to exit
5. If `--force` and still running after 30s: sends SIGKILL

---

#### `run-agent gc`

Clean up old run directories to reclaim disk space.

```bash
run-agent gc [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | "./runs" | Root runs directory |
| `--older-than` | duration | 168h (7 days) | Delete runs older than this |
| `--dry-run` | bool | false | Print what would be deleted without deleting |
| `--project` | string | "" | Limit gc to a specific project |
| `--keep-failed` | bool | false | Preserve runs with non-zero exit codes |

**Examples:**

```bash
# Dry run to see what would be deleted
run-agent gc --root ./runs --dry-run

# Delete runs older than 7 days
run-agent gc --root ./runs --older-than 168h

# Clean only a specific project, keep failed runs
run-agent gc --root ./runs --project my-project --keep-failed

# Aggressive cleanup (1 day, including failed runs)
run-agent gc --root ./runs --older-than 24h
```

**Behavior:**
- Skips runs that are currently `running`
- Reports freed disk space in MB
- Only deletes completed or failed runs older than the cutoff

---

#### `run-agent validate`

Validate conductor configuration and agent CLI availability.

```bash
run-agent validate [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | "" | Config file path (auto-discovered if not set) |
| `--root` | string | "" | Root directory to validate (checks writability) |
| `--agent` | string | "" | Validate only this agent (default: all) |
| `--check-network` | bool | false | Run network connectivity test for REST agents |

**Example:**

```bash
# Validate config and all agents
run-agent validate --config config.yaml

# Validate a specific agent
run-agent validate --config config.yaml --agent claude

# Validate config and root directory
run-agent validate --config config.yaml --root ./runs
```

**Output:**
```
Conductor Loop Configuration Validator

Config: config.yaml

Agents:
  ✓ claude      2.1.49     (CLI found, token: ANTHROPIC_API_KEY set)
  ✓ codex       0.104.0    (CLI found, token: OPENAI_API_KEY set)
  ✗ gemini                 (CLI "gemini" not found in PATH, token: GEMINI_API_KEY not set)

Validation: 2 OK, 1 WARNING
```

---

#### `run-agent bus`

Read from and post to message bus files.

```bash
run-agent bus <subcommand> [flags]
```

**Subcommands:**

##### `run-agent bus post`

Post a message to a message bus file.

```bash
run-agent bus post [--bus PATH] [--type TYPE] [--body BODY] [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--bus` | string | "" | Path to message bus file (uses `$MESSAGE_BUS` env var if not set) |
| `--type` | string | "INFO" | Message type |
| `--project` | string | "" | Project ID |
| `--task` | string | "" | Task ID |
| `--run` | string | "" | Run ID |
| `--body` | string | "" | Message body (reads stdin if not provided and stdin is a pipe) |

**Examples:**

```bash
# Post a message
run-agent bus post --bus ./task-bus.md --type INFO --body "Processing started"

# Post from stdin
echo "Done!" | run-agent bus post --bus ./task-bus.md --type DONE
```

##### `run-agent bus read`

Read messages from a message bus file.

```bash
run-agent bus read [--bus PATH] [--tail N] [--follow]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--bus` | string | "" | Path to message bus file (uses `$MESSAGE_BUS` env var if not set) |
| `--tail` | int | 20 | Print last N messages |
| `--follow` | bool | false | Watch for new messages (Ctrl-C to exit) |

**Examples:**

```bash
# Read last 20 messages
run-agent bus read --bus ./task-bus.md

# Follow new messages
run-agent bus read --bus ./task-bus.md --follow
```

---

#### `run-agent version`

Print version information.

```bash
run-agent --version
```

**Output:**
```
dev
```

### When to Use `run-agent` Directly

**Normal Use Case:** Don't call `run-agent` directly. The `conductor` server manages it.

**Advanced Use Cases:**

1. **Testing agent execution**
   ```bash
   run-agent job \
     --project test \
     --task task-20260220-160000-agent-test \
     --config config.yaml \
     --agent codex \
     --prompt "test prompt"
   ```

2. **Debugging the Ralph Loop**
   ```bash
   run-agent task \
     --project debug \
     --task task-20260220-161500-ralph-debug \
     --config config.yaml \
     --agent codex \
     --prompt "debug task" \
     --max-restarts 1
   ```

3. **Manual task execution**
   ```bash
   # When conductor server is not running
   run-agent job \
     --project manual \
     --task task-20260220-163000-manual-run \
     --config config.yaml \
     --agent codex \
     --prompt "manual task"
   ```

4. **Scripting/Automation**
   ```bash
   #!/bin/bash
   # batch-process.sh
   for file in *.txt; do
     run-agent job \
       --project batch \
       --task "task-$(date +%Y%m%d-%H%M%S)-${file%.*}" \
       --config config.yaml \
       --agent codex \
       --prompt "Process $file"
   done
   ```

### Common Flags Explained

#### `--root` (Root Directory)

The root directory where run data is stored.

```
--root /data
```

Structure:
```
/data/
├── runs/
│   ├── run_20260205_100001_abc123/
│   │   ├── output.log
│   │   └── metadata.json
│   └── ...
└── message-bus.jsonl
```

#### `--config` (Config File)

Path to the configuration file.

```
--config /etc/conductor/config.yaml
```

See [Configuration Reference](configuration.md) for config file format.

#### `--agent` (Agent Type)

Agent to use for execution. Must be configured in config.yaml.

```
--agent codex
--agent claude
--agent gemini
```

#### `--prompt` vs `--prompt-file`

**Direct prompt:**
```bash
--prompt "Write a hello world script"
```

**Prompt from file:**
```bash
--prompt-file /tmp/prompt.txt
```

Use `--prompt-file` for:
- Long prompts
- Multi-line prompts
- Prompts with special characters
- Reusable prompts

#### `--message-bus` (Message Bus Path)

Path to the message bus file for cross-task communication.

```
--message-bus /data/message-bus.jsonl
```

The agent can:
- Read messages from other tasks
- Write messages for coordination
- Request child tasks
- Report progress

#### `--max-restarts` (Max Restarts)

Maximum number of restarts on failure.

```
--max-restarts 3    # Restart up to 3 times
--max-restarts 0    # Restart indefinitely
```

Default: 0 (infinite restarts)

Use cases:
- `0`: Long-running tasks that should never give up
- `3`: Flaky tasks that might succeed after a few tries
- `1`: Tasks that should retry once

#### `--restart-delay` (Restart Delay)

Delay between restart attempts.

```
--restart-delay 1s     # 1 second (default)
--restart-delay 10s    # 10 seconds
--restart-delay 1m     # 1 minute
```

Use longer delays for:
- Rate-limited APIs
- Resource contention
- Network issues

## Examples by Use Case

### Development: Start Server

```bash
conductor --config config.yaml --root $(pwd)
```

### Production: Systemd Service

```bash
# /etc/systemd/system/conductor.service
[Service]
ExecStart=/usr/local/bin/conductor \
  --config /etc/conductor/config.yaml \
  --root /var/lib/conductor
```

### Testing: API-Only Mode

```bash
conductor --config config.yaml --disable-task-start
```

### Debugging: Manual Agent Run

```bash
run-agent job \
  --project debug \
  --task task-20260220-165000-debug-test \
  --config config.yaml \
  --agent codex \
  --prompt "Debug test"
```

### Automation: Batch Processing

```bash
#!/bin/bash
for i in {1..10}; do
  run-agent job \
    --project batch \
    --task "task-$(date +%Y%m%d-%H%M%S)-item-${i}" \
    --config config.yaml \
    --agent codex \
    --prompt "Process item $i"
done
```

## Next Steps

- [API Reference](api-reference.md) - REST API for task creation
- [Configuration](configuration.md) - Configure agents and settings
- [Quick Start](quick-start.md) - Try it out
- [Troubleshooting](troubleshooting.md) - Solve common issues
