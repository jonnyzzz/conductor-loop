# Deployment Architecture

This page documents deployment behavior as implemented in:

- `cmd/run-agent/serve.go`
- `cmd/conductor/main.go`
- `internal/api/server.go`
- `internal/config/config.go`
- `internal/storage/*`
- `cmd/run-agent/gc.go`
- `internal/api/self_update.go`
- `README.md`

## Deployment Model (Single Runtime, Two Binaries)

Conductor Loop has one server runtime (`Server` in `internal/api/server.go`) with two CLI entrypoints:

- `run-agent serve`: starts the HTTP API + SSE + Web UI server.
- `conductor` (no subcommand): starts the same server runtime in server-first CLI form.

Operationally:

- `run-agent` is the primary binary and supports both local filesystem workflows and server mode.
- `conductor` is optional and wraps the same server behavior with server-centric subcommands.
- The release installer installs `run-agent` only; server mode remains fully available via `run-agent serve`.
- `run-agent server ...` commands are API clients (they talk to a running server), while top-level local commands (`task`, `job`, `bus`, `gc`, `list`, etc.) operate directly on filesystem state.
- `scripts/start-conductor.sh` prefers a `conductor` binary when available and falls back to `run-agent serve` when it is not.

### Topology

```text
                          HTTP/REST + SSE
+--------------------+   +-----------------------------+
| CLI/API clients    |-->| run-agent serve OR          |
| run-agent server   |   | conductor (server mode)     |
| conductor <subcmd> |   | internal/api.Server         |
+--------------------+   +-------------+---------------+
                                      |
                                      | spawn/manage runs
                                      v
                           +----------+-----------+
                           | runner + agents      |
                           | Ralph loop + jobs    |
                           +----------+-----------+
                                      |
                                      v
                           +----------+-----------+
                           | Filesystem state      |
                           | <root>/<project>/...  |
                           +----------+-----------+
                                      ^
                                      |
+--------------------+                |
| Web UI (/ui/)      |----------------+
+--------------------+
```

## Config Discovery and Precedence

### 1. Config path discovery chain

Server startup resolves config path in this order:

1. `--config` CLI flag (if non-empty)
2. `CONDUCTOR_CONFIG` env var
3. `config.FindDefaultConfig()` search chain:
   - `./config.yaml`
   - `./config.yml`
   - `./config.hcl`
   - `$HOME/.config/conductor/config.yaml`
   - `$HOME/.config/conductor/config.yml`
   - `$HOME/.config/conductor/config.hcl`

Behavior difference:

- `run-agent serve`: config is optional; if none found, startup continues with defaults.
- `conductor` server mode: config is required (explicit or discovered); startup fails if none is found.

### 2. Config load and normalization

`config.LoadConfigForServer(...)`:

- Parses YAML by default, HCL when extension is `.hcl`.
- Applies API defaults (`host=0.0.0.0`, `port=14355`, SSE defaults).
- Applies env overrides inside config loading:
  - `CONDUCTOR_API_KEY` enables auth and overrides API key.
  - `CONDUCTOR_AGENT_<NAME>_TOKEN` overrides per-agent token.
- Resolves `storage.runs_dir` and `storage.extra_roots` to absolute paths relative to config file location (with `~` expansion support).

### 3. Runtime value precedence (server mode)

For resolved runtime settings, effective precedence is:

- `config_path`: `--config` > `CONDUCTOR_CONFIG` > default discovery
- `root_dir`: `--root` > `CONDUCTOR_ROOT` > `storage.runs_dir` (from config) > API default `$HOME/run-agent`
- `host`: explicit `--host` > `CONDUCTOR_HOST` > `api.host` > default `0.0.0.0`
- `port`: explicit `--port` > `CONDUCTOR_PORT` > `api.port` > default `14355`
- `api_key`: `--api-key` > `CONDUCTOR_API_KEY` > `api.api_key`

Additional rules:

- CLI default values do not override config unless the flag was explicitly set (`cmd.Flags().Changed(...)`).
- `CONDUCTOR_PORT` is treated as explicit when valid, which disables auto-bind range fallback.
- If `auth_enabled=true` but no API key is resolved, auth is disabled with a warning.
- `CONDUCTOR_DISABLE_TASK_START` (if set) overrides the CLI boolean for task-start enablement.
- `conductor` sets `disableTaskStart=true` if config loading fails; `run-agent serve` keeps running with defaults.

## Run Directory Structure

Canonical storage layout:

