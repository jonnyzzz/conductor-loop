# Quick Start Guide

This quick start uses current `run-agent` behavior and schema.

## 1. Prepare Config and Token

```bash
mkdir -p ~/.config/conductor/tokens

echo "<your-token>" > ~/.config/conductor/tokens/codex.token
chmod 600 ~/.config/conductor/tokens/codex.token
```

Create `~/.config/conductor/config.yaml`:

```yaml
agents:
  codex:
    type: codex
    token_file: ~/.config/conductor/tokens/codex.token

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

Notes:

- Use `~/.config/conductor/config.yaml` (not `~/.conductor/config.yaml`).
- Per-agent `timeout` is not implemented in runtime config.

## 2. Validate

```bash
run-agent validate --config ~/.config/conductor/config.yaml --check-tokens
```

## 3. Run Your First Task (CLI-only)

```bash
run-agent task \
  --project my-project \
  --task task-20260223-100000-hello-world \
  --agent codex \
  --prompt "Write a short hello-world response in markdown" \
  --config ~/.config/conductor/config.yaml \
  --root ./runs
```

Watch and inspect:

```bash
run-agent watch --project my-project --task task-20260223-100000-hello-world --root ./runs
run-agent status --project my-project --root ./runs
run-agent output --project my-project --task task-20260223-100000-hello-world --root ./runs
```

## 4. Message Bus Basics

Read task messages:

```bash
run-agent bus read \
  --project my-project \
  --task task-20260223-100000-hello-world \
  --root ./runs \
  --tail 20
```

Post a progress message:

```bash
run-agent bus post \
  --project my-project \
  --task task-20260223-100000-hello-world \
  --root ./runs \
  --type PROGRESS \
  --body "manual checkpoint reached"
```

## 5. Optional: Start HTTP Server

`run-agent serve` (default port `14355`):

```bash
run-agent serve --config ~/.config/conductor/config.yaml --root ./runs
```

Then use API-client commands:

```bash
run-agent server status
run-agent server task list --project my-project
run-agent server bus read --project my-project --task task-20260223-100000-hello-world --tail 10
```

## 6. Optional: `conductor` Binary

Current checked-in `./bin/conductor` defaults to port `8080` and uses server URL `http://localhost:8080` for subcommands.

Start it explicitly:

```bash
./bin/conductor --config ~/.config/conductor/config.yaml --root ./runs --port 14355
```

Then:

```bash
./bin/conductor status
./bin/conductor task list --project my-project
./bin/conductor watch --project my-project --task task-20260223-100000-hello-world
```

## 7. Common Follow-ups

Resume an exhausted task:

```bash
run-agent resume --project my-project --task task-20260223-100000-hello-world --root ./runs
```

Stop a running task:

```bash
run-agent stop --project my-project --task task-20260223-100000-hello-world --root ./runs
```

Clean old runs:

```bash
run-agent gc --root ./runs --older-than 168h --dry-run
```

## Known CLI Edge Case

`run-agent watch --help` describes `--task` as optional, but the current implementation requires at least one `--task`.

## Next Steps

- [Configuration](configuration.md)
- [CLI Reference](cli-reference.md)
- [API Reference](api-reference.md)
