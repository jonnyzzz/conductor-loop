# Installation Guide

This guide is aligned with current facts, `install.sh`, and live CLI help output.

## Prerequisites

- Go `1.24.0` or newer (required for source builds; see `go.mod`)
- Git (for source checkout)
- One configured agent token (for real task execution)

Optional:

- `curl` or `wget` (installer download)
- Docker (container deployment)

## Option 1: Install `run-agent` from Latest Release (Recommended)

From repo root:

```bash
./install.sh
```

`install.sh` defaults:

- mirror base: `https://run-agent.jonnyzzz.com/releases/latest/download`
- fallback base: `https://github.com/jonnyzzz/conductor-loop/releases/latest/download`
- install dir: `/usr/local/bin`

Supported overrides:

- `RUN_AGENT_DOWNLOAD_BASE`
- `RUN_AGENT_FALLBACK_DOWNLOAD_BASE`
- `RUN_AGENT_INSTALL_DIR`

Example:

```bash
RUN_AGENT_INSTALL_DIR="$HOME/.local/bin" ./install.sh
```

Integrity:

- Installer downloads `<asset>` and `<asset>.sha256`
- Verifies SHA-256 before installation

## Option 2: Build from Source

```bash
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop

go build -o ./bin/run-agent ./cmd/run-agent
go build -o ./bin/conductor ./cmd/conductor
```

Verify:

```bash
./bin/run-agent --version
./bin/conductor --version
```

## Windows Launcher (`run-agent.cmd`)

`run-agent.cmd` resolves the runtime binary in this order:

1. `RUN_AGENT_BIN`
2. sibling binary (`run-agent` or `run-agent.exe`)
3. `dist/run-agent-<os>-<arch>` (or Windows `dist\run-agent-windows-<arch>.exe`)
4. PATH lookup (`run-agent`, unless `RUN_AGENT_CMD_DISABLE_PATH=1`)

## Initial Configuration

Create config directory:

```bash
mkdir -p ~/.config/conductor
mkdir -p ~/.config/conductor/tokens
```

Create token file(s):

```bash
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

Important:

- `~/.conductor/config.yaml` is stale; use `~/.config/conductor/config.yaml`.
- Per-agent `timeout` is not part of the runtime schema.

## Validate Installation

```bash
run-agent --help
run-agent validate --config ~/.config/conductor/config.yaml
run-agent validate --config ~/.config/conductor/config.yaml --check-tokens
```

## Start the Server

### `run-agent serve` (default port `14355`)

```bash
run-agent serve --config ~/.config/conductor/config.yaml --root ./runs
```

### `conductor` current binary (default port `8080`)

```bash
./bin/conductor --config ~/.config/conductor/config.yaml --root ./runs
```

Port note:

- `run-agent serve` default: `14355`
- current `./bin/conductor` default: `8080`

If you want explicit consistency, pass `--port` explicitly.

## Smoke Test

Start one task (Ralph loop):

```bash
run-agent task \
  --project demo \
  --task task-20260223-000000-hello \
  --agent codex \
  --prompt "Write a short hello-world output" \
  --config ~/.config/conductor/config.yaml \
  --root ./runs
```

Inspect status/output:

```bash
run-agent status --project demo --root ./runs
run-agent output --project demo --task task-20260223-000000-hello --root ./runs
```

## Troubleshooting

### Go version too old

```bash
go version
```

Use Go `1.24.0+`.

### Port already in use

Check ports:

```bash
lsof -i :14355
lsof -i :8080
```

Then either stop the conflicting process or start with a different `--port`.

### Config not found

Check discovery locations:

- `./config.yaml`
- `./config.yml`
- `./config.hcl`
- `~/.config/conductor/config.yaml`
- `~/.config/conductor/config.yml`
- `~/.config/conductor/config.hcl`

### Invalid config field errors

Common cause: old docs/examples using `agents.<name>.timeout`.

Remove per-agent timeout and keep timeout under `defaults.timeout`.

## Next Steps

- [Quick Start](quick-start.md)
- [Configuration](configuration.md)
- [CLI Reference](cli-reference.md)
