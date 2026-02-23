# Conductor Loop - Agent Orchestration Framework

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

**Conductor Loop** is an agent orchestration framework for running AI agents (Claude, Codex, Gemini, Perplexity, xAI) with Ralph Loop task management, hierarchical task execution, live log streaming, and a web UI for monitoring.

## Features

- **Multiple AI Agents**: Support for Claude, Codex, Gemini, Perplexity, and xAI (current runner execution path: Claude/Codex/Gemini via CLI; Perplexity/xAI via built-in REST backends)
- **Ralph Loop**: Automatic task restart and recovery on failure
- **Run Concurrency Limit**: Optional global run semaphore via `defaults.max_concurrent_runs` (`0` means unlimited)
- **Agent Diversification Policy**: Optional `defaults.diversification` policy can distribute runs across configured agents (`round-robin` or `weighted`) with optional single-step fallback retry via `fallback_on_failure`
- **Root-Task Planner Queue**: Server-side deterministic FIFO scheduling for API/UI-submitted root tasks when `defaults.max_concurrent_root_tasks > 0` (configurable root-task concurrency)
- **Hierarchical Task Lineage**: Parent-child task/run relationships with `parent_run_id` propagation across run metadata and project APIs
- **Task Dependencies (`depends_on`)**: Declarative task dependencies via `--depends-on` flags (`run-agent task`, `run-agent server job submit`, `conductor job submit`, `conductor job submit-batch`) with cycle checks and runtime dependency gating
- **Batch Job Submission**: Local sequential batching via `run-agent job batch` and server-side batch submission via `conductor job submit-batch`
- **Live Log Streaming**: Real-time SSE-based log streaming via REST API
- **Web UI**: React dashboard (`frontend/dist`) with message bus compose form, stop/resume controls, JSON-aware message rendering, thinking block rendering, and heartbeat indicators; when `frontend/dist` is unavailable, server falls back to embedded `web/src` assets for baseline monitoring
- **Message Bus**: Cross-task communication with GET/POST plus SSE stream endpoints (`/api/v1/messages/stream?project_id=...` with optional `task_id=...`, `/api/projects/{project}/messages/stream`, `/api/projects/{project}/tasks/{task}/messages/stream`); compose form in web UI, CLI auto-discovery via `run-agent bus discover` (supports `--from` upward search), server-side streaming reads via `run-agent server bus read --follow`, and context-aware posting (`run-agent bus post` resolves bus path via `--bus`, `MESSAGE_BUS`, `--project`/`--task`, then auto-discovery; message scope fields `project_id`/`task_id`/`run_id` resolve via explicit flags, inferred context, then `JRUN_*` env vars)
- **Task Completion Fact Propagation**: When a task reaches `DONE`, `run-agent task` posts a deduplicated completion `FACT` summary from task scope to the project message bus (`PROJECT-MESSAGE-BUS.md`)
- **Run-State Liveness Healing**: Run status reads reconcile stale/dead PIDs; when task `DONE` exists, stale `running`/reconciled `failed` runs are promoted to `completed` with normalized exit code (`0`)
- **Task Resume**: Remove `DONE` markers via API/Web UI/CLI (`run-agent resume`, `run-agent server task resume`, `conductor task resume`); re-run stopped/failed task loops with `run-agent task resume`
- **Task Stop/Delete Controls**: Stop running tasks and delete task state via CLI/API (`run-agent stop`, `run-agent task delete`, `run-agent server task stop|delete`, `conductor task stop|delete`)
- **Project Lifecycle Controls**: Query and maintain project state via CLI/API (`run-agent gc --project ...`, `run-agent server project list|stats|gc|delete`, `conductor project list|stats|gc|delete`)
- **Web UI Safety Guardrails**: Browser-origin destructive actions are blocked in the web UI (run/task/project deletion and project GC); use CLI/API clients for those operations
- **Prometheus Metrics**: `/metrics` endpoint with uptime, active/completed/failed/queued run counts, message bus append totals, and API request counters
- **Form Submission Audit Log**: Accepted form submissions are appended as sanitized JSONL records in `<root>/_audit/form-submissions.jsonl`
- **Webhook Notifications**: Optional asynchronous `run_stop` delivery via `webhook` config (`url`, optional `events`, `secret`, `timeout`), with retry/backoff and optional HMAC signing (`X-Conductor-Signature`)
- **Safe Server Self-Update**: `/api/v1/admin/self-update` with `run-agent server update status` and `run-agent server update start --binary ...` supports deferred handoff when active root runs exist (in-place handoff is not supported on native Windows)
- **API Path Safety**: REST handlers reject project/task/run path traversal attempts and enforce path confinement within configured root directories
- **API Authentication**: Optional API key auth for server APIs (`run-agent serve --api-key ...` / `conductor --api-key ...`), with `/api/v1/health`, `/api/v1/version`, `/metrics`, and UI routes under `/ui/` kept publicly accessible for health checks and UI loading (use `/ui/` with a trailing slash)
- **Request Correlation**: API responses include `X-Request-ID`; clients can provide their own request ID for end-to-end tracing across API logs and audit records
- **Structured Observability Logs**: Centralized redacted `key=value` logging (`internal/obslog`) across startup, API, runner, message bus, and storage events
- **Storage**: Persistent run storage with structured logging; managed runs ensure `output.md` exists, and stream parsers extract assistant text for Claude/Codex/Gemini when available
- **Output Inspection**: `run-agent output` supports `--follow` for running jobs (tails live run files) plus `--tail` for quick run-output slicing; without `--follow`, `--file output` reads `output.md` first and falls back to `agent-stdout.txt` when `output.md` is missing, while `--follow --file output` tails `agent-stdout.txt`; `--file prompt` prints stored `prompt.md`
- **Activity Signals**: `run-agent list --activity` and `run-agent status --activity` surface latest bus/output signals and analysis-drift hints (tunable with `--drift-after`); `run-agent status --concise` emits tab-separated snapshots for scripting
- **TODO-Driven Task Monitoring**: `run-agent monitor` manages unchecked `TODOs.md` tasks in local filesystem mode (start/resume/recover stale/finalize) with `--interval`, `--stale-after`, and `--rate-limit` controls, and requires `--project` (start/resume/recover actions also require `--agent` or a resolvable config; TODO file flag is `--todo`; task IDs must match canonical `task-YYYYMMDD-HHMMSS-...` format); `conductor monitor` provides analogous TODO automation through server APIs with `--interval`, `--stale-threshold`, and `--rate-limit` controls (requires `--project`; TODO file flag is `--todos`; defaults `--agent` to `claude`; currently daemon-only, no `--once`/`--dry-run` flags)
- **CLI Modes**: `run-agent` for local filesystem workflows (no server required for local commands), `run-agent server ...` for API-client workflows against a running API server (`run-agent serve` or `conductor` server mode; `run-agent server job submit` supports `--wait` and `--follow`), and optional `conductor` (server-centric CLI) for server-first workflows; running `conductor` with no subcommand starts the API server, and subcommands cover `status`, `task`, `job`, `project`, `watch`, `monitor`, `bus`, `workflow`, `goal`, and Cobra-generated `completion`
- **Shell Wrapping Helpers**: `run-agent shell-setup install|uninstall` manages shell aliases for `claude`/`codex`/`gemini` through `run-agent wrap`; `run-agent wrap --task-prompt` seeds `TASK.md` when wrap creates a new task directory
- **PATH Injection (Deduplicated)**: The current `run-agent`/`conductor` executable directory is prepended to PATH only when missing
- **Docker Support**: Full containerization with docker-compose

