# FACTS: User & Developer Documentation
# Source: README.md, docs/user/*, docs/dev/*, config examples
# Format: [YYYY-MM-DD HH:MM:SS] [tags: ...] <fact text>
# Generated: 2026-02-23

---

## Installation

[2026-02-22 12:41:54] [tags: user-docs, installation, cli]
Installer URL (always-latest): `curl -fsSL https://run-agent.jonnyzzz.com/install.sh | bash`

[2026-02-22 12:41:54] [tags: user-docs, installation, cli]
Installer downloads from mirror first: `https://run-agent.jonnyzzz.com/releases/latest/download`, falls back to GitHub release assets.

[2026-02-22 12:41:54] [tags: user-docs, installation, cli]
Installer default install directory: `/usr/local/bin/run-agent`. Override: `RUN_AGENT_INSTALL_DIR="$HOME/.local/bin"`.

[2026-02-22 12:41:54] [tags: user-docs, installation, cli]
Installer verifies SHA256 checksums before install/update: asset `<name>.sha256`.

[2026-02-22 12:41:54] [tags: user-docs, installation, config]
Config file discovery order: `--config` flag → `CONDUCTOR_CONFIG` env var → `./config.yaml` → `~/.conductor/config.yaml`.

[2026-02-23 00:00:00] [tags: user-docs, installation, requirements]
Go requirement: 1.24 or higher (source build); documentation older than 2026-02-23 may say 1.21.

[2026-02-23 00:00:00] [tags: user-docs, installation, requirements]
Node.js requirement for frontend build: `^20.19.0 || >=22.12.0` (matches Vite 7 toolchain requirement).

[2026-02-23 00:00:00] [tags: user-docs, installation, requirements]
Docker requirement: 20.10+ (required for docs site serve/build; optional for non-container local runtime).

[2026-02-22 12:41:54] [tags: user-docs, installation, cli]
Build from source commands:
  `git clone https://github.com/jonnyzzz/conductor-loop.git`
  `go build -o run-agent ./cmd/run-agent`
  `go build -o conductor ./cmd/conductor`

[2026-02-22 12:41:54] [tags: user-docs, installation, cli]
Build all binaries to bin/: `go build -o bin/ ./cmd/...`

[2026-02-22 12:41:54] [tags: user-docs, installation, cli, windows]
Windows launcher: single-file `run-agent.cmd`; resolves binary via `RUN_AGENT_BIN` env var → `run-agent.exe` next to cmd → `dist\run-agent-windows-<arch>.exe` → PATH.

[2026-02-23 00:00:00] [tags: user-docs, installation, platform]
macOS quarantine fix: `xattr -d com.apple.quarantine conductor && xattr -d com.apple.quarantine run-agent`

[2026-02-23 00:00:00] [tags: user-docs, installation, platform]
Shell aliases install (wraps claude/codex/gemini calls as tracked runs): `run-agent shell-setup install` / `run-agent shell-setup uninstall`

---

## Default Port and Server URL

[2026-02-05 17:28:15] [tags: user-docs, config, server]
Default API server port: **14355**

[2026-02-05 17:28:15] [tags: user-docs, config, server]
Default API server host: `0.0.0.0`

[2026-02-05 17:28:15] [tags: user-docs, web-ui]
Web UI URL: `http://localhost:14355/ui/` (also accessible at root `/`)

[2026-02-23 00:00:00] [tags: user-docs, config, server]
Server auto-binds next free port from configured base port (up to 100 attempts) when port is not explicitly set via CLI/env.

[2026-02-20 22:02:53] [tags: user-docs, config, cli]
Override port via CLI flag: `conductor --config config.yaml --port 9090`

[2026-02-22 12:41:54] [tags: user-docs, config, server]
Startup wrappers default port 14355; override via `--port` flag or `CONDUCTOR_HOST`/`CONDUCTOR_PORT` env vars.

---

## Configuration Schema (config.yaml)

[2026-02-05 17:28:15] [tags: user-docs, config]
Minimal configuration:
```yaml
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token
storage:
  runs_dir: ./runs
```

[2026-02-05 17:28:15] [tags: user-docs, config, agents]
Agent fields: `type` (required), `token_file` (recommended) or `token` (not recommended), `timeout` (int seconds, default 300).

[2026-02-05 17:28:15] [tags: user-docs, config, agents]
Valid agent types: `codex`, `claude`, `gemini`, `perplexity`, `xai`

