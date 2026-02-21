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
| `--port` | int | 14355 | HTTP listen port (overrides config) |
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
  --port 14355 \
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
conductor 2026/02/05 10:00:00 API listening on http://0.0.0.0:14355
conductor 2026/02/05 10:00:00 Task execution: enabled
```

#### `conductor status`

Show the conductor server status (active runs, uptime, configured agents).

```bash
conductor status [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor status
conductor status --server http://prod-conductor:14355
conductor status --json
```

**Output:**
```
Version:            dev
Uptime:             2h 3m 4s
Active Runs:        3
Configured Agents:  claude, codex
```

---

#### `conductor task`

Manage tasks via the conductor server API.

```bash
conductor task <subcommand> [flags]
```

**Subcommands:**

##### `conductor task stop <task-id>`

Stop all running runs of a task (writes a DONE file and sends SIGTERM to the task's processes).

```bash
conductor task stop <task-id> [--server URL] [--project PROJECT] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (optional filter) |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor task stop task-20260220-140000-my-task
conductor task stop task-20260220-140000-my-task --project my-project
```

**Output:**
```
Task task-20260220-140000-my-task: stopped 2 run(s)
```

---

##### `conductor task status <task-id>`

Get the status of a task.

```bash
conductor task status <task-id> [--server URL] [--project PROJECT] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
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

##### `conductor task list`

List tasks in a project, optionally filtered by status.

```bash
conductor task list --project <id> [--status <filter>] [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--status` | string | "" | Filter by status: `running`, `active`, `done`, `failed` |
| `--json` | bool | false | Output raw JSON response |

**Status filter values:**
- `running` / `active` — tasks with at least one currently-running run
- `done` — tasks where the `DONE` file exists (Ralph loop completed)
- `failed` — tasks with no running runs, no DONE file, and at least one failed run

**Example:**
```bash
conductor task list --project my-project
conductor task list --project my-project --status running
conductor task list --project my-project --status done
conductor task list --project my-project --json
```

**Output:**
```
TASK ID                           STATUS   RUNS  LAST ACTIVITY
task-20260220-140000-hello        done     3     2026-02-20 14:01
task-20260220-150000-analysis     running  1     2026-02-20 15:02
```

When the result is paginated, a footer line is shown:
```
(showing 20 of 42 tasks; use --limit to see more)
```

---

##### `conductor task delete <task-id>`

Delete a task and all its runs via the conductor server API.

```bash
conductor task delete <task-id> --project <id> [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor task delete task-20260220-140000-hello --project my-project
```

**Output:**
```
Task task-20260220-140000-hello deleted.
```

**Error cases:**
- `409 Conflict`: task has running runs — stop them first with `conductor task stop`
- `404 Not Found`: task does not exist in the specified project

---

##### `conductor task logs <task-id>`

Stream the output of a task's agent run via the conductor server.

```bash
conductor task logs <task-id> --project PROJECT [--run RUN] [--follow] [--tail N] [--server URL]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--run` | string | "" | Specific run ID to stream (default: latest active or most recent) |
| `--follow` | bool | false | Keep streaming; reconnect if connection drops |
| `--tail` | int | 0 | Output only the last N lines (0 = all) |

When `--run` is omitted, the command automatically selects the currently-running run, or the
most recent run if the task is not running.

When `--follow` is true, the command reconnects with exponential backoff (2s → 30s) if the
connection drops, streaming until the run completes or you press Ctrl-C.

**Example:**
```bash
# Stream all output for the most recent run of a task
conductor task logs task-20260221-120000-my-task --project my-project

# Follow a currently-running task (stay connected)
conductor task logs task-20260221-120000-my-task --project my-project --follow

# Show only the last 50 lines
conductor task logs task-20260221-120000-my-task --project my-project --tail 50

# Stream a specific run
conductor task logs task-20260221-120000-my-task --project my-project \
  --run 20260221-1200000000-12345-1
```

##### `conductor task runs <task-id>`

List all runs for a specific task with status, exit code, duration, and agent version.

```bash
conductor task runs <task-id> --project PROJECT [--limit N] [--json] [--server URL]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--limit` | int | 50 | Maximum number of runs to show |
| `--json` | bool | false | Output as JSON |

Shows all runs for the task, newest first. Useful for tracking Ralph loop restarts and debugging.

**Example:**
```bash
# List all runs for a task
conductor task runs task-20260221-120000-my-task --project my-project

# List only last 10 runs
conductor task runs task-20260221-120000-my-task --project my-project --limit 10

# JSON output for scripting
conductor task runs task-20260221-120000-my-task --project my-project --json
```

**Sample output:**
```
RUN ID                          AGENT    STATUS      EXIT  DURATION  STARTED               ERROR
20260221-120100-12346           claude   completed      0  5m23s     2026-02-21 12:01:00
20260221-115900-12345           claude   failed         1  0m12s     2026-02-21 11:59:00   exit code 1: general failure
```

##### `conductor task resume <task-id>`

Resume an exhausted task by removing its DONE file (allowing it to be re-queued).

```bash
conductor task resume <task-id> --project PROJECT [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--json` | bool | false | Output raw JSON response |

**When to use**: After a task has exhausted its restart limit (the `DONE` file was written with an error state), use `conductor task resume` to clear the done marker so the task can be submitted again. The task directory and all its run history are preserved.

**Example:**
```bash
# Resume a task that failed after max restarts
conductor task resume task-20260221-120000-my-task --project my-project

# Resume and verify with JSON output
conductor task resume task-20260221-120000-my-task --project my-project --json
```

**Output:**
```
Task my-project/task-20260221-120000-my-task resumed (DONE file removed)
```

**Note:** This is the server-based equivalent of `run-agent resume`. After resuming, submit a new job to the task to restart execution.

---

#### `conductor job`

Manage jobs via the conductor server API.

```bash
conductor job <subcommand> [flags]
```

**Subcommands:**

##### `conductor job submit`

Submit a new job to the conductor server.

```bash
conductor job submit --project PROJECT --agent AGENT (--prompt PROMPT | --prompt-file PATH) [--task TASK] [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--task` | string | "" | Task ID (optional; auto-generated as `task-YYYYMMDD-HHMMSS-xxxxxx` if omitted) |
| `--agent` | string | "" | Agent type, e.g. claude (required) |
| `--prompt` | string | "" | Task prompt (mutually exclusive with `--prompt-file`) |
| `--prompt-file` | string | "" | Path to file containing task prompt (mutually exclusive with `--prompt`) |
| `--project-root` | string | "" | Working directory for the task |
| `--attach-mode` | string | "create" | Attach mode: create, attach, or resume |
| `--wait` | bool | false | Wait for task completion by polling |
| `--follow` | bool | false | Stream task output after submission (implies `--wait`); reconnects on connection drop |
| `--json` | bool | false | Output raw JSON response |

Exactly one of `--prompt` or `--prompt-file` must be provided. Errors are returned if both are set, neither is set, the file is not found, or the file is empty.

When `--task` is omitted, a task ID is auto-generated in the format `task-YYYYMMDD-HHMMSS-xxxxxx` (6-char random hex suffix). The assigned task ID is printed to stdout on success.

When `--follow` is set, after submitting the job the command streams the task's agent output to stdout via the conductor server's SSE endpoint, blocking until the task completes. If both `--follow` and `--wait` are set, `--follow` takes precedence (streaming output is strictly better than silent polling). The command waits up to 30 seconds for the first run to start.

**Example:**
```bash
# Auto-generate task ID (recommended for one-off jobs)
conductor job submit \
  --project my-project \
  --agent claude \
  --prompt "Write hello world" \
  --wait

# Explicit task ID (useful when you need a predictable, reusable task)
conductor job submit \
  --project my-project \
  --task task-20260220-140000-hello \
  --agent claude \
  --prompt "Write hello world" \
  --wait

# Submit with prompt from file (useful for long or multi-line prompts)
conductor job submit \
  --project my-project \
  --agent claude \
  --prompt-file /path/to/prompt.md \
  --wait

# Stream output in real-time (no need to separately run conductor task logs)
conductor job submit \
  --project my-project \
  --agent claude \
  --prompt-file /path/to/prompt.md \
  --follow
```

##### `conductor job list`

List tasks on the conductor server.

```bash
conductor job list [--server URL] [--project PROJECT] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
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

#### `conductor project`

Manage projects via the conductor server API.

```bash
conductor project <subcommand> [flags]
```

**Subcommands:**

##### `conductor project list`

List all projects known to the conductor server.

```bash
conductor project list [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor project list
conductor project list --json
```

**Output:**
```
PROJECT        TASKS  LAST ACTIVITY
my-project     5      2026-02-20 15:02
other-project  2      2026-02-19 10:30
```

---

##### `conductor project stats`

Show detailed statistics for a project.

```bash
conductor project stats --project <id> [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--json` | bool | false | Output raw JSON response |

**Example:**
```bash
conductor project stats --project my-project
conductor project stats --project my-project --json
```

**Output:**
```
Project:            my-project
Tasks:              5
Runs (total):       12
  Running:          1
  Completed:        9
  Failed:           2
  Crashed:          0
Message bus files:  2
Message bus size:   4.50 KB
```

---

##### `conductor project gc`

Garbage collect old completed/failed runs for a project via the conductor server API.

```
conductor project gc --project PROJECT [--older-than DURATION] [--dry-run] [--keep-failed] [--server URL] [--json]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | (required) | Project ID |
| `--older-than` | `168h` | Delete runs older than this duration (e.g. `24h`, `168h` for 7 days) |
| `--dry-run` | false | Show what would be deleted without actually deleting |
| `--keep-failed` | false | Keep runs that exited with a non-zero exit code |
| `--server` | `http://localhost:14355` | Conductor server URL |
| `--json` | false | Output as JSON |

**Examples:**
```bash
# Dry run - see what would be deleted
conductor project gc --project my-project --dry-run

# Delete runs older than 7 days
conductor project gc --project my-project --older-than 168h

# Delete runs older than 24h, keep failed runs
conductor project gc --project my-project --older-than 24h --keep-failed

# JSON output
conductor project gc --project my-project --dry-run --json
```

---

##### `conductor project delete`

Delete an entire project — all tasks and all their runs — via the conductor server API.

```bash
conductor project delete <project-id> [--force] [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--force` | bool | false | Stop running tasks and delete anyway |
| `--json` | bool | false | Output raw JSON response |

**Behavior:**
- If any tasks have running runs, the command returns an error unless `--force` is specified.
- With `--force`, running tasks are stopped (SIGTERM) before deletion.
- Returns the number of tasks deleted and disk space freed.

**Example:**
```bash
# Delete a project (fails if tasks are running)
conductor project delete my-project

# Force-delete even with running tasks
conductor project delete my-project --force

# JSON output
conductor project delete my-project --json
```

**Output:**
```
Project my-project deleted (5 tasks, 42.00 MB freed).
```

**Error cases:**
- `409 Conflict`: project has running tasks — stop them first or use `--force`
- `404 Not Found`: project does not exist

---

#### `conductor bus`

Read messages from the project or task message bus via the conductor server API.

##### `conductor bus read`

Read messages from the message bus (project-level or task-level).

```bash
conductor bus read --project <project> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | string | "" | Project ID (required) |
| `--task` | string | "" | Task ID (optional; reads task-level bus if set, project-level otherwise) |
| `--server` | string | `http://localhost:14355` | Conductor server URL |
| `--tail` | int | 0 | Show last N messages (0 = all) |
| `--follow` | bool | false | Stream new messages via SSE (keep watching) |
| `--json` | bool | false | Output as raw JSON array |

**Examples:**

```bash
# Read all messages from a project bus
conductor bus read --project myproject

# Read the last 10 messages from a task bus
conductor bus read --project myproject --task task-20260221-070000-myfeature --tail 10

# Stream new messages as they arrive
conductor bus read --project myproject --task task-20260221-070000-myfeature --follow

# Output as JSON for scripting
conductor bus read --project myproject --json

# Use a different server
conductor bus read --project myproject --server http://conductor.example.com:14355
```

**Output (formatted text):**
```
[2026-02-21 07:00:00] RUN_START     run started
[2026-02-21 07:01:00] PROGRESS      Starting sub-agent for task X
[2026-02-21 07:02:00] FACT          Build passed
[2026-02-21 07:03:00] RUN_STOP      run completed
```

**Note:** This is the server-based equivalent of `run-agent bus read` (which requires local file access). Use `conductor bus read` when working with a remote conductor server.

##### `conductor bus post`

Post a message to the project or task message bus via the conductor server API.

```bash
conductor bus post --project PROJECT [--task TASK] [--type TYPE] [--body BODY] [--server URL]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | string | "" | Project ID (required) |
| `--task` | string | "" | Task ID (optional; posts to task-level bus if set) |
| `--type` | string | "INFO" | Message type |
| `--body` | string | "" | Message body (reads from stdin if not provided and stdin is a pipe) |
| `--server` | string | `http://localhost:14355` | Conductor server URL |

**Examples:**

```bash
# Post a message to the project bus
conductor bus post --project my-project --type PROGRESS --body "Build started"

# Post to a task bus
conductor bus post --project my-project --task task-20260221-120000-feat --type FACT --body "Tests passed"

# Post from stdin (useful in scripts)
echo "Deployment complete" | conductor bus post --project my-project --type FACT

# Use a remote server
conductor bus post --project my-project --type INFO --body "Hello" --server http://conductor.example.com:14355
```

**Output:**
```
msg_id: MSG-20260221-110000-abc123
```

**Note:** This is the server-based equivalent of `run-agent bus post` (which requires local file access). Use `conductor bus post` when working with a remote conductor server.

---

#### `conductor watch`

Watch tasks in a project until they reach a terminal state (completed, failed, done, error).

```bash
conductor watch --project PROJECT [--task TASK-ID ...] [--timeout DURATION] [--interval DURATION] [--server URL] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | "http://localhost:14355" | Conductor server URL |
| `--project` | string | "" | Project ID (required) |
| `--task` | stringArray | [] | Task ID(s) to watch (repeatable; default: watch all tasks in project) |
| `--timeout` | duration | 30m | Max wait time; exits with code 1 if timeout is reached |
| `--interval` | duration | 5s | Poll interval between status checks |
| `--json` | bool | false | Output final status as JSON when all tasks are done |

**Behavior:**
- Polls the conductor server at the specified interval
- Prints a status table on each poll showing task ID, status, and run count
- Exits with code 0 when **all** watched tasks reach a terminal state
- Exits with code 1 if the timeout is reached before all tasks complete
- If no `--task` flags are specified, watches **all tasks** in the project

**Examples:**
```bash
# Watch all tasks in a project (waits up to 30 minutes)
conductor watch --project my-project

# Watch specific tasks
conductor watch --project my-project \
  --task task-20260221-120000-feature-a \
  --task task-20260221-120001-feature-b

# Watch with a longer timeout and faster poll interval
conductor watch --project my-project --timeout 2h --interval 10s

# Wait for completion and output JSON summary
conductor watch --project my-project --json

# Use in CI scripts (exit code 1 if tasks don't complete in time)
conductor watch --project my-project --timeout 1h && echo "All done!" || echo "Timed out"
```

**Sample output:**
```
Watching all tasks in project "my-project"...
[2026-02-21 12:00:00] Poll #1
TASK ID                              STATUS    RUNS
task-20260221-120000-feature-a       running   2
task-20260221-120001-feature-b       running   1

[2026-02-21 12:00:05] Poll #2
TASK ID                              STATUS      RUNS
task-20260221-120000-feature-a       completed   3
task-20260221-120001-feature-b       running     2

[2026-02-21 12:00:10] Poll #3
TASK ID                              STATUS      RUNS
task-20260221-120000-feature-a       completed   3
task-20260221-120001-feature-b       completed   2

All tasks completed.
```

**Note:** This is the server-based equivalent of `run-agent watch`. Use it when working with a remote conductor server or when you need to wait for multiple tasks across a project.

---

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
| `--timeout` | duration | 0 | Maximum agent run duration per job (e.g. `30m`, `2h`); 0 means no limit |

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

##### `run-agent task delete`

Delete a task directory and all its runs from disk.

```bash
run-agent task delete --project <id> --task <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--project` | string | Project ID |
| `--task` | string | Task ID to delete |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | `$RUNS_DIR` or `./runs` | Root runs directory |
| `--force` | bool | false | Delete even if the task has running runs |

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | Task directory deleted successfully |
| 1 | Task not found, running runs present (without `--force`), or filesystem error |

**Examples:**

```bash
# Delete a completed task
run-agent task delete \
  --root ./runs \
  --project my-project \
  --task task-20260220-140000-hello

# Force-delete a task that has running runs
run-agent task delete \
  --root ./runs \
  --project my-project \
  --task task-20260220-150000-stuck \
  --force
```

**Behavior:**
- Validates that the task directory exists; exits with code 1 if not found.
- Without `--force`: scans `<task>/runs/` for any run with `status: running`. If found, exits with code 1 and an error message naming the running run.
- With `--force`: skips the running-run check and removes the directory unconditionally.
- Deletes the entire task directory including all run subdirectories, agent output, the task message bus, and the TASK.md file.
- Prints `Deleted task: <task-id>` on success.
- The REST API counterpart is `DELETE /api/projects/{p}/tasks/{t}`.

---

#### `run-agent resume`

Reset an exhausted task's restart counter and optionally retry it.

```bash
run-agent resume --project <id> --task <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--project` | string | Project ID |
| `--task` | string | Task ID |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | "./runs" | Root runs directory |
| `--agent` | string | "" | Agent type; if set, launches a new run after reset |
| `--prompt` | string | "" | Prompt text (used when `--agent` is set) |
| `--prompt-file` | string | "" | Prompt file path (used when `--agent` is set) |
| `--config` | string | "" | Config file path |

**Examples:**

```bash
# Reset an exhausted task (remove DONE file so the loop can re-run it)
run-agent resume \
  --root ./runs \
  --project my-project \
  --task task-20260220-140000-hello

# Reset and immediately launch a new run
run-agent resume \
  --root ./runs \
  --project my-project \
  --task task-20260220-140000-hello \
  --agent claude \
  --prompt-file /path/to/prompt.md
```

**Behavior:**
- Removes the `DONE` file from the task directory, resetting the restart counter so the Ralph Loop can run again.
- If `--agent` is provided, launches a new job run immediately after the reset using the supplied `--prompt` or `--prompt-file`.

**`run-agent resume` vs `run-agent task resume`:**

| Command | When to use |
|---------|-------------|
| `run-agent resume` | The task reached its `maxRestarts` limit and was marked DONE. Use this to clear the exhausted state and optionally kick off a fresh run. |
| `run-agent task resume` | The task's Ralph Loop process stopped or failed before the task was marked DONE (e.g. host restart). Use this to re-enter the loop from the existing task directory. |

---

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
| `--timeout` | duration | 0 | Maximum agent run duration (e.g. `30m`, `2h`); 0 means no limit |
| `--follow`, `-f` | bool | false | Stream output to stdout while job runs (pre-allocates run dir, blocks until job completes) |

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

#### `run-agent wrap`

Run an agent CLI command as-is while recording it as a tracked Conductor task/run.

This is intended for shell alias wrappers (for example `claude`, `codex`, `gemini`)
so normal console usage is persisted into `<root>/<project>/<task>/runs/<run_id>/`.

```bash
run-agent wrap --agent <agent> [wrap-flags] -- [agent-args...]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--agent` | string | Agent command to execute (for example `claude`, `codex`, `gemini`) |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | string | `JRUN_PROJECT_ID` or sanitized CWD basename | Project ID for tracked metadata |
| `--task` | string | auto-generated | Task ID (`task-<timestamp>-<slug>`) |
| `--root` | string | `~/run-agent` | Run-agent root directory |
| `--cwd` | string | current working directory | Working directory for wrapped command |
| `--message-bus` | string | `<task>/TASK-MESSAGE-BUS.md` | Message bus file path |
| `--parent-run-id` | string | `JRUN_ID` (if set) | Parent run ID |
| `--previous-run-id` | string | "" | Previous run ID |
| `--timeout` | duration | 0 | Max wrapped command duration; 0 means no limit |
| `--task-prompt` | string | auto-generated | TASK.md content when creating a new task |

**Examples:**

```bash
# Track a Claude prompt invocation as a task/run
run-agent wrap --agent claude -- -p "Explain this codebase"

# Wrap codex exec flags unchanged
run-agent wrap --agent codex -- exec --dangerously-bypass-approvals-and-sandbox "Refactor this file"

# Use explicit project/task
run-agent wrap \
  --project my-project \
  --task task-20260222-100100-shell-wrap \
  --agent gemini \
  -- --help
```

**Behavior:**

1. Resolves/creates project + task directories and writes `TASK.md` if missing.
2. Creates a new run directory and `run-info.yaml`.
3. Executes the wrapped CLI command with passed args unchanged.
4. Preserves terminal behavior by passing through stdio directly.
5. Writes run lifecycle events (`RUN_START`, `RUN_STOP` or `RUN_CRASH`) to task message bus.

**Note:** Because stdio is passed through to preserve CLI behavior, terminal transcript
is not mirrored into `agent-stdout.txt`/`agent-stderr.txt`; those files contain a notice.

#### `run-agent shell-setup`

Install or remove a managed shell init block that aliases `claude`, `codex`, and `gemini`
to `run-agent wrap --agent <agent> --`.

This feature is safe opt-in: no shell files are edited unless you run this command explicitly.

```bash
run-agent shell-setup install [flags]
run-agent shell-setup uninstall [flags]
```

**Install Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--shell` | string | inferred from `$SHELL` | Shell type (`zsh` or `bash`) |
| `--rc-file` | string | shell default (`~/.zshrc` or `~/.bashrc`) | Init file to edit (overrides `--shell`) |
| `--run-agent-bin` | string | `run-agent` | Executable token used in alias command |

**Uninstall Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--shell` | string | inferred from `$SHELL` | Shell type (`zsh` or `bash`) |
| `--rc-file` | string | shell default (`~/.zshrc` or `~/.bashrc`) | Init file to edit (overrides `--shell`) |

**Examples:**

```bash
# Install aliases in your inferred shell init file
run-agent shell-setup install

# Install aliases in bash init file
run-agent shell-setup install --shell bash

# Install aliases into an explicit file
run-agent shell-setup install --rc-file ~/.zshrc

# Remove the managed alias block
run-agent shell-setup uninstall
```

**Behavior:**

1. Adds a managed block with begin/end markers so changes are isolated.
2. `install` is idempotent: re-running updates/reuses the same managed block without duplicates.
3. `uninstall` is idempotent: re-running when absent is a no-op.
4. `uninstall` removes only the managed block added by this command.
5. Alias changes take effect after `source <rc-file>` or opening a new shell.

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
`http://localhost:14355` (conductor's port). If using `run-agent serve`, either open the UI
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
| `--delete-done-tasks` | bool | false | Delete task directories that have a DONE file, empty `runs/`, and are older than `--older-than` |
| `--rotate-bus` | bool | false | Rotate message bus files that exceed `--bus-max-size` |
| `--bus-max-size` | string | "10MB" | Size threshold for bus file rotation (e.g. `10MB`, `5MB`, `100KB`) |

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

# Delete task directories for completed (DONE) tasks older than 1 hour
run-agent gc --root ./runs --delete-done-tasks --older-than 1h

# Rotate oversized message bus files (default threshold: 10MB)
run-agent gc --root ./runs --rotate-bus

# Rotate bus files larger than 5MB
run-agent gc --root ./runs --rotate-bus --bus-max-size 5MB
```

**Behavior:**
- Skips runs that are currently `running`
- Reports freed disk space in MB
- Only deletes completed or failed runs older than the cutoff
- `--delete-done-tasks`: removes the entire task directory (including all runs, TASK.md, and message bus) when a DONE file is present and the task has no active runs
- `--rotate-bus`: archives the current bus file to a timestamped backup and starts a fresh bus file

---

#### `run-agent list`

List projects, tasks, or runs from the filesystem without a running server.

```bash
run-agent list [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | "./runs" | Root runs directory (uses `RUNS_DIR` env var if not set) |
| `--project` | string | "" | Project ID; lists tasks for this project if set |
| `--task` | string | "" | Task ID (requires `--project`); lists runs for this task if set |
| `--status` | string | "" | Filter tasks by status: `running`, `active`, `done`, `failed` (only applies when `--project` is set) |
| `--json` | bool | false | Output as JSON |
| `--activity` | bool | false | Include per-task progress signals (latest bus message, output/log activity, meaningful-signal age, drift risk) in task-list mode |
| `--drift-after` | duration | 20m | Drift threshold used by `--activity` to mark `analysis_drift_risk=true` |

**Behavior:**
- No `--project`: lists all project names in the root directory, one per line
- `--project`: shows a table of tasks with run count, latest status, and DONE marker
- `--project --task`: shows a table of runs with status, exit code, start time, and duration
- `--task` requires `--project`
- `--status` filters tasks when `--project` is set; unknown values show all tasks with a warning
- `--activity` is task-list mode only (`--project` without `--task`)
- With `--activity`, meaningful bus signals are derived from message types: `FACT`, `DECISION`, `ERROR`, `REVIEW`

**Examples:**

```bash
# List all projects
run-agent list --root ./runs

# List tasks in a project
run-agent list --root ./runs --project my-project

# List only running tasks
run-agent list --root ./runs --project my-project --status running

# List only done tasks
run-agent list --root ./runs --project my-project --status done

# List runs for a specific task
run-agent list --root ./runs --project my-project --task task-20260220-140000-hello

# JSON output for tasks
run-agent list --root ./runs --project my-project --json

# Task activity/drift scan for coordinators
run-agent list --root ./runs --project my-project --status running --activity --drift-after 30m
```

**Output examples:**

Listing projects (no `--project`):
```
my-project
other-project
```

Listing tasks (`--project`):
```
TASK_ID                           RUNS  LATEST_STATUS  DONE  BLOCKED_BY  LAST_ACTIVITY
task-20260220-140000-hello        3     completed      DONE  -           2026-02-20 14:12
task-20260220-150000-analysis     1     running        -     -           2026-02-20 15:30
task-20260220-160000-failed-task  2     failed         -     -           2026-02-20 16:11
```

Listing tasks with activity signals (`--project --activity`):
```
TASK_ID                       RUNS  LATEST_STATUS  DONE  BLOCKED_BY  LAST_ACTIVITY      LAST_BUS                                          LAST_OUTPUT        MEANINGFUL_AGE  DRIFT_RISK  DRIFT_REASON
task-20260220-150000-analysis 1     running        -     -           2026-02-20 15:30   2026-02-20T15:45:11Z PROGRESS validating paths  2026-02-20 15:46   42m0s           risk        no meaningful bus signal for 42m0s
```

Listing runs (`--project --task`):
```
RUN_ID                          STATUS   EXIT_CODE  STARTED              DURATION
20260220-1400000000-abc12345    success  0          2026-02-20 14:00:00  1m30s
20260220-1430000000-def67890    failed   1          2026-02-20 14:30:00  45s
```

**JSON output format:**

Listing projects (`--json` only):
```json
{
  "projects": ["my-project", "other-project"]
}
```

Listing tasks (`--project --json`):
```json
{
  "tasks": [
    {
      "task_id": "task-20260220-140000-hello",
      "runs": 3,
      "latest_status": "completed",
      "done": true,
      "last_activity": "2026-02-20T14:12:00Z"
    }
  ]
}
```

Listing tasks with activity signals (`--project --activity --json`):
```json
{
  "tasks": [
    {
      "task_id": "task-20260220-150000-analysis",
      "runs": 1,
      "latest_status": "running",
      "done": false,
      "last_activity": "2026-02-20T15:30:00Z",
      "activity": {
        "latest_bus_message": {
          "timestamp": "2026-02-20T15:45:11Z",
          "type": "PROGRESS",
          "body_preview": "validating path handling and collecting edge-case notes"
        },
        "latest_output_activity_at": "2026-02-20T15:46:03Z",
        "last_meaningful_signal_at": "2026-02-20T15:04:40Z",
        "meaningful_signal_age_seconds": 2520,
        "analysis_drift_risk": true,
        "drift_threshold_seconds": 1800,
        "drift_reason": "no meaningful bus signal for 42m0s"
      }
    }
  ]
}
```

Listing runs (`--project --task --json`):
```json
{
  "runs": [
    {
      "run_id": "20260220-1400000000-abc12345",
      "status": "success",
      "exit_code": 0,
      "started": "2026-02-20 14:00:00",
      "duration": "1m30s"
    }
  ]
}
```

---

#### `run-agent status`

Show latest run status per task (first-class replacement for manual `task-*` shell loops).

```bash
run-agent status --project <id> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--root` | string | "./runs" | Root runs directory (uses `RUNS_DIR` env var if not set) |
| `--project` | string | "" | Project ID (required) |
| `--task` | string | "" | Task ID (optional; defaults to all tasks in project) |
| `--status` | string | "" | Filter rows by status: `running`, `active`, `completed`, `failed`, `blocked`, `done`, `pending` |
| `--json` | bool | false | Output as JSON (`{"tasks":[...]}`) |
| `--concise` | bool | false | Output concise tab-separated rows: `task_id status exit_code latest_run done pid_alive` |
| `--activity` | bool | false | Include per-task progress signals: latest bus message, latest output/log activity, meaningful-signal age, and drift risk |
| `--drift-after` | duration | 20m | Drift threshold used by `--activity` to mark `analysis_drift_risk=true` |

**Behavior:**
- Derives the latest run per task from `<task>/runs/*` (lexicographically last run directory)
- Reads `status` and `exit_code` from `run-info.yaml`
- Includes `done` (`DONE` marker) and `pid_alive` fields
- `--status` filters rows; `active` is an alias for `running`, `done` filters by `DONE` marker, and `pending` matches tasks with no runs (`status = -`)
- Prints an explicit no-match message when filters yield zero rows (avoids silent output)
- `--json` and `--concise` are mutually exclusive
- With `--activity`, meaningful bus signals are derived from message types: `FACT`, `DECISION`, `ERROR`, `REVIEW`

**Examples:**

```bash
# Table output for all tasks in a project
run-agent status --root ./runs --project my-project

# Loop-equivalent "only running latest runs" view
run-agent status --root ./runs --project my-project --status running

# Automation-friendly concise rows
run-agent status --root ./runs --project my-project --status running --concise

# Coordinator-friendly activity/drift view
run-agent status --root ./runs --project my-project --status running --activity --drift-after 30m

# JSON activity fields for scripting
run-agent status --root ./runs --project my-project --status running --activity --json
```

**Output examples:**

Table output:
```
TASK_ID                           STATUS   EXIT_CODE  LATEST_RUN                      DONE   PID_ALIVE  BLOCKED_BY
task-20260220-140000-hello        completed 0         20260220-1401000000-abc12345   true   false      -
task-20260220-150000-analysis     running   -1        20260220-1500000000-def67890   false  true       -
```

Table output with activity signals (`--activity`):
```
TASK_ID                       STATUS   EXIT_CODE  LATEST_RUN                    DONE   PID_ALIVE  BLOCKED_BY  LAST_BUS                                          LAST_OUTPUT        MEANINGFUL_AGE  DRIFT_RISK  DRIFT_REASON
task-20260220-150000-analysis running  -1         20260220-1500000000-def67890 false  true       -           2026-02-20T15:45:11Z PROGRESS validating paths  2026-02-20 15:46   42m0s           risk        no meaningful bus signal for 42m0s
```

Concise output (`--concise`):
```
task-20260220-150000-analysis	running	-1	20260220-1500000000-def67890	false	true
```

Concise output with activity (`--concise --activity`):
```
task-20260220-150000-analysis	running	-1	20260220-1500000000-def67890	false	true	42m0s	true	PROGRESS	2026-02-20T15:45:11Z	validating paths	2026-02-20T15:46:03Z
```

No-match output (`--status running --concise` with no active tasks):
```
No status rows matched --status "running" in project my-project.
```

JSON output with activity (`--activity --json`) adds a stable `activity` object per task:
```json
{
  "tasks": [
    {
      "task_id": "task-20260220-150000-analysis",
      "status": "running",
      "exit_code": -1,
      "latest_run": "20260220-1500000000-def67890",
      "done": false,
      "pid_alive": true,
      "activity": {
        "latest_bus_message": {
          "timestamp": "2026-02-20T15:45:11Z",
          "type": "PROGRESS",
          "body_preview": "validating path handling and collecting edge-case notes"
        },
        "latest_output_activity_at": "2026-02-20T15:46:03Z",
        "last_meaningful_signal_at": "2026-02-20T15:04:40Z",
        "meaningful_signal_age_seconds": 2520,
        "analysis_drift_risk": true,
        "drift_threshold_seconds": 1800,
        "drift_reason": "no meaningful bus signal for 42m0s"
      }
    }
  ]
}
```

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
| `--check-tokens` | bool | false | Verify token files are readable and non-empty |

**Examples:**

```bash
# Validate config and all agents
run-agent validate --config config.yaml

# Validate a specific agent
run-agent validate --config config.yaml --agent claude

# Validate config and root directory
run-agent validate --config config.yaml --root ./runs

# Verify token files are accessible and non-empty
run-agent validate --config config.yaml --check-tokens
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

**`--check-tokens` output:**

When `--check-tokens` is set, an additional "Token checks" section is printed after the agents table. Each agent is reported with one of these statuses:

| Status | Meaning |
|--------|---------|
| `[OK]` | Token is set (inline token, token file readable, or env var present) |
| `[MISSING - file not found]` | `token_file` path does not exist |
| `[EMPTY]` | Token file exists but contains only whitespace |
| `[NOT SET]` | No inline token, no token file, and the env var is unset |

```
Conductor Loop Configuration Validator

Config: config.yaml

Agents:
  ✓ claude      2.1.49     (CLI found)
  ✓ codex       0.104.0    (CLI found)

Validation: 2 OK, 0 WARNING

Token checks:
  Agent claude:         token_file /run/secrets/anthropic-key [OK]
  Agent codex:          env OPENAI_API_KEY [NOT SET]
```

Exit code is 1 if any token check fails.

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
| `--root` | string | "" | Root directory for project/task bus resolution (default: `$RUNS_DIR`, then `./runs`) |
| `--type` | string | "INFO" | Message type |
| `--project` | string | "" | Project ID (used with `--root`/`--task` to resolve bus path; also sets message `project_id`) |
| `--task` | string | "" | Task ID (used with `--project` to resolve task-level bus; also sets message `task_id`) |
| `--run` | string | "" | Run ID |
| `--body` | string | "" | Message body (reads stdin if not provided and stdin is a pipe) |

**Examples:**

```bash
# Post to a task-scoped bus (TASK-MESSAGE-BUS.md)
run-agent bus post \
  --project my-project \
  --task task-20260221-120000-feat \
  --root ./runs \
  --type PROGRESS \
  --body "starting implementation"

# Post to a project-scoped bus (PROJECT-MESSAGE-BUS.md)
run-agent bus post \
  --project my-project \
  --root ./runs \
  --type DECISION \
  --body "using approach B for retries"

# Post using MESSAGE_BUS (common inside agent runs)
MESSAGE_BUS=/data/runs/my-project/task-20260221-120000-feat/TASK-MESSAGE-BUS.md \
  run-agent bus post --type FACT --body "tests passed"

# Post from stdin
echo "waiting for human input" | run-agent bus post --project my-project --type QUESTION
```

**Practical type conventions:**

| Type | When to use |
|------|-------------|
| `PROGRESS` | Start of a major step or checkpoint |
| `FACT` | Concrete outcome (tests, files, run IDs, commit SHAs) |
| `DECISION` | Non-obvious choice and brief rationale |
| `ERROR` | Blocker plus attempted remediation |
| `QUESTION` | Explicit request for input |
| `INFO` | Neutral status updates |

`RUN_START`, `RUN_STOP`, and `RUN_CRASH` are typically emitted by the runner automatically.

##### `run-agent bus read`

Read messages from a message bus file.

```bash
run-agent bus read [--bus PATH] [--tail N] [--follow]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--bus` | string | "" | Path to message bus file (uses `$MESSAGE_BUS` env var if not set) |
| `--root` | string | "" | Root directory for project/task bus resolution (default: `$RUNS_DIR`, then `./runs`) |
| `--project` | string | "" | Project ID (with `--root` to resolve bus path; reads project-level bus without `--task`) |
| `--task` | string | "" | Task ID (requires `--project`; resolves task-level bus) |
| `--tail` | int | 20 | Print last N messages |
| `--follow` | bool | false | Watch for new messages (Ctrl-C to exit) |

**Examples:**

```bash
# Follow task-scoped messages in real time
run-agent bus read \
  --project my-project \
  --task task-20260221-120000-feat \
  --root ./runs \
  --follow

# Read the latest project-scoped messages
run-agent bus read --project my-project --root ./runs --tail 50

# Read from an explicit file path
run-agent bus read --bus /tmp/custom-bus.md --tail 10
```

**Read/Post/Follow workflow (recommended):**

1. Read recent context with `run-agent bus read --tail <N>`.
2. Follow live traffic in a second terminal with `run-agent bus read --follow`.
3. Post step transitions and outcomes with `run-agent bus post`.

```bash
# Terminal A: follow live task traffic
run-agent bus read --project my-project --task task-20260221-120000-feat --root ./runs --follow

# Terminal B: post lifecycle updates
run-agent bus post --project my-project --task task-20260221-120000-feat --root ./runs \
  --type PROGRESS --body "starting dependency update"
run-agent bus post --project my-project --task task-20260221-120000-feat --root ./runs \
  --type FACT --body "dependency update complete"
```

**Path resolution rules:**

- `run-agent bus post`: `--bus` -> `$MESSAGE_BUS` -> resolved from `--project`/`--task` (+ `--root`).
- `run-agent bus read`: resolved from `--project`/`--task` (+ `--root`) -> `--bus` -> `$MESSAGE_BUS`.
- For `read`, `--bus` and `--project` are mutually exclusive.
- Resolved paths are:
  - Project scope: `<root>/<project>/PROJECT-MESSAGE-BUS.md`
  - Task scope: `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`

---

#### `run-agent watch`

Watch one or more tasks until they all reach a terminal state (completed or failed).

```bash
run-agent watch --project <id> --task <id> [--task <id> ...] [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | string | "" | Project ID (required) |
| `--task` | stringArray | [] | Task ID to watch; repeatable for multiple tasks (at least one required) |
| `--root` | string | "./runs" | Root runs directory (uses `RUNS_DIR` env var if not set) |
| `--timeout` | duration | 30m | Maximum wait time; exits with code 1 if timeout is reached |
| `--json` | bool | false | Output status as JSON lines |

**Behavior:**
- Polls run status every 2 seconds by reading `run-info.yaml` for each task's latest run
- Exits immediately with code 0 when all watched tasks have reached a terminal status
- Exits with code 1 (and an error message) if the timeout is reached before all tasks complete
- Both `completed` and `failed` are considered terminal (done) states

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | All tasks reached a terminal state (completed or failed) |
| 1 | Timeout reached; not all tasks completed within the time limit |

**Examples:**

```bash
# Watch a single task until it completes (or 30m timeout)
run-agent watch --root ./runs --project my-project --task task-20260220-140000-hello

# Watch multiple tasks simultaneously
run-agent watch --root ./runs --project my-project \
  --task task-20260220-140000-hello \
  --task task-20260220-140001-world \
  --task task-20260220-140002-review

# Watch with a custom timeout
run-agent watch --root ./runs --project my-project \
  --task task-20260220-140000-hello \
  --timeout 10m

# Watch with JSON output (one JSON line per poll cycle)
run-agent watch --root ./runs --project my-project \
  --task task-20260220-140000-hello \
  --json
```

**Text Output (default):**

```
Watching 2 task(s) for project "my-project":
  task-20260220-140000-hello              [running   ] elapsed: 0m15s
  task-20260220-140001-world              [completed ] duration: 1m10s
Waiting for 1 running task(s)... (timeout in 29m44s)
  task-20260220-140000-hello              [completed ] duration: 1m05s
  task-20260220-140001-world              [completed ] duration: 1m10s
All tasks complete.
```

**JSON Output (`--json`):**

Each poll cycle emits one JSON line:

```json
{"tasks":[{"task_id":"task-20260220-140000-hello","status":"completed","elapsed":65.3,"done":true}],"all_done":true}
```

JSON fields per task:

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Task ID |
| `status` | string | Latest run status (`running`, `completed`, `failed`, `unknown`) |
| `elapsed` | float64 | Elapsed seconds (from start to now, or start to end if terminal) |
| `done` | bool | Whether the task has reached a terminal state |

**Use `watch` in scripts:**

```bash
#!/bin/bash
# Submit two tasks, then wait for both to finish

run-agent job --project my-project --task task-20260220-140000-step1 \
  --root ./runs --agent claude --prompt "Step 1"

run-agent job --project my-project --task task-20260220-140001-step2 \
  --root ./runs --agent claude --prompt "Step 2"

# Wait up to 1 hour for both
run-agent watch --root ./runs --project my-project \
  --task task-20260220-140000-step1 \
  --task task-20260220-140001-step2 \
  --timeout 1h && echo "All done!" || echo "Timed out"
```

---

#### `run-agent output`

Print or tail output files from a completed or running job.

```bash
run-agent output [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | string | "" | Project ID (required unless `--run-dir` is used) |
| `--task` | string | "" | Task ID (required unless `--run-dir` is used) |
| `--run` | string | "" | Run ID (uses most recent run if omitted) |
| `--run-dir` | string | "" | Direct path to a run directory (overrides `--project/--task/--run`) |
| `--root` | string | "./runs" | Root runs directory (uses `RUNS_DIR` env var if not set) |
| `--file` | string | "output" | File to print: `output`, `stdout`, `stderr`, or `prompt` |
| `--tail` | int | 0 | Print last N lines only (0 = all) |
| `--follow`, `-f` | bool | false | Follow output as it is written (for running jobs) |

**`--file` options:**

| Value | File | Notes |
|-------|------|-------|
| `output` (default) | `output.md` | Falls back to `agent-stdout.txt` if `output.md` is absent |
| `stdout` | `agent-stdout.txt` | Raw agent stdout |
| `stderr` | `agent-stderr.txt` | Raw agent stderr |
| `prompt` | `prompt.md` | The prompt that was sent to the agent |

**`--follow` behavior:**
- If the run is already complete, prints all content and exits immediately
- For running jobs, polls every 500ms for new content written to `agent-stdout.txt`
- Stops automatically when the run transitions to a terminal status
- Also stops after 60 seconds with no new data
- Handles `Ctrl+C` gracefully

**Examples:**

```bash
# Print output of the most recent run
run-agent output --root ./runs --project my-project --task task-20260220-140000-hello

# Follow live output of a running job
run-agent output --root ./runs --project my-project --task task-20260220-150000-analysis --follow

# Print last 50 lines of raw stdout from a specific run
run-agent output \
  --root ./runs \
  --project my-project \
  --task task-20260220-140000-hello \
  --run 20260220-1400000000-abc12345 \
  --file stdout \
  --tail 50

# Print output using a direct run directory path
run-agent output --run-dir ./runs/my-project/task-20260220-140000-hello/runs/20260220-1400000000-abc12345

# Print the prompt that was sent to the agent
run-agent output --root ./runs --project my-project --task task-20260220-140000-hello --file prompt
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
