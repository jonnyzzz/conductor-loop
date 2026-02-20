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

# Start in API-only mode (no task execution)
conductor --config config.yaml --disable-task-start

# Start with all options
conductor \
  --config /etc/conductor/config.yaml \
  --root /opt/conductor \
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

Manage tasks (placeholder - not yet implemented).

```bash
conductor task [flags]
```

**Note:** This command is reserved for future use. Currently returns:
```
task command not implemented yet
```

#### `conductor job`

Manage jobs (placeholder - not yet implemented).

```bash
conductor job [flags]
```

**Note:** This command is reserved for future use. Currently returns:
```
job command not implemented yet
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