[2026-02-05 17:28:15] [tags: user-docs, config, defaults]
`defaults` section fields:
  - `agent`: default agent name
  - `timeout`: default timeout in seconds (default 300)
  - `max_concurrent_runs`: int (0 = unlimited, default)
  - `max_concurrent_root_tasks`: int (0 = unlimited, default)

[2026-02-23 00:00:00] [tags: user-docs, config, defaults]
`defaults.diversification`: optional agent diversification policy — fields: `enabled`, `strategy` (`round-robin` or `weighted`), `agents`, `weights`, `fallback_on_failure`.

[2026-02-05 17:28:15] [tags: user-docs, config, api]
`api` section fields:
  - `host`: string (default `0.0.0.0`)
  - `port`: int (default `14355`)
  - `cors_origins`: []string
  - `auth_enabled`: bool (default `false`)
  - `api_key`: string (required when `auth_enabled: true`)

[2026-02-21 05:33:45] [tags: user-docs, config, auth]
API key auth also enabled via `CONDUCTOR_API_KEY` env var (overrides config; automatically enables auth). CLI flag: `--api-key`.

[2026-02-21 05:33:45] [tags: user-docs, config, auth]
API key sent via header: `Authorization: Bearer <key>` or `X-API-Key: <key>`.

[2026-02-21 05:33:45] [tags: user-docs, config, auth]
Paths exempt from API key auth: `/api/v1/health`, `/api/v1/version`, `/metrics`, `/ui/`

[2026-02-05 17:28:15] [tags: user-docs, config, storage]
`storage` section fields:
  - `runs_dir`: string (required) — run storage directory
  - `extra_roots`: []string — additional root directories to scan

[2026-02-05 17:28:15] [tags: user-docs, config, webhook]
`webhook` section fields:
  - `url`: string (required)
  - `events`: []string (optional; currently only `run_stop`)
  - `secret`: string (optional; HMAC-SHA256 signing)
  - `timeout`: string (optional; default `10s`)

[2026-02-21 00:26:08] [tags: user-docs, config, webhook]
Webhook delivers `run_stop` payloads asynchronously with retry up to 3 times (backoff 1s, 2s). Failures logged as `WARN` to task message bus.

[2026-02-21 00:26:08] [tags: user-docs, config, webhook]
Webhook HMAC signature header: `X-Conductor-Signature: sha256=<hmac-hex>`

[2026-02-05 17:28:15] [tags: user-docs, config, env]
Environment variable overrides:
  - `CONDUCTOR_CONFIG` — config file path
  - `CONDUCTOR_ROOT` — root directory
  - `CONDUCTOR_DISABLE_TASK_START` — disable task execution
  - `CONDUCTOR_API_KEY` — API key (enables auth)

[2026-02-23 00:00:00] [tags: user-docs, config]
HCL format (`.hcl`) also supported alongside YAML (`.yaml`/`.yml`).

---

## Run Storage Structure

[2026-02-05 17:28:15] [tags: user-docs, dev-docs, storage]
Run storage path: `<root>/<project>/<task>/runs/<run_id>/`

[2026-02-05 17:28:15] [tags: user-docs, dev-docs, storage]
Key run files:
  - `run-info.yaml` — run metadata (status, exit code, agent_version, error_summary)
  - `agent-stdout.txt` — agent stdout (written live via O_APPEND)
  - `output.md` — agent-written summary
  - `prompt.md` — stored prompt for this run

[2026-02-05 17:28:15] [tags: user-docs, dev-docs, storage]
Message bus files:
  - `<task_dir>/TASK-MESSAGE-BUS.md` — task-scoped append-only bus
  - `<project_dir>/PROJECT-MESSAGE-BUS.md` — project-scoped bus

[2026-02-23 00:00:00] [tags: user-docs, dev-docs, storage]
Task config file: `<task_dir>/TASK-CONFIG.yaml` — persists `depends_on` and other task metadata.

[2026-02-23 00:00:00] [tags: user-docs, dev-docs, storage]
Form submission audit log: `<root>/_audit/form-submissions.jsonl`

[2026-02-23 00:00:00] [tags: user-docs, dev-docs, storage]
Project root saved at: `<root>/<project_id>/PROJECT-ROOT.txt`

[2026-02-23 00:00:00] [tags: user-docs, dev-docs, storage]
Task completion fact propagation state: `<task_dir>/TASK-COMPLETE-FACT-PROPAGATION.yaml`

[2026-02-23 00:00:00] [tags: user-docs, dev-docs, storage]
Threaded task link: `<task_dir>/TASK-THREAD-LINK.yaml`

---

## Environment Variables Injected into Agent Processes

