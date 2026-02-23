# Configuration Reference

Complete reference for configuring Conductor Loop via `config.yaml`.

## Configuration File Location

Conductor Loop looks for configuration in this order:

1. Path specified with `--config` flag
2. `CONDUCTOR_CONFIG` environment variable
3. `./config.yaml` (current directory)
4. `~/.conductor/config.yaml` (user home)

## Complete Configuration Example

```yaml
# config.yaml - Complete example with all options

agents:
  codex:
    type: codex
    token: sk-xxxxx                           # Direct token (not recommended)
    token_file: /secrets/codex.token          # Token from file (recommended)
    timeout: 300                              # Agent timeout in seconds

  claude:
    type: claude
    token_file: /secrets/claude.token
    timeout: 300

  gemini:
    type: gemini
    token_file: /secrets/gemini.token
    timeout: 300

  perplexity:
    type: perplexity
    token_file: /secrets/perplexity.token
    timeout: 300

  xai:
    type: xai
    token_file: /secrets/xai.token
    timeout: 300

defaults:
  agent: codex                                # Default agent if not specified
  timeout: 300                                # Default timeout in seconds
  max_concurrent_root_tasks: 2                # Root-task slots (0 = unlimited, default)

api:
  host: 0.0.0.0                               # API server host
  port: 14355                                  # API server port
  cors_origins:                               # CORS allowed origins
    - http://localhost:3000
    - http://localhost:14355
  auth_enabled: false                         # Enable API key authentication
  # api_key: "your-secret-key"               # API key (set to enable auth)

storage:
  runs_dir: /data/runs                        # Directory for run storage
```

## Configuration Sections

### `agents` - Agent Configuration

Configure AI agents that execute tasks.

#### Agent Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Agent type: `codex`, `claude`, `gemini`, `perplexity`, `xai` |
| `token` | string | No* | API token (direct value) |
| `token_file` | string | No* | Path to file containing token |
| `timeout` | int | No | Timeout in seconds (default: 300) |

*Either `token` or `token_file` must be specified.

#### Agent Types

##### Codex (OpenAI)

```yaml
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token
    timeout: 300
```

Requirements:
- OpenAI API token with Codex access
- Token should start with `sk-`

##### Claude (Anthropic)

```yaml
agents:
  claude:
    type: claude
    token_file: ~/.conductor/tokens/claude.token
    timeout: 300
```

Requirements:
- Anthropic API token
- Access to Claude models

##### Gemini (Google)

```yaml
agents:
  gemini:
    type: gemini
    token_file: ~/.conductor/tokens/gemini.token
    timeout: 300
```

Requirements:
- Google Cloud API token
- Gemini API access

##### Perplexity

```yaml
agents:
  perplexity:
    type: perplexity
    token_file: ~/.conductor/tokens/perplexity.token
    timeout: 300
```

Requirements:
- Perplexity API token

##### xAI (Grok)

```yaml
agents:
  xai:
    type: xai
    token_file: ~/.conductor/tokens/xai.token
    timeout: 300
```

Requirements:
- xAI API token
- Access to Grok models

#### Token Management

**Recommended: Use `token_file`**

```yaml
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token
```

Benefits:
- Tokens not exposed in config
- Easy rotation
- Better security
- Supports secret management systems

**Not Recommended: Direct `token`**

```yaml
agents:
  codex:
    type: codex
    token: sk-xxxxxxxxxxxxx  # Exposed in config!
```

Issues:
- Token visible in config file
- Harder to rotate
- Security risk if config is committed to git

#### Token File Format

Token files should contain only the token:

```bash
# Create token file
echo "sk-your-token-here" > ~/.conductor/tokens/codex.token

# Set secure permissions
chmod 600 ~/.conductor/tokens/codex.token

# Verify
cat ~/.conductor/tokens/codex.token
```

The token should be on a single line with no extra whitespace.

