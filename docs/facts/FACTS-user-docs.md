> **NOTE:** This file is a historical log of facts extracted from the codebase.
> For the most up-to-date and reconciled information, please refer to [FACTS-reconciled.md](FACTS-reconciled.md).
> Entries marked with `[corrected]` or `[outdated]` have been superseded.

# FACTS: User & Developer Documentation

## Validation Round 2 (codex)

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent]
`./bin/run-agent --help` currently lists top-level commands: `bus`, `completion`, `gc`, `goal`, `help`, `job`, `list`, `monitor`, `output`, `resume`, `serve`, `server`, `shell-setup`, `status`, `stop`, `task`, `validate`, `watch`, `workflow`, `wrap`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent]
`./bin/run-agent iterate` currently fails with: `Error: unknown command "iterate" for "run-agent"`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-task]
`run-agent task` subcommands are `delete` and `resume`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-task]
`run-agent task` flags include `--depends-on`, `--dependency-poll-interval` (default `2s`), `--restart-delay` (default `1s`), and `--timeout` (default `0`, meaning no limit).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-job]
`run-agent job` subcommand is `batch`; both `job` and `job batch` support `--timeout` (default `0`, no idle-output timeout limit) and `-f/--follow`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-bus]
`run-agent bus` subcommands are `discover`, `post`, and `read`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-bus]
`run-agent bus discover` upward-search order per directory is: `TASK-MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`, `MESSAGE-BUS.md`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-bus]
`run-agent bus post` bus path resolution order is: `--bus` -> `MESSAGE_BUS` env -> `--project`/`--task` path resolution -> auto-discovery.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-bus]
`run-agent bus read` bus path resolution order is: `--project`/`--task` path resolution -> `--bus` -> `MESSAGE_BUS` env -> auto-discovery; using both `--bus` and `--project` is an error.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-bus]
`run-agent bus post` default message type is `INFO` (`--type` default).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-list]
`run-agent list` default root is `RUNS_DIR` env when set, otherwise `./runs`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-status]
`run-agent status` default root is `RUNS_DIR` env when set, otherwise `./runs`; `--project` is required.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-watch]
`run-agent watch --help` advertises optional repeated `--task`, but current implementation (`cmd/run-agent/watch.go`) returns an error unless at least one `--task` is provided.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-gc]
`run-agent gc` defaults: `--older-than 168h`, `--bus-max-size 10MB`, root from `RUNS_DIR` else `./runs`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-serve]
`run-agent serve` defaults: `--host 0.0.0.0`, `--port 14355`; flags include `--api-key` and `--disable-task-start`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-workflow]
`run-agent workflow run` defaults: `--template THE_PROMPT_v5`, `--to-stage 12`, `--timeout 0`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-goal]
`run-agent goal decompose` defaults: `--strategy rlm`, `--max-parallel 6`; output format defaults to YAML unless `--json` is set.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-output]
`run-agent output --file` accepted values are `output` (default), `stdout`, `stderr`, `prompt`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-resume]
`run-agent resume` requires `--project` and `--task`; default root is `./runs`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-stop]
`run-agent stop --force` sends `SIGKILL` when graceful stop exceeds an internal `30s` timeout (`stopTimeout`).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-shell-setup]
`run-agent shell-setup` manages aliases for `claude`, `codex`, and `gemini`; `install` accepts `--run-agent-bin`, `--shell`, `--rc-file`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, run-agent-server]
`run-agent server` subcommands are `status`, `task`, `job`, `project`, `watch`, `bus`, `update`; client default server URL is `http://localhost:14355`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, conductor-binary]
`./bin/conductor --help` shows global defaults `--host 0.0.0.0` and `--port 14355` (rebuilt 2026-02-24).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, conductor-binary]
`./bin/conductor --help` lists: `bus`, `completion`, `goal`, `help`, `job`, `monitor`, `project`, `status`, `task`, `watch`, `workflow`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, conductor-binary]
`./bin/conductor status --help` and `./bin/conductor watch --help` default `--server` to `http://localhost:14355`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, cli, conductor-source]
Source code (`cmd/conductor/main.go`, `cmd/conductor/status.go`, `cmd/conductor/watch.go`) defaults conductor port/server URL to `14355` and includes `goal`, `workflow`, and `monitor` commands.