[2026-02-05 17:28:15] [tags: user-docs, cli, agent]
JRUN_* env vars injected into agent process environment:
  - `TASK_FOLDER` — absolute path to task directory
  - `RUN_FOLDER` — absolute path to current run directory
  - `MESSAGE_BUS` — absolute path to `TASK-MESSAGE-BUS.md`
  - `JRUN_PROJECT_ID` — project identifier
  - `JRUN_TASK_ID` — task identifier
  - `JRUN_ID` — run identifier for this execution
  - `JRUN_PARENT_ID` — run ID of the parent (if spawned as sub-agent)
  - `CONDUCTOR_URL` — conductor server URL (if server is running)

---

## CLI Reference: `run-agent`

[2026-02-23 00:00:00] [tags: user-docs, cli]
`run-agent` top-level subcommands: `task`, `job`, `bus`, `list`, `status`, `watch`, `monitor`, `serve`, `server`, `validate`, `output`, `resume`, `stop`, `gc`, `wrap`, `shell-setup`, `workflow`, `goal`, `completion`

[2026-02-23 00:00:00] [tags: user-docs, cli]
`run-agent task` subcommands: `resume`, `delete`

[2026-02-23 00:00:00] [tags: user-docs, cli]
`run-agent job` subcommands: `batch`

[2026-02-23 00:00:00] [tags: user-docs, cli]
`run-agent bus` subcommands: `discover`, `read`, `post`

[2026-02-21 03:02:40] [tags: user-docs, cli, serve]
Start local server: `run-agent serve --host 127.0.0.1 --port 14355`

[2026-02-21 03:02:40] [tags: user-docs, cli, serve]
`run-agent serve` can start without config file (auto-discovery optional). `conductor` server mode requires `--config` or discoverable default config.

[2026-02-21 03:13:08] [tags: user-docs, cli, task]
Run a task (with Ralph Loop): `run-agent task --project P --root R --config C --agent A --prompt "..." [--max-restarts N]`

[2026-02-20 23:33:54] [tags: user-docs, cli, job]
Run a single job (no restart loop): `run-agent job --project P --root R --agent A --prompt "..." [--timeout 30m]`

[2026-02-21 07:05:59] [tags: user-docs, cli, job]
Batch local job submission: `run-agent job batch --project P --root R --agent A --prompt-file f1.md --prompt-file f2.md`

[2026-02-21 17:32:39] [tags: user-docs, cli, run-agent]
`--timeout` flag: kill sub-agent after duration (e.g., `30m`, `2h`): `run-agent job --timeout 30m ...`

[2026-02-23 00:00:00] [tags: user-docs, cli]
Task ID format: `task-<YYYYMMDD>-<HHMMSS>-<slug>`. Example: `task-20260220-190000-my-task`. Omit `--task` to auto-generate.

[2026-02-21 04:15:37] [tags: user-docs, cli, validation]
Agent CLI compatibility floors: `claude >= 1.0.0`, `codex >= 0.1.0`, `gemini >= 0.1.0` (older detected versions are warning-only).

[2026-02-21 04:15:37] [tags: user-docs, cli, validation]
Validate config: `run-agent validate --config config.yaml`
Validate with token check: `run-agent validate --check-tokens`
Validate with network check (placeholder): `run-agent validate --check-network`

[2026-02-21 02:01:28] [tags: user-docs, cli, list]
List projects/tasks/runs: `run-agent list --root runs` / `--project P` / `--task T`

[2026-02-21 02:01:28] [tags: user-docs, cli, output]
Follow live agent output: `run-agent output --project P --task T --follow --root R`
Read last N lines: `run-agent output --project P --task T --tail N --root R`
Read output.md: `run-agent output --project P --task T --file output --root R`

[2026-02-21 02:01:28] [tags: user-docs, cli, status]
Check task status (latest run per task): `run-agent status --root R --project P [--status running] [--concise] [--activity] [--drift-after Nd]`

[2026-02-21 02:28:08] [tags: user-docs, cli, watch]
Watch tasks locally (requires explicit `--task` IDs): `run-agent watch --root R --project P --task T1 --task T2`

[2026-02-21 05:16:21] [tags: user-docs, cli, stop-resume]
Stop a task: `run-agent stop --root R --project P --task T`
Resume an exhausted task: `run-agent resume --root R --project P --task T`
Resume and restart: `run-agent resume --root R --project P --task T --agent A --prompt-file f.md`