### `defaults` - Default Settings

Default values applied when not specified in task creation.

```yaml
defaults:
  agent: codex                # Default agent
  timeout: 300                # Default timeout (seconds)
  max_concurrent_runs: 4      # 0 = unlimited (default)
  max_concurrent_root_tasks: 2  # Root-task concurrency limit (0 = unlimited)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `agent` | string | first agent | Default agent type |
| `timeout` | int | 300 | Default task timeout in seconds |
| `max_concurrent_runs` | int | 0 | Maximum simultaneous agent runs; 0 means unlimited |
| `max_concurrent_root_tasks` | int | 0 | Maximum concurrent root tasks scheduled by planner; 0 means unlimited |

#### Concurrency Limiting

When `max_concurrent_runs` is set to a positive integer, excess runs **wait** (queue) for a
slot rather than being rejected. This prevents runaway resource consumption when many tasks
are submitted simultaneously.

```yaml
defaults:
  max_concurrent_runs: 4  # At most 4 agents run at once; others wait in queue
```

The current queue depth is visible in the Prometheus metrics endpoint as
`conductor_queued_runs_total`. Set to `0` (the default) to disable the limit entirely.

#### Root-Task Planner Concurrency

`max_concurrent_root_tasks` controls how many **root tasks** can run concurrently.
When the limit is reached:

- New root tasks are accepted but marked `queued`.
- A planner chooses which tasks start immediately (up to `N`) and preserves deterministic FIFO order.
- As running root tasks complete/fail/stop, queued tasks are promoted automatically.

Fairness and determinism policy:

- Primary sort key: submission order (oldest first).
- Tie-breakers: `project_id`, `task_id`, then `run_id` (lexicographic).

Example:

```yaml
defaults:
  max_concurrent_root_tasks: 2
```

If five root tasks are submitted at once, two start immediately and three stay queued.
When one running root task exits, the oldest queued task is promoted.

### `api` - API Server Configuration

Configure the REST API server.

```yaml
api:
  host: 0.0.0.0                    # Listen address
  port: 14355                       # Listen port
  cors_origins:                    # CORS allowed origins
    - http://localhost:3000
    - https://app.example.com
  auth_enabled: true               # Enable API key authentication
  api_key: "your-secret-key"       # API key (required when auth_enabled: true)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | `0.0.0.0` | Server listen address |
| `port` | int | `14355` | Server listen port |
| `cors_origins` | []string | `[]` | CORS allowed origins |
| `auth_enabled` | bool | `false` | Enable API key authentication |
| `api_key` | string | `""` | API key value; required when `auth_enabled` is `true` |

**API key authentication:**

When `auth_enabled: true` and `api_key` is set, all API requests (except `/api/v1/health`, `/api/v1/version`, `/metrics`, and `/ui/`) must include the key via `Authorization: Bearer <key>` or `X-API-Key: <key>` header.

The API key can also be set via the `CONDUCTOR_API_KEY` environment variable, which automatically enables authentication:

```bash
export CONDUCTOR_API_KEY="your-secret-key"
./conductor --config config.yaml
```

Or via the `--api-key` CLI flag:

```bash
./conductor --config config.yaml --api-key "your-secret-key"
```

If `auth_enabled: true` is set without an `api_key`, a warning is logged and authentication is disabled.

#### Host Configuration

- `0.0.0.0`: Listen on all interfaces (default)
- `127.0.0.1`: Listen on localhost only (more secure)
- `192.168.1.10`: Listen on specific interface

#### CORS Configuration

CORS is required if the frontend is on a different origin:

```yaml
api:
  cors_origins:
    - http://localhost:3000        # Development frontend
    - https://conductor.example.com # Production frontend
```

To allow all origins (NOT RECOMMENDED for production):

```yaml
api:
  cors_origins:
    - "*"
```

### `webhook` - Webhook Notifications