```text
<root>/
  <project_id>/
    PROJECT-MESSAGE-BUS.md
    PROJECT-ROOT.txt                  # API-created projects
    <task_id>/
      TASK.md
      TASK-MESSAGE-BUS.md
      TASK-CONFIG.yaml                # dependency config, when present
      DONE                            # task completion marker
      runs/
        <run_id>/
          run-info.yaml
          prompt.md
          output.md
          agent-stdout.txt
          agent-stderr.txt
```

Core points:

- Run metadata is persisted in `run-info.yaml` and updated atomically with lock-protected read-modify-write.
- `run-info.yaml` stores project/task/run IDs, status, PID/PGID, timestamps, output file paths, lineage (`parent_run_id`, `previous_run_id`), and ownership metadata.
- API lookup supports direct layout and compatibility layout (`<root>/runs/<project>/<task>`) for project/task discovery.
- `storage.extra_roots` are scanned in flat form (`<extra_root>/runs/<run_id>/run-info.yaml`) for additional run visibility.

## Retention and GC Policy (`run-agent gc`)

`run-agent gc` performs filesystem retention and optional maintenance.

Defaults and scope:

- Default root: `--root` or `RUNS_DIR`, else `./runs`.
- Default retention window: `--older-than=168h` (7 days).
- Optional project scope: `--project <id>`.
- Preview mode: `--dry-run`.

Run deletion policy:

- Never deletes `running` runs.
- Deletes only `completed` or `failed` runs older than cutoff.
- Age source: `start_time`, fallback to `end_time`.
- `--keep-failed` preserves failed runs.

Bus rotation policy:

- Enabled via `--rotate-bus`.
- Threshold via `--bus-max-size` (default `10MB`, supports `KB/MB/GB`).
- Rotates (renames) these files when oversized:
  - `<project>/PROJECT-MESSAGE-BUS.md`
  - `<project>/<task>/TASK-MESSAGE-BUS.md`
- Archive naming: `<bus>.YYYYMMDD-HHMMSS.archived`.

Task directory pruning policy:

- Enabled via `--delete-done-tasks`.
- Deletes task directory only when all conditions hold:
  - `DONE` file exists
  - `runs/` is empty (or missing)
  - task directory mtime is older than cutoff

## Self-Update and Deferred Handoff

Server self-update endpoint:

- `GET /api/v1/admin/self-update` status
- `POST /api/v1/admin/self-update` start request

CLI client path:

- `run-agent server update status`
- `run-agent server update start --binary <candidate>`

State model:

- `idle`
- `deferred` (waiting for active root runs to drain)
- `applying`
- `failed`

Behavior:

1. Candidate binary is validated (`exists`, `file`, `executable`).
2. Active root runs are counted.
3. If active root runs > 0, state becomes `deferred` and a worker polls every second.
4. While `deferred` or `applying`, new root task creation is rejected (`409`, self-update drain in progress).
5. On apply:
   - verify candidate by running `--version` (default 10s timeout)
   - backup current executable (`.rollback-<timestamp>`)
   - stage candidate (`.update-<timestamp>.tmp`) and replace target
   - hand off via re-exec with same args/env
6. On handoff failure, rollback is attempted and state becomes `failed`.

Platform note:

- Unix-like systems use `syscall.Exec` for in-place handoff.
- Native Windows does not support in-place self-update handoff (default re-exec returns an error); rollback is attempted and state becomes `failed`.

## Port Binding Behavior

Base configuration:

- Default bind host: `0.0.0.0`
- Default bind port: `14355`

Binding algorithm (`Server.ListenAndServe(explicit bool)`):

- `explicit=true`: one bind attempt on configured port; if busy, startup fails.
- `explicit=false`: tries up to 100 consecutive ports (`base` to `base+99`) and skips only `EADDRINUSE` collisions.

When `explicit` becomes true:

- `--port` flag explicitly set, or
- valid `CONDUCTOR_PORT` env var provided

Operational implication:

- Config-file port alone is non-explicit and still gets auto-bind range behavior.
- Wrapper scripts that always pass `--port` make port binding explicit (no auto-range fallback).
- Startup logs print the actual bound API/UI URLs; UI URL maps wildcard hosts to `localhost` for navigation.

## Ops Notes

- If you want a single deployed executable, `run-agent` is sufficient (`serve` + `server` client subcommands + local workflows).
- Use consistent root configuration across server and local commands; server default (`$HOME/run-agent`) differs from many local command defaults (`./runs`/`RUNS_DIR`).
- Use explicit `--port` for deterministic fail-fast service startup; omit explicit port only when you want automatic free-port fallback.
- Schedule `run-agent gc` (with appropriate `--older-than`, `--rotate-bus`, and optionally `--delete-done-tasks`) as regular housekeeping.
- On native Windows, plan self-update as an external restart rollout rather than in-place handoff.