[2026-02-21 03:43:53] [tags: user-docs, cli, gc]
Garbage collection: `run-agent gc --dry-run --root R --older-than 168h`
Delete DONE task dirs: `run-agent gc --delete-done-tasks --root R --older-than 1h`
Rotate bus files: `run-agent gc --rotate-bus --bus-max-size 10MB --root R`
Additional flags: `--keep-failed`, `--project`

[2026-02-21 08:22:23] [tags: user-docs, cli, bus]
Read message bus: `run-agent bus read --project P --task T --root R [--follow] [--tail N]`
Post message: `run-agent bus post --type PROGRESS|FACT|DECISION|ERROR|QUESTION|INFO --body "..." [--project P] [--task T] [--root R]`
Auto-discover bus: `run-agent bus discover --from <dir>`

[2026-02-21 08:22:23] [tags: user-docs, cli, bus]
`run-agent bus post` bus path resolution order: `--bus` flag → `MESSAGE_BUS` env → `--project`/`--task` path resolution → auto-discovery via `run-agent bus discover`

[2026-02-23 00:00:00] [tags: user-docs, cli, monitor]
`run-agent monitor` (filesystem mode): `--root R --project P --todo TODOs.md --agent A [--once] [--dry-run] [--interval D] [--stale-after D] [--rate-limit N]`
Requires `--project`; start/resume/recover actions require `--agent` or resolvable config.

[2026-02-23 00:00:00] [tags: user-docs, cli, wrap]
Shell wrapping: `run-agent shell-setup install|uninstall` — manages shell aliases for `claude`/`codex`/`gemini` through `run-agent wrap`

[2026-02-22 12:41:54] [tags: user-docs, cli, scripts]
Startup wrappers:
  `./scripts/start-conductor.sh --config config.yaml --root ./runs`
  `./scripts/start-run-agent-monitor.sh --config config.yaml --root ./runs [--background]`
  `./scripts/start-monitor.sh --root ./runs [--enable-task-start]`

[2026-02-22 12:41:54] [tags: user-docs, cli, scripts]
Background mode startup: `./scripts/start-conductor.sh --background --pid-file ./runs/conductor.pid --log-file ./runs/conductor.log`

[2026-02-22 12:41:54] [tags: user-docs, cli, scripts]
`scripts/start-run-agent-monitor.sh` resolves `--root` from: `$RUN_AGENT_MONITOR_ROOT` → `$CONDUCTOR_ROOT` → `$HOME/run-agent`
`scripts/start-monitor.sh` resolves `--root` from: `$RUN_AGENT_ROOT` → `$CONDUCTOR_ROOT` → `$RUNS_DIR` → repository `runs` dir

---

## CLI Reference: `run-agent server ...` (API-client commands)

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
`run-agent server` subcommands: `status`, `task`, `job`, `project`, `watch`, `bus`, `update`

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
`run-agent server task` subcommands: `status`, `list`, `runs`, `logs`, `stop`, `resume`, `delete`

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
`run-agent server job` subcommands: `submit` (supports `--attach-mode create|attach|resume`, `--wait`, `--follow`), `list`

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
`run-agent server project` subcommands: `list`, `stats`, `gc`, `delete`

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
`run-agent server bus` subcommands: `read` (supports `--follow` SSE streaming), `post`

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
`run-agent server update` subcommands: `status`, `start`

[2026-02-21 08:22:23] [tags: user-docs, cli, server-client]
Submit job via server: `run-agent server job submit --server http://127.0.0.1:14355 --project P --agent A --prompt-file f.md [--wait] [--follow]`

[2026-02-21 03:43:53] [tags: user-docs, cli, server-client]
Watch via server (defaults to all tasks when `--task` omitted): `run-agent server watch --project P [--task T] [--timeout 30m]`

---

## CLI Reference: `conductor` (server-centric CLI)

[2026-02-21 03:02:40] [tags: user-docs, cli, conductor]
`conductor` with no subcommand starts the API server. Global flags: `--config`, `--root`, `--host`, `--port`, `--disable-task-start`, `--version`, `--api-key`

[2026-02-21 03:02:40] [tags: user-docs, cli, conductor]
`conductor` subcommands: `status`, `task`, `job`, `project`, `watch`, `monitor`, `bus`, `workflow`, `goal`, `completion`

[2026-02-21 05:47:48] [tags: user-docs, cli, conductor]
`conductor task` subcommands: `stop`, `status`, `list` (`--status running|active|done|failed`), `delete`, `logs`, `runs`, `resume`

[2026-02-21 07:12:15] [tags: user-docs, cli, conductor]
`conductor task runs <task-id>`: list all runs with status, exit code, duration, agent version. Flags: `--project`, `--limit`, `--json`, `--server`