Send HTTP POST notifications when runs complete. Useful for integrating with Slack, GitHub Actions, PagerDuty, or custom monitoring systems.

```yaml
webhook:
  url: "https://hooks.slack.com/services/..."  # Required: webhook endpoint
  events:                                       # Optional: filter events (default: all)
    - "run_stop"
  secret: "my-signing-secret"                  # Optional: HMAC-SHA256 signing secret
  timeout: "10s"                               # Optional: HTTP timeout (default: 10s)
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Webhook endpoint URL |
| `events` | []string | No | Events to send; if empty, sends all. Currently `run_stop` |
| `secret` | string | No | HMAC-SHA256 secret for `X-Conductor-Signature` header |
| `timeout` | string | No | HTTP timeout per attempt (default: `10s`) |

#### Webhook Payload

When a run completes, the following JSON is POSTed:

```json
{
  "event": "run_stop",
  "project_id": "my-project",
  "task_id": "task-20260221-...",
  "run_id": "20260221-...",
  "agent_type": "claude",
  "status": "completed",
  "exit_code": 0,
  "started_at": "2026-02-21T00:00:00Z",
  "stopped_at": "2026-02-21T00:05:00Z",
  "duration_seconds": 300,
  "error_summary": ""
}
```

Possible `status` values: `completed`, `failed`.

#### Webhook Signing

If `secret` is configured, each request includes an `X-Conductor-Signature: sha256=<hmac-hex>` header. Verify it in your endpoint:

```python
import hmac, hashlib

def verify(secret: str, body: bytes, sig_header: str) -> bool:
    expected = "sha256=" + hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, sig_header)
```

#### Delivery Guarantees

- Delivered asynchronously (does not block run finalization)
- Retried up to 3 times with exponential backoff (1s, 2s between attempts)
- Failures logged to the task message bus as `WARN` (non-fatal)

### `storage` - Storage Configuration

Configure persistent storage for runs.

```yaml
storage:
  runs_dir: /data/runs      # Run storage directory
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `runs_dir` | string | Yes | Directory for storing run data |

#### Storage Structure

```
runs_dir/
├── run_20260205_100001_abc123/
│   ├── metadata.json              # Run metadata
│   ├── output.log                 # Agent output
│   ├── status.txt                 # Current status
│   └── message-bus.jsonl          # Message bus events
├── run_20260205_100002_def456/
│   └── ...
└── ...
```

#### Storage Requirements

- Must be writable by the conductor process
- Should have sufficient space for logs
- Can be a network mount (NFS, etc.)
- Should be backed up regularly

## Environment Variable Overrides

Environment variables override config file values:

| Environment Variable | Config Field | Example |
|---------------------|--------------|---------|
| `CONDUCTOR_CONFIG` | N/A | Path to config file |
| `CONDUCTOR_ROOT` | N/A | Root directory for run-agent |
| `CONDUCTOR_DISABLE_TASK_START` | N/A | Disable task execution |
| `CONDUCTOR_API_KEY` | `api.api_key` | Sets API key and enables authentication |

### Example with Environment Variables

```bash
# Override config path
export CONDUCTOR_CONFIG=/etc/conductor/config.yaml

# Override root directory
export CONDUCTOR_ROOT=/opt/conductor

# Disable task execution (API-only mode)
export CONDUCTOR_DISABLE_TASK_START=true

# Start server
./conductor
```

## Configuration Examples

### Minimal Configuration

Minimum required configuration:

```yaml
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token

storage:
  runs_dir: ./runs
```

### Development Configuration

Good for local development:

```yaml
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token
    timeout: 600  # Longer timeout for debugging

defaults:
  agent: codex
  timeout: 600

api:
  host: 127.0.0.1  # Localhost only
  port: 14355
  cors_origins:
    - http://localhost:3000
    - http://localhost:5173  # Vite dev server

storage:
  runs_dir: ./runs
```

### Production Configuration

Recommended for production deployment:

