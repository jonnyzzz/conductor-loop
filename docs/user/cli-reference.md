# CLI Reference

This reference is verified against live help output from:

- `./bin/run-agent`
- `./bin/conductor`

Validation timestamp: 2026-02-25.

## Command Sets

### `run-agent` top-level commands

`bus`, `completion`, `gc`, `goal`, `help`, `job`, `list`, `monitor`, `output`, `resume`, `serve`, `server`, `shell-setup`, `status`, `stop`, `task`, `validate`, `watch`, `workflow`, `wrap`

### `conductor` top-level commands in the current binary

`bus`, `completion`, `help`, `job`, `project`, `status`, `task`, `watch`

Important:

- `run-agent iterate` is not available (`unknown command "iterate"`).
- `conductor monitor`, `conductor workflow`, and `conductor goal` are not available in the current `./bin/conductor` (`unknown command`).
- The checked-in `./bin/conductor` currently differs from source code defaults. This document reflects the live binary behavior.

## Port Defaults (Reconciled)

- `run-agent serve` default port: `14355`
- `conductor` (current `./bin/conductor`) default port: `14355`
- `run-agent server ...` API-client commands default `--server`: `http://localhost:14355`
- `conductor ...` API-client commands default `--server`: `http://localhost:14355`

## `run-agent`

### Global flags

- `-h, --help`
- `-v, --version`

### `run-agent task`

Usage:

```bash
run-agent task [flags]
run-agent task [command]
```

Subcommands:

- `delete`
- `resume`

Flags:

- `--agent string`
- `--child-poll-interval duration`
- `--child-wait-timeout duration`
- `--conductor-url string`
- `--config string`
- `--cwd string`
- `--dependency-poll-interval duration` (default `2s`)
- `--depends-on stringArray`
- `--max-restarts int`
- `--project string`
- `--prompt string`
- `--prompt-file string`
- `--restart-delay duration` (default `1s`)
- `--root string`
- `--task string`
- `--timeout duration` (default `0`, no idle-output timeout limit)

### `run-agent task delete`

Usage:

```bash
run-agent task delete [flags]
```

Flags:

- `--force`
- `--project string` (required)
- `--root string` (default: `$RUNS_DIR` or `./runs`)
- `--task string` (required)

### `run-agent task resume`

Usage:

```bash
run-agent task resume [flags]
```

Flags:

- `--agent string`
- `--child-poll-interval duration`
- `--child-wait-timeout duration`
- `--config string`
- `--cwd string`
- `--max-restarts int` (default `3`)
- `--project string`
- `--restart-delay duration` (default `1s`)
- `--root string`
- `--task string`

### `run-agent resume`

Usage:

```bash
run-agent resume [flags]
```

Flags:

- `--agent string` (if set, launches a new run after reset)
- `--config string`
- `--project string` (required)
- `--prompt string` (used when `--agent` is set)
- `--prompt-file string` (used when `--agent` is set)
- `--root string` (default `./runs`)
- `--task string` (required)

### `run-agent job`

Usage:

```bash
run-agent job [flags]
run-agent job [command]
```

Subcommand:

- `batch`

Flags:

- `--agent string`
- `--conductor-url string`
- `--config string`
- `--cwd string`
- `-f, --follow`
- `--parent-run-id string`
- `--previous-run-id string`
- `--project string`
- `--prompt string`
- `--prompt-file string`
- `--root string`
- `--task string`
- `--timeout duration` (default `0`, no idle-output timeout limit)

### `run-agent job batch`

Usage:

```bash
run-agent job batch [flags]
```

Flags:

- `--agent string`
- `--conductor-url string`
- `--config string`
- `--continue-on-fail`
- `--cwd string`
- `-f, --follow`
- `--parent-run-id string`
- `--project string`
- `--prompt stringArray`
- `--prompt-file stringArray`
- `--root string`
- `--task stringArray`
- `--timeout duration` (default `0`, no idle-output timeout limit)

### `run-agent wrap`

Usage:

```bash
run-agent wrap --agent <agent> -- [args...]
```

Flags:

- `--agent string`
- `--cwd string`
- `--parent-run-id string`
- `--previous-run-id string`
- `--project string`
- `--root string`
- `--task string`
- `--task-prompt string`
- `--timeout duration` (default `0`, no limit)

### `run-agent bus`

Usage:

```bash
run-agent bus [command]
```

Subcommands:

- `discover`
- `post`
- `read`

#### `run-agent bus post`

Usage:

```bash
run-agent bus post [flags]
```

Flags:

- `--body string`
- `--project string`
- `--root string` (default: `storage.runs_dir` from config, then `~/.run-agent/runs`)
- `--run string`
- `--task string`
- `--type string` (default `INFO`)

Bus path resolution order:

1. `MESSAGE_BUS` env (set by runners — canonical path for child agents)
2. `--project` (+ optional `--task`) path resolution
3. upward auto-discover from current directory
4. error

Note: `MESSAGE_BUS` precedes `--project` here because agents running inside a task inherit this env var from the runner and should use it automatically.