[2026-02-21 07:21:53] [tags: user-docs, cli, conductor]
`conductor job` subcommands: `submit` (`--project`, `--task` optional, `--agent`, `--prompt` or `--prompt-file`, `--attach-mode create|attach|resume`, `--wait`, `--follow`, `--depends-on`), `submit-batch`

[2026-02-21 07:21:53] [tags: user-docs, cli, conductor]
`conductor job submit` auto-generates task ID as `task-YYYYMMDD-HHMMSS-xxxxxx` (6-char random hex suffix) when `--task` is omitted.

[2026-02-21 07:21:53] [tags: user-docs, cli, conductor]
`conductor job submit --follow`: streams SSE output to stdout after submission; waits up to 30s for first run to start; reconnects on connection drop.

[2026-02-21 07:48:03] [tags: user-docs, cli, conductor]
`conductor project` subcommands: `list`, `stats`, `gc`, `delete`

[2026-02-21 07:48:03] [tags: user-docs, cli, conductor]
`conductor status [--server URL] [--json]`: shows version, uptime, active runs, configured agents.

[2026-02-21 03:43:53] [tags: user-docs, cli, conductor]
`conductor watch --project P [--task T] [--timeout 30m] [--interval 5s] [--json]`: watches task completion; defaults to all tasks in project when `--task` omitted.

[2026-02-21 08:05:23] [tags: user-docs, cli, conductor]
`conductor bus read --project P [--task T] [--follow]` — read or stream message bus via server
`conductor bus post --project P [--task T] --type TYPE --body "..."` — post message via server

[2026-02-23 14:50:19] [tags: user-docs, cli, conductor]
`conductor monitor --server URL --project P --todos TODOs.md --agent A [--interval D] [--stale-threshold D] [--rate-limit N]`
Defaults `--agent` to `claude`; runs continuous loop only (no `--once`/`--dry-run`).

[2026-02-21 03:43:53] [tags: user-docs, cli, conductor]
`conductor project gc`: server-based GC. `conductor project delete --project P`: delete project via server.

---

## API Reference

[2026-02-05 17:28:15] [tags: user-docs, api]
Base URL: `http://localhost:14355/api/v1`

[2026-02-05 17:28:15] [tags: user-docs, api]
Two API surfaces:
  1. `/api/v1/...` — primary REST API
  2. `/api/projects/...` — project-centric API used by web UI

[2026-02-05 17:28:15] [tags: user-docs, api]
`GET /api/v1/health` → `{"status":"ok","version":"dev"}`

[2026-02-05 17:28:15] [tags: user-docs, api]
`GET /api/v1/version` → `{"version":"dev"}`

[2026-02-21 05:16:21] [tags: user-docs, api, metrics]
`GET /metrics` → Prometheus text format (content-type: `text/plain; version=0.0.4`)
Metrics exposed:
  - `conductor_uptime_seconds` (gauge)
  - `conductor_active_runs_total` (gauge)
  - `conductor_completed_runs_total` (counter)
  - `conductor_failed_runs_total` (counter)
  - `conductor_queued_runs_total` (gauge — queue depth for `max_concurrent_runs`)
  - `conductor_messagebus_appends_total` (counter)
  - `conductor_api_requests_total{method, status}` (counter)

[2026-02-23 00:00:00] [tags: user-docs, api, self-update]
`GET /api/v1/admin/self-update` → `{"state":"idle","active_runs_now":0}`
States: `idle`, `deferred`, `applying`, `failed`
`POST /api/v1/admin/self-update` body: `{"binary_path":"/absolute/path/to/run-agent"}` → `202 Accepted`

[2026-02-05 17:28:15] [tags: user-docs, api, tasks]
`POST /api/v1/tasks` — create task
Required fields: `project_id`, `task_id`, `agent_type`, `prompt`
Optional: `attach_mode` (`create|attach|resume`), `config`, `depends_on`, `thread_parent`, `thread_message_type`
Response 201: `{"project_id","task_id","run_id","status":"started|queued","queue_position","depends_on":[]}`

[2026-02-05 17:28:15] [tags: user-docs, api, tasks]
`GET /api/v1/tasks` — list all tasks; optional `?project_id=` filter
`GET /api/v1/tasks/:taskId?project_id=P` — get task detail with runs

[2026-02-05 17:28:15] [tags: user-docs, api, messages]
`GET /api/v1/messages?project_id=P[&task_id=T]` — list message bus messages
`POST /api/v1/messages` — post message to bus