[2026-02-24 08:48:00] [tags: user-docs, dev-docs, cli, conductor-binary]
Binary-vs-source drift resolved: `bin/conductor` rebuilt from source, now matches source defaults (port `14355`, full command set including `goal`, `workflow`, `monitor`).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, discovery]
Default config discovery (`internal/config.FindDefaultConfig`) order is: `./config.yaml`, `./config.yml`, `./config.hcl`, `$HOME/.config/conductor/config.yaml`, `$HOME/.config/conductor/config.yml`, `$HOME/.config/conductor/config.hcl`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, discovery]
`run-agent serve` and `conductor` resolve config in order: `--config` flag -> `CONDUCTOR_CONFIG` env -> `FindDefaultConfig` search.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, schema]
`internal/config.AgentConfig` fields are `type`, `token`, `token_file`, `base_url`, `model`; there is no per-agent `timeout` field in the runtime schema.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, schema]
`defaults` schema fields are `agent`, `timeout`, `max_concurrent_runs`, `max_concurrent_root_tasks`, and optional `diversification`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, validation]
Validation requires `defaults.timeout > 0` and `defaults.max_concurrent_root_tasks >= 0`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, api]
API config runtime defaults are: `host=0.0.0.0`, `port=14355`, `sse.poll_interval_ms=100`, `sse.discovery_interval_ms=1000`, `sse.heartbeat_interval_s=30`, `sse.max_clients_per_run=10`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, env]
`CONDUCTOR_API_KEY` env override sets `api.api_key` and forces `api.auth_enabled=true`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, env]
Per-agent token env override name format is `CONDUCTOR_AGENT_<AGENT_NAME>_TOKEN` (non-alphanumeric chars normalized to `_`).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, config, format]
Both YAML and HCL are supported; `.hcl` is selected by file extension.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, server, port]
API server runtime default port is `14355`; when the configured/default port is not explicit, server attempts up to 100 consecutive ports (`basePort..basePort+99`) before failing.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, routes]
Top-level routes in `internal/api/routes.go`: `/metrics`, `/api/v1/health`, `/api/v1/version`, `/api/v1/status`, `/api/v1/admin/self-update`, `/api/v1/runs/stream/all`, `/api/v1/tasks`, `/api/v1/tasks/`, `/api/v1/runs`, `/api/v1/runs/`, `/api/v1/messages`, `/api/v1/messages/stream`, `/api/projects`, `/api/projects/home-dirs`, `/api/projects/`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, v1]
`/api/v1/tasks` supports `GET` (list) and `POST` (create); `/api/v1/tasks/{taskId}` supports `GET` and `DELETE`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, v1]
`/api/v1/runs/{runId}` supports `GET`; `/api/v1/runs/{runId}/info` supports `GET`; `/api/v1/runs/{runId}/stop` supports `POST`; `/api/v1/runs/{runId}/stream` supports `GET`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, v1]
`/api/v1/messages` supports `GET`; `POST /api/v1/messages` is explicitly routed; `/api/v1/messages/stream` supports `GET` SSE.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, projects]
`/api/projects` supports `GET` and `POST`; `/api/projects/{projectId}` supports `GET` and `DELETE`; `/api/projects/{projectId}/stats` supports `GET`; `/api/projects/{projectId}/gc` supports `POST`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, projects]
Project task routes include: `GET /api/projects/{p}/tasks`, `GET|DELETE /api/projects/{p}/tasks/{t}`, `POST /api/projects/{p}/tasks/{t}/resume`, `GET /api/projects/{p}/tasks/{t}/file?name=TASK.md`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, projects]
Project run routes include: `GET /api/projects/{p}/tasks/{t}/runs`, `GET /api/projects/{p}/tasks/{t}/runs/stream`, `GET|DELETE /api/projects/{p}/tasks/{t}/runs/{r}`, `POST /api/projects/{p}/tasks/{t}/runs/{r}/stop`, `GET /api/projects/{p}/tasks/{t}/runs/{r}/file`, `GET /api/projects/{p}/tasks/{t}/runs/{r}/stream`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, messages]
Project/task bus routes include: `GET|POST /api/projects/{p}/messages`, `GET /api/projects/{p}/messages/stream`, `GET|POST /api/projects/{p}/tasks/{t}/messages`, `GET /api/projects/{p}/tasks/{t}/messages/stream`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, self-update]
`/api/v1/admin/self-update` supports `GET` status and `POST` request; state constants are `idle`, `deferred`, `applying`, `failed`; successful POST returns HTTP `202 Accepted`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, auth]
API key auth accepts `Authorization: Bearer <key>` and `X-API-Key: <key>`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, auth]
Auth-exempt paths are exactly `/api/v1/health`, `/api/v1/version`, `/metrics`, and paths with prefix `/ui/`; `OPTIONS` is always allowed through auth middleware.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, correlation]
`X-Request-ID` is always set on responses by middleware; incoming `X-Request-ID` is reused when provided.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, security]
Path-safety checks reject project/task/run identifiers containing `/`, `\\`, or `..` (including URL-decoded forms) and enforce filesystem confinement with `requirePathWithinRoot`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, api, security]
Browser/UI-origin destructive actions are blocked with `403` for project deletion, task deletion, run deletion, and project GC (`rejectUIDestructiveAction`).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, metrics, api]
Metrics names emitted include: `conductor_uptime_seconds`, `conductor_active_runs_total`, `conductor_completed_runs_total`, `conductor_failed_runs_total`, `conductor_messagebus_appends_total`, `conductor_queued_runs_total`, `conductor_api_requests_total`, plus `conductor_agent_runs_total` and `conductor_agent_fallbacks_total` when populated.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, message-bus, format]
Message ID format is `MSG-YYYYMMDD-HHMMSS-<9-digit nanoseconds>-PID<5-digit pid>-<4-digit seq>` (`internal/messagebus/msgid.go`).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, message-bus, defaults]
Message bus defaults: exclusive lock timeout `10s`, poll interval `200ms`, append retries `3`, retry backoff base `100ms` (exponential).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, installation, script]
`install.sh` defaults: mirror base `https://run-agent.jonnyzzz.com/releases/latest/download`, fallback `https://github.com/jonnyzzz/conductor-loop/releases/latest/download`, install dir `/usr/local/bin`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, installation, script]
`install.sh` verifies SHA256 using `<asset>.sha256` and supports overrides `RUN_AGENT_DOWNLOAD_BASE`, `RUN_AGENT_FALLBACK_DOWNLOAD_BASE`, and `RUN_AGENT_INSTALL_DIR`.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, installation, launcher]
`run-agent.cmd` binary resolution precedence is: `RUN_AGENT_BIN` -> sibling `run-agent`/`run-agent.exe` -> `dist/run-agent-<os>-<arch>` (or Windows dist exe) -> PATH (unless `RUN_AGENT_CMD_DISABLE_PATH=1`).

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, docs, drift]
Current docs drift examples observed during validation: `docs/user/configuration.md` still describes `~/.conductor/config.yaml` and per-agent `timeout`, while runtime code uses `~/.config/conductor/*` and does not define `agents.<name>.timeout`.