## Quick Start

Use one of these startup paths:

```bash
# Option A: install latest released run-agent binary
curl -fsSL https://run-agent.jonnyzzz.com/install.sh | bash
```

The installer downloads a release asset (mirror first, GitHub fallback) and supports Linux/macOS. It installs `run-agent` only.

For development branches and source parity, build from source:

```bash
# 1. Clone and build
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
go build -o run-agent ./cmd/run-agent
go build -o conductor ./cmd/conductor

# 2. Configure (edit config.yaml)
cat > config.yaml <<EOF
agents:
  codex:
    type: codex
    token_file: ./tokens/codex.token

defaults:
  agent: codex
  timeout: 300
  max_concurrent_runs: 4
  max_concurrent_root_tasks: 2

storage:
  runs_dir: ./runs
EOF

# 3. Start the full conductor server (one command)
./scripts/start-conductor.sh --config config.yaml --root ./runs

# 4. Or start monitor-only mode (no task execution)
./scripts/start-run-agent-monitor.sh --config config.yaml --root ./runs

# 5. Open the web UI (default port: 14355)
# macOS:
open http://localhost:14355/ui/
# Linux:
xdg-open http://localhost:14355/ui/

# 6. Submit a task to the running server and stream output until completion
# Use ./run-agent for the locally built binary.
# If you installed via installer, replace ./run-agent with run-agent.
./run-agent server job submit \
  --server http://127.0.0.1:14355 \
  --project my-project \
  --agent codex \
  --task task-20260222-154500-demo \
  --prompt "Run a smoke test task and summarize results" \
  --follow
```