[2026-02-05 17:28:15] [tags: user-docs, api, sse]
SSE streams:
  - `GET /api/v1/messages/stream?project_id=P[&task_id=T]`
  - `GET /api/projects/{p}/messages/stream`
  - `GET /api/projects/{p}/tasks/{t}/messages/stream`
  - `GET /api/projects/{p}/tasks/{t}/runs/stream` — fan-in all runs of a task
  - `GET /api/projects/{p}/tasks/{t}/runs/{r}/stream?name=output.md` — stream growing file

[2026-02-05 17:28:15] [tags: user-docs, api, project-centric]
Project-centric API endpoints:
  - `GET /api/projects` — list projects
  - `POST /api/projects` — create project
  - `GET /api/projects/home-dirs` — known project home/work folders
  - `GET /api/projects/{p}/tasks` — list tasks
  - `GET /api/projects/{p}/tasks/{t}` — task detail
  - `GET /api/projects/{p}/tasks/{t}/runs/{r}` — run detail
  - `GET /api/projects/{p}/tasks/{t}/runs/{r}/file?name=output.md` — read run file
  - `POST /api/projects/{p}/tasks/{t}/runs/{r}/stop` — stop run (202 SIGTERM sent; 409 not running)
  - `DELETE /api/projects/{p}/tasks/{t}/runs/{r}` — delete run (204; 409 if still running)
  - `DELETE /api/projects/{p}/tasks/{t}` — delete task (204; 409 if running; 404 not found)
  - `GET /api/projects/{p}/stats` — project statistics
  - `GET/POST /api/projects/{p}/messages` — project-level bus
  - `GET/POST /api/projects/{p}/tasks/{t}/messages` — task-level bus
  - `POST /api/projects/{p}/tasks/{t}/resume` — remove DONE file (200; 404; 400 if no DONE)
  - `GET /api/projects/{p}/tasks/{t}/file?name=TASK.md` — read TASK.md

[2026-02-23 00:00:00] [tags: user-docs, api, security]
API handlers reject path traversal for project/task/run resources; enforce confinement within configured root directories.

[2026-02-23 00:00:00] [tags: user-docs, api, security]
Web UI browser-origin destructive calls (delete + project GC) are rejected with `403 Forbidden` by server middleware.

[2026-02-23 00:00:00] [tags: user-docs, api, correlation]
API responses include `X-Request-ID` header. Clients can provide their own request ID for end-to-end tracing.

[2026-02-22 15:26:16] [tags: user-docs, api, audit]
Form submission audit log: `<root>/_audit/form-submissions.jsonl`. Each line includes: `timestamp`, `request_id`, `correlation_id`, `method`, `path`, `endpoint`, `remote_addr`, `project_id`, `task_id`, `run_id`, `message_id`, `payload` (sanitized).
Logged endpoints: `POST /api/v1/tasks`, `POST /api/projects`, `POST /api/projects/{p}/messages`, `POST /api/projects/{p}/tasks/{t}/messages`, `POST /api/v1/messages`

---

## Web UI Features

[2026-02-05 17:28:15] [tags: user-docs, web-ui]
Primary web UI: React 18 + TypeScript app built in `frontend/` served from `frontend/dist/`. Falls back to embedded `web/src/` assets when `frontend/dist/` not present.

[2026-02-05 17:28:15] [tags: user-docs, web-ui]
Web UI main views: Task List, Run Details, Message Bus, Run Tree visualization.

[2026-02-05 17:28:15] [tags: user-docs, web-ui]
Run Detail tabs: `TASK.MD`, `OUTPUT` (default; falls back to agent-stdout.txt), `STDOUT` (JSON/thinking block rendering), `STDERR`, `PROMPT`, `MESSAGES`