[2026-02-24 08:30:00] [tags: user-docs, dev-docs, docs, drift, corrected]
Correction: Repository `go.mod` requires `go 1.24.0`. Documentation referencing `1.21+` is outdated and must be updated. Config path is `~/.config/conductor/`. `run-agent iterate`, `output synthesize`, and `review quorum` are currently unavailable in the binary (confirmed via help output), despite some stale `todos.md` entries misreporting them as closed. See FACTS-reconciled.md.

[2026-02-23 19:21:02] [tags: user-docs, dev-docs, docs, git-history]
Latest docs-related commit in sampled history is `493ab3832d9e2c40bee1c0922cd2bd5441f6fd31` at `2026-02-23 14:50:19` (`feat(cli): add TODO-driven monitor commands`).

## Reconciliation (2026-02-24)

[2026-02-24 07:45:00] [tags: reconciliation, user-docs, port]
Server Port: `run-agent serve` source defaults to `14355`. `14355` is the correct canonical default.

[2026-02-24 09:50:00] [tags: reconciliation, user-docs, port, resolved]
Binary drift resolved: `bin/conductor` rebuilt from source on 2026-02-24. `./bin/conductor --help` now shows `--port int ... (default 14355)` and includes all commands (`goal`, `monitor`, `workflow`). Binary-vs-source mismatch is no longer present.