Optional source-checkout verification before running workflows:

```bash
go test ./...
golangci-lint run   # if installed
```

`scripts/start-conductor.sh` prefers a `conductor` executable when one is found and falls back to `run-agent serve` only when no `conductor` binary is found.
`scripts/start-conductor.sh` and `scripts/start-run-agent-monitor.sh` require a config file path (explicit `--config` or successful auto-discovery); use `scripts/start-monitor.sh` for config-optional monitor mode.

### Two Supported Working Scenarios

Conductor Loop supports the same task lifecycle in two ways:

1. **Console cloud-agent workflow**: You (or a cloud agent running in a console session) drive task submission, monitoring, and lifecycle actions with `run-agent` local commands (including `run-agent job batch` for sequential local submissions) and/or `run-agent server ...` API-client commands (`run-agent server job` supports `submit`/`list`, and `submit` supports `--attach-mode create|attach|resume`; `conductor` starts the server by default and adds `conductor job submit-batch`, alongside `status`, `task`, `job`, `project`, `watch`, `monitor`, `bus`, `workflow`, `goal`, and Cobra-generated `completion`; use `run-agent server update ...` for self-update operations).
2. **Web UI workflow**: You perform the same lifecycle directly in the browser at `/ui/` using project/task controls, live logs, and message bus panels.

**When to use which:**

| Scenario | Best for |
|----------|----------|
| Console cloud-agent workflow | Automation, scripting, remote/headless environments, and reproducible terminal logs |
| Web UI workflow | Interactive monitoring, quick manual task operations, and visual run/message-bus navigation |

See [Quick Start Guide](docs/user/quick-start.md), [CLI Reference](docs/user/cli-reference.md), and [Web UI Guide](docs/user/web-ui.md) for complete flows.

Daily startup wrappers:

```bash
# Conductor server (task execution enabled)
./scripts/start-conductor.sh --config config.yaml

# Monitor-only server (run-agent serve --disable-task-start)
./scripts/start-run-agent-monitor.sh --config config.yaml --root ./runs --background

# Monitor-first run-agent wrapper (defaults to --disable-task-start)
./scripts/start-monitor.sh --root ./runs

# Conductor in background mode with explicit files
./scripts/start-conductor.sh --config config.yaml --background --pid-file ./runs/conductor.pid --log-file ./runs/conductor.log
```

`scripts/start-run-agent-monitor.sh` resolves `--root` as the first non-empty value of `$RUN_AGENT_MONITOR_ROOT`, then `$CONDUCTOR_ROOT`, then `$HOME/run-agent`.
`scripts/start-monitor.sh` resolves `--root` as the first non-empty value of `$RUN_AGENT_ROOT`, then `$CONDUCTOR_ROOT`, then `$RUNS_DIR`, then the repository `runs` directory.
`scripts/start-conductor.sh`, `scripts/start-run-agent-monitor.sh`, and `scripts/start-monitor.sh` pass `--port` explicitly (default `14355`); if that port is busy, pass a different port via `--port` or the matching env var.
`scripts/start-monitor.sh` defaults to monitoring mode; pass `--enable-task-start` when you want task execution enabled from that wrapper (requires `--config` or a discoverable config file).

Direct monitor command examples:

```bash
# One local filesystem monitoring pass (reads unchecked TODO items with task IDs)
./run-agent monitor --root ./runs --project my-project --todo TODOs.md --agent codex --once

# Continuous server-driven monitoring loop (Ctrl+C to stop)
./conductor monitor --server http://127.0.0.1:14355 --project my-project --todos TODOs.md --agent codex
```

## Documentation Website (Docker-only)

Conductor Loop documentation website lives in `website/` and is built with Hugo in Docker only.
Local Hugo installation is intentionally not required.

```bash
# Start local docs preview on http://localhost:1313/
./scripts/docs.sh serve

# Build static docs to website/public/
./scripts/docs.sh build

# Verify generated artifacts and key internal links
./scripts/docs.sh verify
```