[2026-02-05 17:28:15] [tags: user-docs, web-ui]
Agent heartbeat badge based on recent activity in `agent-stdout.txt`:
  - `● LIVE` (green): output within last 60 seconds
  - `● STALE` (yellow): no output 1–5 minutes
  - `● SILENT` (red): no output more than 5 minutes

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Task list auto-refreshes every 5 seconds.

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Task search bar filters by ID substring (case-insensitive); shows "Showing N of M tasks".

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Task status colors: Running=Yellow, Queued=Light blue, Blocked=Orange, Success=Green, Failed=Red, Unknown=Gray.

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Message types usable in compose form: `USER`, `FACT`, `PROGRESS`, `DECISION`, `ERROR`, `QUESTION`
Auto-emitted runner messages: `RUN_START`, `RUN_STOP`, `RUN_CRASH`

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Compose form scope mapping:
  - Project scope → `POST /api/projects/{p}/messages` → `<root>/<project>/PROJECT-MESSAGE-BUS.md`
  - Task scope → `POST /api/projects/{p}/tasks/{t}/messages` → `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Stop Agent: button sends SIGTERM via `POST /api/projects/{p}/tasks/{t}/runs/{r}/stop`
Resume Task: button calls `POST /api/projects/{p}/tasks/{t}/resume` (removes DONE file)

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Web UI keyboard shortcuts: `Ctrl+R` refresh, `Ctrl+F` search, `Esc` close modal, `↑/↓` navigate, `Enter` open, `Space` pause/resume auto-scroll

[2026-02-23 00:00:00] [tags: user-docs, web-ui, dev]
Frontend dev: `cd frontend && npm install && npm run dev` → Vite dev server on `http://localhost:5173`
Add CORS origin `http://localhost:5173` to `api.cors_origins` in config.

[2026-02-23 00:00:00] [tags: user-docs, web-ui]
Supported browsers: Chrome 90+, Firefox 88+, Safari 14+, Edge 90+. IE not supported.

---

## Message Bus Protocol

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Message bus: append-only YAML event log with `---` document separators. Format: two YAML documents per message — header (metadata) + body (content).

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Message ID format: `MSG-{YYYYMMDD-HHMMSS}-{NANOSECONDS}-PID{PID}-{SEQUENCE}`
Example: `MSG-20260205-143052-000123456-PID12345-0001`

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Message struct fields: `msg_id`, `ts` (UTC), `type`, `project_id`, `task_id`, `run_id`, `parents` (optional threading), `attachment_path` (optional), body (separate YAML document).

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Concurrency model: lockless reads (never block writers/readers); exclusive writes via `flock` (Unix advisory) / `LockFileEx` (Windows mandatory). Write lock timeout default: 10 seconds.

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Write throughput: ~37,000+ messages/sec measured with 10 concurrent writers on macOS (writes go to OS page cache, no fsync).

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Poll interval for new messages: 200ms default (configurable via `WithPollInterval`).

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Auto-rotation: `WithAutoRotate(maxBytes int64)` renames bus file to `<path>.YYYYMMDD-HHMMSS.archived` when write would exceed threshold.

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
`ReadLastN(n int)`: efficient tail-only reads using 64KB seek window (doubles up to 3× before full read fallback).

[2026-02-21 00:45:47] [tags: dev-docs, message-bus, platform]
Windows limitation: `LockFileEx` mandatory locks may block readers; use WSL2 for full Unix advisory flock semantics.

[2026-02-21 00:45:47] [tags: dev-docs, message-bus]
Network filesystem warning: O_APPEND + flock may not work on NFS, SMB, or distributed filesystems. Use local storage.

[2026-02-21 03:19:08] [tags: dev-docs, message-bus]
Threaded task answer linkage fields in `TASK-THREAD-LINK.yaml`: `parent_project_id`, `parent_task_id`, `parent_run_id`, `parent_message_id`. Child bus message type: `USER_REQUEST`. Parent source types allowed: `QUESTION`, `FACT`.

---

## Developer Documentation

[2026-02-05 17:28:15] [tags: dev-docs, build]
Build commands: `go build -o bin/ ./cmd/...` or per-binary: `go build -o bin/conductor ./cmd/conductor` / `go build -o bin/run-agent ./cmd/run-agent`

[2026-02-05 17:28:15] [tags: dev-docs, test]
Test commands: `go test ./...` / `go test -race ./internal/... ./cmd/...`

[2026-02-05 17:28:15] [tags: dev-docs, test]
Integration tests: `ACCEPTANCE=1 go test ./test/acceptance/...`

[2026-02-05 17:28:15] [tags: dev-docs, test]
Test coverage check: `make test-coverage` (enforces >= 60% threshold). Override: `COVERAGE_THRESHOLD=75 make test-coverage`

[2026-02-05 17:28:15] [tags: dev-docs, test]
Startup script integration tests: `go test ./test/integration -run StartupScript -count=1`

[2026-02-05 17:28:15] [tags: dev-docs, lint]
Lint: `golangci-lint run`

[2026-02-23 00:00:00] [tags: dev-docs, observability]
Structured runtime logs via `internal/obslog`: `key=value` (logfmt) format with fields `ts`, `level`, `subsystem`, `event`, `project_id`, `task_id`, `run_id`, `request_id`, `correlation_id`.