```yaml
agents:
  claude:
    type: claude
    token_file: /secrets/claude.token
    timeout: 300

  codex:
    type: codex
    token_file: /secrets/codex.token
    timeout: 300

defaults:
  agent: claude
  timeout: 300

api:
  host: 0.0.0.0
  port: 14355
  cors_origins:
    - https://conductor.example.com

storage:
  runs_dir: /var/lib/conductor/runs
```

### Multi-Agent Configuration

Using multiple agents:

```yaml
agents:
  codex-fast:
    type: codex
    token_file: /secrets/codex.token
    timeout: 120  # Fast tasks

  codex-slow:
    type: codex
    token_file: /secrets/codex.token
    timeout: 3600  # Slow tasks

  claude:
    type: claude
    token_file: /secrets/claude.token
    timeout: 300

  gemini:
    type: gemini
    token_file: /secrets/gemini.token
    timeout: 300

defaults:
  agent: codex-fast
  timeout: 120

api:
  host: 0.0.0.0
  port: 14355
  cors_origins:
    - http://localhost:3000

storage:
  runs_dir: /data/runs
```

Then specify the agent per task:

```bash
# Use fast agent
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{"agent": "codex-fast", "prompt": "Quick task"}'

# Use slow agent
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{"agent": "codex-slow", "prompt": "Complex task"}'
```

### Docker Configuration

Configuration for Docker deployment:

```yaml
agents:
  codex:
    type: codex
    token_file: /secrets/codex.token  # Mounted volume
    timeout: 300

defaults:
  agent: codex

api:
  host: 0.0.0.0  # Listen on all interfaces
  port: 14355
  cors_origins:
    - http://localhost:3000

storage:
  runs_dir: /data/runs  # Mounted volume
```

Docker Compose volumes:

```yaml
# docker-compose.yml
services:
  conductor:
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./secrets:/secrets:ro  # Read-only tokens
      - ./runs:/data/runs      # Persistent storage
```

## Configuration Validation

### Check Configuration

Test your configuration:

```bash
# Try starting the server
./conductor --config config.yaml

# Check logs for errors
# conductor 2026/02/05 10:00:00 config load failed: ...
```

### Common Configuration Errors

#### Missing Token File

```
Error: open /secrets/codex.token: no such file or directory
```

Fix: Create the token file or update the path.

#### Invalid Agent Type

```
Error: unknown agent type: codex2
```

Fix: Use valid agent type: `codex`, `claude`, `gemini`, `perplexity`, `xai`.

#### Port In Use

```
Error: listen tcp :14355: bind: address already in use
```

Fix: Change port or stop the process using port 14355.

#### Permission Denied

```
Error: open /data/runs: permission denied
```

Fix: Ensure the runs directory is writable.

## Security Best Practices

1. **Never commit tokens to git**
   ```bash
   # Add to .gitignore
   echo "*.token" >> .gitignore
   echo "config.yaml" >> .gitignore
   ```

2. **Use restrictive file permissions**
   ```bash
   chmod 600 ~/.conductor/tokens/*.token
   chmod 644 config.yaml
   ```

3. **Use token files, not direct tokens**
   ```yaml
   # Good
   token_file: /secrets/codex.token

   # Bad
   token: sk-xxxxx
   ```

4. **Limit CORS origins in production**
   ```yaml
   # Don't use "*" in production
   cors_origins:
     - https://conductor.example.com
   ```

5. **Run with minimal permissions**
   ```bash
   # Create dedicated user
   sudo useradd -r -s /bin/false conductor
   sudo chown -R conductor:conductor /var/lib/conductor
   ```

## Next Steps

- [CLI Reference](cli-reference.md) - Command-line options
- [API Reference](api-reference.md) - REST API endpoints
- [Quick Start](quick-start.md) - Try it out
- [Troubleshooting](troubleshooting.md) - Solve configuration issues