See [Documentation Site Guide](docs/dev/documentation-site.md) for details.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│      Web UI (frontend/dist or embedded web/src fallback)     │
│                  http://localhost:14355/ui/                   │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API + SSE
┌─────────────────────────▼───────────────────────────────────┐
│       run-agent serve (or conductor server mode)             │
│  - REST API (`/api/v1/*`) + project API (`/api/projects/*`) │
│  - SSE (per-run, all-runs, message, and project streams)      │
│  - Message Bus (`/api/v1/messages`, `/api/projects/...`)    │
└─────────────────────────┬───────────────────────────────────┘
                          │ spawn processes
┌─────────────────────────▼───────────────────────────────────┐
│                      run-agent                               │
│  - Ralph Loop: task/job execution with restart logic        │
│  - Agent Execution: Claude/Codex/Gemini/Perplexity/xAI     │
│  - Child Task Orchestration                                  │
│  - Message Bus Integration                                   │
└──────────────────────────────────────────────────────────────┘
```

### Key Concepts

- **Task**: A unit of work with a prompt, executed by an agent using the Ralph Loop (with restarts)
- **Job**: A single agent execution without restart logic
- **Ralph Loop**: Automatic restart mechanism that monitors child tasks and retries on failure
- **Run**: An execution instance of a task/job with logs and status
- **Message Bus**: Shared communication channel for cross-task coordination

## Documentation

- [Documentation Site Guide](docs/dev/documentation-site.md) - Docker-only Hugo docs workflow
- [Installation Guide](docs/user/installation.md) - Installation instructions for all platforms
- [Quick Start](docs/user/quick-start.md) - 5-minute tutorial
- [Configuration](docs/user/configuration.md) - Complete config.yaml reference
- [CLI Reference](docs/user/cli-reference.md) - All commands and flags
- [API Reference](docs/user/api-reference.md) - REST API documentation
- [Web UI Guide](docs/user/web-ui.md) - Using the web interface
- [Troubleshooting](docs/user/troubleshooting.md) - Common issues and solutions
- [FAQ](docs/user/faq.md) - Frequently asked questions
- [RLM Orchestration](docs/user/rlm-orchestration.md) - Recursive orchestration with parallel sub-agents

### Developer Documentation

- [Developer Guide](docs/dev/development-setup.md) - Contributing to Conductor Loop
- [Architecture](docs/dev/architecture.md) - System design and components
- [Logging and Observability](docs/dev/logging-observability.md) - Structured runtime logs, redaction rules, and operator inspection points
- [Safe Self-Update](docs/dev/self-update.md) - Deferred update/handoff behavior while root runs are active
- [Testing Guide](docs/dev/testing.md) - Running tests
- [Release Checklist](docs/dev/release-checklist.md) - Release validation steps

### Examples

- [Examples Overview](examples/README.md) - End-to-end scenarios and runnable sample configs
- [Patterns and Best Practices](examples/patterns.md) - Reusable orchestration patterns

## Use Cases

Conductor Loop is designed for:

- **Autonomous Agent Systems**: Build self-healing, long-running agent workflows
- **Multi-Agent Coordination**: Orchestrate multiple agents working together
- **Task Automation**: Automate complex multi-step processes with AI
- **Research & Experimentation**: Test and compare different AI agents
- **Production AI Workflows**: Deploy reliable AI-powered automation

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS    | Fully supported | Primary development platform |
| Linux    | Fully supported | All features work |
| Windows  | Limited | Message bus uses mandatory file locks on native Windows (concurrent readers may block), and in-place self-update handoff is unsupported. Use WSL2 for full compatibility. |

## Requirements

- **Go**: 1.24 or higher
- **Docker**: 20.10+ (required for docs site serve/build; optional for non-container local runtime)
- **Git**: Any recent version
- **Node.js**: `^20.19.0 || >=22.12.0` (for frontend development/build; matches the Vite 7 toolchain requirement)
- **API Tokens**: For your chosen agents (Claude, Codex, etc.)

### Agent Integrations (configure at least one)
| Agent | Runtime Requirement | Install / Setup |
|-------|----------------------|-----------------|
| Claude | `claude` CLI in `PATH` | [Claude CLI](https://claude.ai/code) |
| Codex | `codex` CLI in `PATH` | [OpenAI Codex](https://github.com/openai/codex) |
| Gemini | `gemini` CLI in `PATH` | [Gemini CLI](https://github.com/google-gemini/gemini-cli) |
| Perplexity | REST API token (no CLI required) | API token required |
| xAI | REST API token (no CLI required) | API token required |

Run `run-agent validate` to verify config parsing, CLI availability for CLI-backed agents, and detected CLI versions.
When CLI validation runs at task startup, compatibility floors for CLI agents are `claude >= 1.0.0`, `codex >= 0.1.0`, and `gemini >= 0.1.0` (older detected versions are warning-only).
Run `run-agent validate --check-tokens` to verify token files and token env sources.
`run-agent validate --check-network` currently reports placeholder status and does not yet run outbound REST probes.

## License

Apache License 2.0 - see [LICENSE](LICENSE), [NOTICE](NOTICE), and
[Third-Party License Inventory](docs/legal/THIRD_PARTY_LICENSES.md).

## Contributing

Contributions are welcome! Please see [Contributing Guide](docs/dev/contributing.md) for guidelines.

## Support

- **Issues**: [GitHub Issues](https://github.com/jonnyzzz/conductor-loop/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jonnyzzz/conductor-loop/discussions)

## Status

Current version: `dev` (pre-release)

Current implementation highlights (verified on 2026-02-23 against source in `cmd/`, `internal/`, and `scripts/`; runnable CLI/help checks from `go run ./cmd/run-agent --help`, `go run ./cmd/run-agent task --help`, `go run ./cmd/run-agent job --help`, `go run ./cmd/run-agent monitor --help`, `go run ./cmd/run-agent gc --help`, `go run ./cmd/run-agent bus discover --help`, `go run ./cmd/run-agent bus read --help`, `go run ./cmd/run-agent bus post --help`, `go run ./cmd/run-agent watch --help`, `go run ./cmd/run-agent output --help`, `go run ./cmd/run-agent list --help`, `go run ./cmd/run-agent status --help`, `go run ./cmd/run-agent validate --help`, `go run ./cmd/run-agent server --help`, `go run ./cmd/run-agent server watch --help`, `go run ./cmd/run-agent server bus read --help`, `go run ./cmd/run-agent server bus post --help`, `go run ./cmd/run-agent server task --help`, `go run ./cmd/run-agent server project --help`, `go run ./cmd/run-agent server job submit --help`, `go run ./cmd/run-agent server update --help`, `go run ./cmd/conductor --help`, `go run ./cmd/conductor watch --help`, `go run ./cmd/conductor monitor --help`, `go run ./cmd/conductor task --help`, `go run ./cmd/conductor job --help`, `go run ./cmd/conductor job submit --help`, and `go run ./cmd/conductor job submit-batch --help`; startup wrapper `--help` checks for `scripts/start-conductor.sh`, `scripts/start-run-agent-monitor.sh`, and `scripts/start-monitor.sh`; all listed checks returned exit code `0`):
- `run-agent` is the primary CLI and exposes `task`, `job`, `bus`, `list`, `status`, `watch`, `monitor`, `serve`, `server`, `validate`, `output`, `resume`, `stop`, `gc`, `wrap`, `shell-setup`, `workflow`, `goal`, and Cobra-generated `completion`; `task` includes `resume`/`delete`, `job` includes `batch`, and `bus` includes `discover`/`read`/`post`
- `run-agent status` supports activity-aware snapshots via `--activity`/`--drift-after` and script-friendly tab-separated output via `--concise`; `run-agent list` also supports the same activity/drift signal view
- `run-agent monitor` (filesystem mode) reads unchecked `TODOs.md` items containing canonical task IDs (`task-YYYYMMDD-HHMMSS-...`), supports `--once`, `--dry-run`, `--interval`, `--stale-after`, and `--rate-limit`, requires `--project` (and for start/resume/recover actions requires `--agent` or a resolvable config), starts missing tasks, resumes failed/dead tasks, recovers stale running tasks (stop+resume), and finalizes completed tasks by creating `DONE` when the latest output file is non-empty (TODO file flag: `--todo`); `conductor monitor` performs analogous TODO-driven actions against server APIs (`--interval`, `--stale-threshold`, `--rate-limit`), also requires `--project`, defaults `--agent` to `claude`, uses TODO file flag `--todos`, and currently runs only in continuous loop mode (no `--once`/`--dry-run`)
- `run-agent gc` supports run cleanup plus bus/task maintenance flags: `--rotate-bus`, `--bus-max-size`, and `--delete-done-tasks` (in addition to `--older-than`, `--keep-failed`, `--dry-run`, and scoping flags)
- `defaults.diversification` is implemented for runner-side agent selection (`enabled`, `strategy`, `agents`, `weights`, `fallback_on_failure`); supported strategies are `round-robin` and `weighted`, and fallback mode performs one retry with a policy-selected alternate agent after an initial failure
- Task dependency support is implemented end-to-end: `run-agent task --depends-on`, `run-agent server job submit --depends-on`, `conductor job submit --depends-on`, and `conductor job submit-batch --depends-on`; dependencies are persisted in `TASK-CONFIG.yaml`, validated for cycles, and exposed as `depends_on`/`blocked_by` with `blocked` status in CLI and API views
- `run-agent bus post` resolves bus path in priority order: `--bus`, `MESSAGE_BUS`, `--project`/`--task` path resolution, then auto-discovery; message scope fields (`project_id`, `task_id`, `run_id`) resolve via explicit flags, inferred context (resolved bus path, `RUN_FOLDER`, `TASK_FOLDER`), then `JRUN_PROJECT_ID`/`JRUN_TASK_ID`/`JRUN_ID`
- Local `run-agent` commands operate on filesystem state and do not require `run-agent serve` (`run-agent server ...` is the API-client group)
- `run-agent serve` and `conductor` server mode host REST/SSE APIs and the Web UI; when port is not explicitly set by CLI/env, server startup auto-binds the next free port from the configured base port (default `14355`, up to 100 attempts)
- `run-agent serve` can start without an explicit config file (auto-discovery optional), while `conductor` server mode requires `--config` or a discoverable default config
- `run-agent server ...` provides API-client command groups: `status`, `task`, `job`, `project`, `watch`, `bus`, `update`; subcommand coverage includes `task {status,list,runs,logs,stop,resume,delete}`, `job {submit,list}` (`submit` supports `--attach-mode create|attach|resume`, `--wait`, and `--follow`), `project {list,stats,gc,delete}`, `bus {read,post}` (`read` supports `--follow` SSE streaming), and `update {status,start}`
- Watch mode differs by CLI: local `run-agent watch` requires explicit `--task` IDs, while `run-agent server watch` and `conductor watch` default to all tasks in the project when `--task` is omitted
- Task lifecycle controls are available across local and server-first CLIs: `run-agent stop`, `run-agent task delete`, `run-agent server task stop|delete`, and `conductor task stop|delete`
- Project lifecycle controls are available in server-first CLIs: `run-agent server project {list,stats,gc,delete}` and `conductor project {list,stats,gc,delete}`
- Web UI requests from browsers are intentionally prevented from destructive operations (`run`/`task`/`project` deletion and project GC); use CLI/API clients for those actions
- API middleware supports request correlation (`X-Request-ID`) and writes sanitized form-submission audit records to `<root>/_audit/form-submissions.jsonl`
- Structured `key=value` runtime logs are emitted via `internal/obslog` with key/pattern-based redaction for token-like values
- Run completion webhooks are supported via `webhook` config: `run_stop` payloads are delivered asynchronously with optional `events` filtering, optional HMAC signature (`X-Conductor-Signature`), and retry attempts (failures are posted to task message bus as `WARN`)
- API handlers enforce root-bounded path resolution for project/task/run resources and reject traversal outside configured roots
- `conductor` remains an optional server-centric CLI: running `conductor` with no subcommand starts server mode; command groups include `status`, `task`, `job`, `project`, `watch`, `monitor`, `bus`, `workflow`, `goal`, and Cobra-generated `completion`; `conductor job` adds `submit-batch`
- `run-agent task` posts deduplicated completion summaries to the project message bus after `DONE`, with idempotent propagation state in `TASK-COMPLETE-FACT-PROPAGATION.yaml`
- Run-state liveness reconciliation heals stale dead-PID runs to `completed` when a task `DONE` marker exists (instead of leaving/locking them as stale `failed`/`running`)
- Use source-built binaries (`go build -o run-agent ./cmd/run-agent`, `go build -o conductor ./cmd/conductor`) for branch-accurate verification when validating local, uncommitted changes (local binaries in your checkout may be missing or stale)
- Release installer installs `run-agent`; server self-update client actions remain under `run-agent server update`