[2026-02-23 00:00:00] [tags: dev-docs, observability]
Log redaction: token-like key names (`token`, `api_key`, `authorization`, `secret`, `password`) and pattern-based bearer/JWT masking. Message bus body payloads never logged.

[2026-02-23 00:00:00] [tags: dev-docs, docs-site]
Documentation site: Hugo in Docker only. Commands:
  `./scripts/docs.sh serve` → local preview at `http://localhost:1313/`
  `./scripts/docs.sh build` → static files in `website/public/`
  `./scripts/docs.sh verify` → check generated pages and key links
Env overrides: `DOCKER_UID`, `DOCKER_GID`, `HUGO_BASE_URL`

---

## Agent Integration Requirements

[2026-02-23 00:00:00] [tags: user-docs, agents]
Agent runtime requirements:
  - Claude: `claude` CLI in PATH. [Claude CLI](https://claude.ai/code)
  - Codex: `codex` CLI in PATH. [OpenAI Codex](https://github.com/openai/codex)
  - Gemini: `gemini` CLI in PATH. [Gemini CLI](https://github.com/google-gemini/gemini-cli)
  - Perplexity: REST API token only (no CLI required)
  - xAI: REST API token only (no CLI required)

[2026-02-21 04:15:37] [tags: user-docs, agents]
CLI version floors (warning-only if older): `claude >= 1.0.0`, `codex >= 0.1.0`, `gemini >= 0.1.0`

---

## RLM Orchestration

[2026-02-21 20:19:09] [tags: user-docs, rlm]
RLM activation threshold: context > 50K tokens → use RLM; context > 16K AND multi-hop → use RLM; files > 5 → use RLM.

[2026-02-21 20:19:09] [tags: user-docs, rlm]
RLM six-step protocol: 1. ASSESS (scope), 2. DECIDE (strategy), 3. DECOMPOSE (boundaries), 4. EXECUTE (parallel sub-agents with `&`/`wait`), 5. SYNTHESIZE (collect outputs), 6. VERIFY (`go test ./...`, `go build ./...`).

[2026-02-21 20:19:09] [tags: user-docs, rlm]
Target sub-task context window: 4K–10K tokens per sub-agent.

[2026-02-21 20:19:09] [tags: user-docs, rlm, cli]
Spawn sub-agent with parent tracking:
  `run-agent job --project $JRUN_PROJECT_ID --root $CONDUCTOR_ROOT --agent claude --parent-run-id $JRUN_ID --timeout 30m --prompt "..." &`
  Always use `wait` after backgrounded spawns before synthesizing.

[2026-02-21 20:19:09] [tags: user-docs, rlm, cli]
Read sub-agent output: `run-agent output --project P --task <sub-task-id> --root R`
List runs: `run-agent list --project P --root R`

---

## Troubleshooting Tips

[2026-02-20 15:42:12] [tags: user-docs, troubleshooting]
Port already in use: `lsof -i :14355` (macOS/Linux) or `netstat -ano | findstr :14355` (Windows). Change port via `api.port` in config.yaml.

[2026-02-23 00:00:00] [tags: user-docs, troubleshooting, windows]
Windows message bus blocking: mandatory `LockFileEx` locks may block concurrent readers. Recommendation: use WSL2 for full Unix semantics.

[2026-02-23 00:00:00] [tags: user-docs, troubleshooting]
503 Service Unavailable from `POST /api/v1/tasks`: server started with `--disable-task-start`. Remove flag or unset `CONDUCTOR_DISABLE_TASK_START`.

[2026-02-23 00:00:00] [tags: user-docs, troubleshooting]
Debug logging: `CONDUCTOR_LOG_LEVEL=debug conductor --config config.yaml`

[2026-02-23 00:00:00] [tags: user-docs, troubleshooting]
Token file security: `chmod 600 ~/.conductor/tokens/*.token` and never use inline `token:` in config.yaml.

[2026-02-23 00:00:00] [tags: user-docs, troubleshooting]
Codex token format: starts with `sk-`. Claude token format: starts with `sk-ant-`.

[2026-02-23 00:00:00] [tags: user-docs, troubleshooting]
Message bus compose disabled or stale: use `run-agent bus read/post` CLI as source of truth, or switch scope (Task/Project) to force stream reset.

---

## Platform Support

[2026-02-23 00:00:00] [tags: user-docs, platform]
macOS: Fully supported (primary development platform).
Linux: Fully supported (all features).
Windows: Limited — message bus mandatory file locks block concurrent readers; in-place self-update handoff unsupported. Use WSL2 for full compatibility.

---