#### `run-agent bus read`

Usage:

```bash
run-agent bus read [flags]
```

Flags:

- `--follow`
- `--project string`
- `--root string` (default: `storage.runs_dir` from config, then `~/.run-agent/runs`)
- `--tail int` (default `20`)
- `--task string`

Bus path resolution order:

1. `--project` (+ optional `--task`) path resolution
2. `MESSAGE_BUS` env
3. upward auto-discover from current directory
4. error

#### `run-agent bus discover`

Usage:

```bash
run-agent bus discover [flags]
```

Flags:

- `--from string` (start directory; defaults to current working directory)

Search order per directory:

1. `TASK-MESSAGE-BUS.md`
2. `PROJECT-MESSAGE-BUS.md`
3. `MESSAGE-BUS.md`

### `run-agent list`

Usage:

```bash
run-agent list [flags]
```

Flags:

- `--activity`
- `--drift-after duration` (default `20m0s`)
- `--json`
- `--project string`
- `--root string` (default: `./runs` or `RUNS_DIR` env)
- `--status string` (`running`, `active`, `done`, `failed`, `blocked`)
- `--task string`

### `run-agent status`

Usage:

```bash
run-agent status [flags]
```

Flags:

- `--activity`
- `--concise`
- `--drift-after duration` (default `20m0s`)
- `--json`
- `--project string` (required)
- `--root string` (default: `./runs` or `RUNS_DIR` env)
- `--status string` (`running`, `active`, `completed`, `failed`, `blocked`, `done`, `pending`)
- `--task string`

### `run-agent watch`

Usage:

```bash
run-agent watch [flags]
```

Flags:

- `--json`
- `--project string` (required)
- `--root string` (default: `./runs` or `RUNS_DIR` env)
- `--task stringArray`
- `--timeout duration` (default `30m0s`)

Known behavior mismatch:

- Help text allows omitted `--task`, but current implementation returns: `at least one --task is required`.

### `run-agent stop`

Usage:

```bash
run-agent stop [flags]
```

Flags:

- `--force` (sends `SIGKILL` if graceful stop exceeds internal `30s` timeout)
- `--project string`
- `--root string`
- `--run string`
- `--run-dir string`
- `--task string`

### `run-agent output`

Usage:

```bash
run-agent output [flags]
```

Flags:

- `--file string` (`output` default, `stdout`, `stderr`, `prompt`)
- `-f, --follow`
- `--project string`
- `--root string` (default: `./runs` or `RUNS_DIR` env)
- `--run string`
- `--run-dir string`
- `--tail int`
- `--task string`

### `run-agent gc`

Usage:

```bash
run-agent gc [flags]
```

Flags:

- `--bus-max-size string` (default `10MB`)
- `--delete-done-tasks`
- `--dry-run`
- `--keep-failed`
- `--older-than duration` (default `168h0m0s`)
- `--project string`
- `--root string` (default: `./runs` or `RUNS_DIR` env)
- `--rotate-bus`

### `run-agent monitor`

Usage:

```bash
run-agent monitor [flags]
```

Flags:

- `--agent string`
- `--config string`
- `--cwd string`
- `--dry-run`
- `--interval duration` (default `30s`)
- `--once`
- `--project string` (required)
- `--rate-limit duration` (default `2s`)
- `--root string` (default: `./runs` or `RUNS_DIR` env)
- `--stale-after duration` (default `20m0s`)
- `--todo string` (default `TODOs.md`)

### `run-agent validate`

Usage:

```bash
run-agent validate [flags]
```

Flags:

- `--agent string`
- `--check-network`
- `--check-tokens`
- `--config string`
- `--root string`

### `run-agent shell-setup`

Subcommands:

- `install`
- `uninstall`

#### `run-agent shell-setup install`

Flags:

- `--rc-file string`
- `--run-agent-bin string` (default `run-agent`)
- `--shell string` (`zsh` or `bash`)

#### `run-agent shell-setup uninstall`

Flags:

- `--rc-file string`
- `--shell string` (`zsh` or `bash`)

### `run-agent serve`

Usage:

```bash
run-agent serve [flags]
```

Flags:

- `--api-key string`
- `--config string`
- `--disable-task-start`
- `--host string` (default `0.0.0.0`)
- `--port int` (default `14355`)
- `--root string`

### `run-agent server` (API client group)

Usage:

```bash
run-agent server [command]
```

Subcommands:

- `status`
- `task`
- `job`
- `project`
- `watch`
- `bus`
- `update`

Default `--server` URL across this group: `http://localhost:14355`.

#### `run-agent server status`

Flags:

- `--json`
- `--server string`

#### `run-agent server task` subcommands

- `stop`, `status`, `list`, `delete`, `logs`, `runs`, `resume`

Common flags by subcommand:

- `--server string`
- `--project string` (required for `list`, `delete`, `logs`, `runs`, `resume`; optional in `stop`, `status`)
- `--json` on `stop`, `status`, `list`, `delete`, `resume`
- `--follow`, `--run`, `--tail` on `logs`
- `--limit`, `--json` on `runs`
- `--status` filter on `list`: `running`, `active`, `done`, `failed`, `blocked`