[2026-02-24 07:45:00] [tags: reconciliation, user-docs, iterate]
`run-agent iterate`: Command is missing from `bin/run-agent` ("unknown command") despite task logs claiming implementation. This is a known discrepancy; the command is effectively unavailable in the current binary.

[2026-02-24 10:10:00] [tags: reconciliation, user-docs, go-version]
Go Version: Repository `go.mod` requires `go 1.24.0`. Documentation referencing `1.21+` was outdated and has been updated to `1.24.0` across `docs/`, `examples/`, and `prompts/`.

[2026-02-24 07:45:00] [tags: reconciliation, user-docs, config-path]
Config Path: Runtime code uses `~/.config/conductor/` (Standard XDG-like path) and local `./config.*`. Documentation referencing `~/.conductor/` is outdated.

[2026-02-24 08:30:00] [tags: reconciliation, workflow, prompts]
Workflow prompt files moved: `THE_PROMPT_v5*.md` and `THE_PLAN_v5.md` moved from root to `docs/workflow/`. CLI code references to root paths (e.g. `internal/runner/orchestrator.go` help text injection) are now stale and need update to use `docs/workflow/` prefix.

[2026-02-24 09:00:00] [tags: reconciliation, workflow, prompts, xref-fix]
Stale CLI code references fixed (iteration 4): `internal/runner/orchestrator.go` now looks for `THE_PROMPT_v5_conductor.md` in `docs/workflow/`. `internal/goaldecompose/spec.go` now uses `docs/workflow/THE_PROMPT_v5.md` for `thePromptSemanticsDocument` and `docs/workflow/THE_PROMPT_v5_<role>.md` for all `rolePrompt` blueprint values. `Instructions.md` moved to `docs/dev/instructions.md` (staged rename). Root-only .md files remaining: `README.md`, `AGENTS.md`, `CLAUDE.md`, `MESSAGE-BUS.md`, `output.md`.

## Documentation Structure (2026-02-24)

[2026-02-24 09:30:00] [tags: docs, structure, inventory]
Current documentation layout verified:
- **docs/workflow/**: \`THE_PLAN_v5.md\`, \`THE_PROMPT_v5.md\`, \`THE_PROMPT_v5_*.md\` (role prompts).
- **docs/dev/**:
  - \`docs/dev/development.md\` (General guide; migrated from historical root `DEVELOPMENT.md`)
  - \`docs/dev/architecture.md\`, \`docs/dev/architecture-review.md\`, \`docs/dev/dependency-analysis.md\` (Design; includes historical root docs `ARCHITECTURE-REVIEW-SUMMARY.md` and `DEPENDENCY_ANALYSIS.md`)
  - \`issues.md\`, \`questions.md\`, \`todos.md\`, \`implementation-status.md\` (Tracking)
  - \`instructions.md\` (Tooling paths)
  - \`output-examples.md\` (Artifact references)
  - \`agent-protocol.md\`, \`adding-agents.md\`, \`message-bus.md\`, \`logging-observability.md\` (Subsystems)
