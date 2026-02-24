# Configuration Reference

This document reflects the current runtime config schema in `internal/config` and live CLI behavior.

## Personal Home Configuration (`~/.run-agent/conductor-loop.hcl`)

The primary place to store your API tokens and personal defaults is:

```
~/.run-agent/conductor-loop.hcl
```

This file is **created automatically on first run** with a commented template so
you can open it and fill in your tokens. It uses a simple HCL block format where
the agent type is inferred from the block name — no `type` field required:

```hcl
# ~/.run-agent/conductor-loop.hcl
# Full reference: https://github.com/jonnyzzz/conductor-loop/blob/main/docs/user/configuration.md

codex {
  token_file = "~/.config/tokens/openai"
}

claude {
  token_file = "~/.config/tokens/anthropic"
}

gemini {
  token_file = "~/.config/tokens/google"
}

defaults {
  agent               = "claude"
  timeout             = 300
  max_concurrent_runs = 4
}
```

## Configuration File Discovery

When `--config` is not passed, config resolution is:

1. `CONDUCTOR_CONFIG` environment variable
2. `./config.yaml` (project-local)
3. `./config.yml` (project-local)
4. `~/.run-agent/conductor-loop.hcl` ← **auto-created on first run**

Important:

- `~/.conductor/config.yaml` and `~/.config/conductor/config.yaml` are no longer
  in the discovery path. Use `~/.run-agent/conductor-loop.hcl` instead.
- `.hcl` files in project directories are **not** discovered automatically;
  pass them explicitly with `--config`.
- Both YAML (`config.yaml`) and HCL (`conductor-loop.hcl`) are supported.

## Config Precedence for Server Startup

For `run-agent serve` and `conductor`:

1. `--config`
2. `CONDUCTOR_CONFIG`
3. default discovery order above

Then environment overrides apply for host/port/auth (see Environment Overrides section).

## Schema

## Top-level sections

- `agents` (required)
- `defaults` (required)
- `api` (optional; defaults are applied)
- `storage` (optional but strongly recommended)
- `webhook` (optional)

### `agents`

Map keyed by agent name:

```yaml
agents:
  codex:
    type: codex
    token_file: ./secrets/codex.token
```

Fields:

- `type` (optional in HCL — inferred from block/agent name; required in YAML): one of `claude`, `codex`, `gemini`, `perplexity`, `xai`
- `token` (optional): inline token
- `token_file` (optional): token file path
- `base_url` (optional)
- `model` (optional)

Notes:

- `token` and `token_file` cannot both be set at once.
- `token_file` supports `~` expansion and relative paths (relative to config file directory).
- There is no per-agent `timeout` field in the runtime schema.

### `defaults`

```yaml
defaults:
  agent: codex
  timeout: 300
  max_concurrent_runs: 4
  max_concurrent_root_tasks: 2
  diversification:
    enabled: true
    strategy: round-robin
    agents: [codex, claude]
    fallback_on_failure: true
```

Fields:

- `agent` (string)
- `timeout` (int, required, must be `> 0`)
- `max_concurrent_runs` (int; `0` means unlimited)
- `max_concurrent_root_tasks` (int, must be `>= 0`)
- `diversification` (optional object)

`diversification` fields:

- `enabled` (bool)
- `strategy` (`round-robin` or `weighted`)
- `agents` (`[]string` of configured agent names)
- `weights` (`[]int`, required length match when strategy is weighted, all values `> 0`)
- `fallback_on_failure` (bool)

### `api`

```yaml
api:
  host: 0.0.0.0
  port: 14355
  cors_origins:
    - http://localhost:5173
  auth_enabled: false
  # api_key: "..."
  sse:
    poll_interval_ms: 100
    discovery_interval_ms: 1000
    heartbeat_interval_s: 30
    max_clients_per_run: 10
```

Fields:

- `host` (default `0.0.0.0`)
- `port` (default `14355`)
- `cors_origins` (`[]string`)
- `auth_enabled` (bool)
- `api_key` (string)
- `sse.poll_interval_ms` (default `100`)
- `sse.discovery_interval_ms` (default `1000`)
- `sse.heartbeat_interval_s` (default `30`)
- `sse.max_clients_per_run` (default `10`)

Validation:

- `api.port` must be between `0` and `65535`
- SSE numeric fields must be non-negative

### `storage`

```yaml
storage:
  runs_dir: ./runs
  extra_roots:
    - /mnt/other-runs
```

Fields:

- `runs_dir` (string)
- `extra_roots` (`[]string`, optional)

### `webhook`

```yaml
webhook:
  url: https://example.com/hook
  events: [run_stop]
  secret: signing-secret
  timeout: 10s
```

Fields:

- `url` (string URL)
- `events` (`[]string`)
- `secret` (string)
- `timeout` (duration string)

## Environment Overrides

- `CONDUCTOR_CONFIG`: config path
- `CONDUCTOR_ROOT`: root runs directory (used by server commands)
- `CONDUCTOR_HOST`: API host override
- `CONDUCTOR_PORT`: API port override
- `CONDUCTOR_DISABLE_TASK_START`: disable task execution (`true/1/yes/on`)
- `CONDUCTOR_API_KEY`: sets `api.api_key` and forces `api.auth_enabled=true`

Per-agent token override:

- `CONDUCTOR_AGENT_<AGENT_NAME>_TOKEN`
- Agent name is uppercased and non-alphanumeric characters become `_`

Examples:

- `CONDUCTOR_AGENT_CODEX_TOKEN`
- `CONDUCTOR_AGENT_MY_AGENT_TOKEN`

## Port Selection Behavior

Server bind behavior:

- If port is explicitly set (CLI flag/env override), bind must succeed on that exact port.
- If port is not explicit, server attempts up to 100 consecutive ports (`basePort` to `basePort+99`).

## Recommended Config (YAML)

```yaml
agents:
  codex:
    type: codex
    token_file: ~/.openai
  claude:
    type: claude
    token_file: ~/.anthropic

defaults:
  agent: codex
  timeout: 300
  max_concurrent_runs: 4
  max_concurrent_root_tasks: 2

api:
  host: 0.0.0.0
  port: 14355

storage:
  runs_dir: ./runs
```

## Equivalent HCL Example (`~/.run-agent/conductor-loop.hcl`)

HCL uses flat top-level blocks. Agent type is inferred from the block name so
`type` is optional.

```hcl
# Agent blocks — type inferred from block name, no "type" field required.
codex {
  token_file = "~/.config/tokens/openai"
}

claude {
  token_file = "~/.config/tokens/anthropic"
}

defaults {
  agent                  = "codex"
  timeout                = 300
  max_concurrent_runs    = 4
  max_concurrent_root_tasks = 2
}

api {
  host = "0.0.0.0"
  port = 14355
}

storage {
  runs_dir = "~/.run-agent/runs"
}
```

## Validation

Validate config and agent availability:

```bash
run-agent validate --config ./config.yaml
```

Token checks:

```bash
run-agent validate --config ./config.yaml --check-tokens
```

## Security Notes

- Prefer `token_file` over inline `token`.
- Keep token files out of version control.
- Restrict permissions on token files (for example `chmod 600`).
- Avoid wildcard CORS in production.

## Related Docs

- [Installation](installation.md)
- [Quick Start](quick-start.md)
- [CLI Reference](cli-reference.md)