#### `run-agent server job` subcommands

- `submit`, `list`

`submit` flags:

- `--agent string` (required)
- `--attach-mode string` (default `create`)
- `--depends-on stringArray`
- `--follow`
- `--json`
- `--project string` (required)
- `--project-root string`
- `--prompt string`
- `--prompt-file string`
- `--server string`
- `--task string`
- `--wait`

`list` flags:

- `--json`
- `--project string`
- `--server string`

#### `run-agent server project` subcommands

- `list`, `stats`, `gc`, `delete`

Highlights:

- all support `--server string`
- `list`, `stats`, `gc`, `delete` support `--json`
- `gc`: `--dry-run`, `--keep-failed`, `--older-than string`, `--project string` (required)
- `delete`: `<project-id>` arg and optional `--force`

#### `run-agent server watch`

Flags:

- `--interval duration` (default `5s`)
- `--json`
- `--project string` (required)
- `--server string`
- `--task stringArray`
- `--timeout duration` (default `30m0s`)

#### `run-agent server bus`

Subcommands: `read`, `post`

`read` flags:

- `--follow`
- `--json`
- `--project string` (required)
- `--server string`
- `--tail int`
- `--task string`

`post` flags:

- `--body string`
- `--project string` (required)
- `--server string`
- `--task string`
- `--type string` (default `INFO`)

#### `run-agent server update`

Subcommands: `start`, `status`

- `start` flags: `--binary string`, `--json`, `--server string`
- `status` flags: `--json`, `--server string`

### `run-agent goal`

Usage:

```bash
run-agent goal decompose [flags]
```

Flags:

- `--goal string`
- `--goal-file string`
- `--json`
- `--max-parallel int` (default `6`)
- `--out string`
- `--project string`
- `--root string`
- `--strategy string` (default `rlm`)
- `--template string` (default `THE_PROMPT_v5`)

### `run-agent workflow`

Usage:

```bash
run-agent workflow run [flags]
```

Flags:

- `--agent string`
- `--config string`
- `--cwd string`
- `--dry-run`
- `--from-stage int`
- `--json`
- `--project string`
- `--resume`
- `--root string`
- `--state-file string`
- `--task string`
- `--template string` (default `THE_PROMPT_v5`)
- `--timeout duration` (default `0`, no idle-output timeout limit)
- `--to-stage int` (default `12`)

### `run-agent completion`

Subcommands:

- `bash`
- `fish`
- `powershell`
- `zsh`

## `conductor` (Current `./bin/conductor`)

### Global flags

- `--api-key string`
- `--config string`
- `--disable-task-start`
- `--host string` (default `0.0.0.0`)
- `--port int` (default `14355`)
- `--root string`
- `-v, --version`

### Top-level subcommands

- `status`
- `watch`
- `bus`
- `job`
- `task`
- `project`

Default `--server` URL in these subcommands: `http://localhost:14355`.

### `conductor status`

Flags:

- `--json`
- `--server string`

### `conductor watch`

Flags:

- `--interval duration` (default `5s`)
- `--json`
- `--project string` (required)
- `--server string`
- `--task stringArray` (default: all tasks in project)
- `--timeout duration` (default `30m0s`)

### `conductor bus`

Subcommands: `read`, `post`

`read` flags:

- `--follow`
- `--json`
- `--project string` (required)
- `--server string`
- `--tail int`
- `--task string`

`post` flags:

- `--body string`
- `--project string` (required)
- `--server string`
- `--task string`
- `--type string` (default `INFO`)

### `conductor job`

Subcommands in current binary:

- `submit`
- `list`

`submit` flags:

- `--agent string` (required)
- `--attach-mode string` (default `create`)
- `--follow`
- `--json`
- `--project string` (required)
- `--project-root string`
- `--prompt string`
- `--prompt-file string`
- `--server string`
- `--task string`
- `--wait`

`list` flags:

- `--json`
- `--project string`
- `--server string`

Note:

- `submit-batch` is not available in the current `./bin/conductor`.

### `conductor task`

Subcommands:

- `stop`
- `status`
- `list`
- `delete`
- `logs`
- `runs`
- `resume`

Highlights:

- default server: `http://localhost:14355`
- `list --status` supports: `running`, `active`, `done`, `failed`
- `logs` supports `--follow`, `--run`, `--tail`
- `runs` supports `--limit` (default `50`) and `--json`

### `conductor project`

Subcommands:

- `list`
- `stats`
- `gc`
- `delete`

Highlights:

- default server: `http://localhost:14355`
- `gc` supports `--older-than` (default `168h`), `--dry-run`, `--keep-failed`, `--json`
- `delete` supports optional `--force` and `--json`

## Related Docs

- [Quick Start](quick-start.md)
- [Installation](installation.md)
- [Configuration](configuration.md)
- [API Reference](api-reference.md)
