# Conductor Loop - Agent Orchestration Framework

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

**Conductor Loop** is an agent orchestration framework for running AI agents (Claude, Codex, Gemini, Perplexity, xAI) with Ralph Loop task management, hierarchical task execution, live log streaming, and a web UI for monitoring.

## Features

- **Multiple AI Agents**: Support for Claude, Codex, Gemini, Perplexity, and xAI
- **Ralph Loop**: Automatic task restart and recovery on failure
- **Run Concurrency Limit**: Optional global run semaphore via `defaults.max_concurrent_runs` (`0` means unlimited)
- **Root-Task Planner Queue**: Server-side deterministic FIFO scheduling for API/UI-submitted root tasks when `defaults.max_concurrent_root_tasks > 0` (configurable root-task concurrency)
- **Hierarchical Task Lineage**: Parent-child task/run relationships with `parent_run_id` propagation across run metadata and project APIs
- **Task Dependencies (`depends_on`)**: Declarative task dependencies via `--depends-on` flags (`run-agent task`, `run-agent server job submit`, `conductor job submit`, `conductor job submit-batch`) with cycle checks and runtime dependency gating
- **Batch Job Submission**: Local sequential batching via `run-agent job batch` and server-side batch submission via `conductor job submit-batch`
- **Live Log Streaming**: Real-time SSE-based log streaming via REST API
- **Web UI**: React dashboard (`frontend/dist`) with message bus compose form, stop/resume controls, JSON-aware message rendering, thinking block rendering, and heartbeat indicators; when `frontend/dist` is unavailable, server falls back to embedded `web/src` assets for baseline monitoring
- **Message Bus**: Cross-task communication with GET/POST plus SSE stream endpoints (`/api/v1/messages/stream?project_id=...` with optional `task_id=...`, `/api/projects/{project}/messages/stream`, `/api/projects/{project}/tasks/{task}/messages/stream`); compose form in web UI, CLI auto-discovery via `run-agent bus discover` (supports `--from` upward search), server-side streaming reads via `run-agent server bus read --follow`, and context-aware posting (`run-agent bus post` resolves bus path via `--bus`, `MESSAGE_BUS`, `--project`/`--task`, then auto-discovery; message IDs resolve via explicit flags, inferred context, then `JRUN_*` env vars)
- **Task Completion Fact Propagation**: When a task reaches `DONE`, `run-agent task` posts a deduplicated completion `FACT` summary from task scope to the project message bus (`PROJECT-MESSAGE-BUS.md`)
- **Run-State Liveness Healing**: Run status reads reconcile stale/dead PIDs; when task `DONE` exists, stale `running`/reconciled `failed` runs are promoted to `completed` with normalized exit code (`0`)
- **Task Resume**: Remove `DONE` markers via API/Web UI/CLI (`run-agent resume`, `run-agent server task resume`, `conductor task resume`); re-run stopped/failed task loops with `run-agent task resume`
- **Task Stop/Delete Controls**: Stop running tasks and delete task state via CLI/API (`run-agent stop`, `run-agent task delete`, `run-agent server task stop|delete`, `conductor task stop|delete`)
- **Project Lifecycle Controls**: Query and maintain project state via CLI/API (`run-agent gc --project ...`, `run-agent server project list|stats|gc|delete`, `conductor project list|stats|gc|delete`)
- **Web UI Safety Guardrails**: Browser-origin destructive actions are blocked in the web UI (run/task/project deletion and project GC); use CLI/API clients for those operations
- **Prometheus Metrics**: `/metrics` endpoint with uptime, active/completed/failed/queued run counts, message bus append totals, and API request counters
- **Form Submission Audit Log**: Accepted form submissions are appended as sanitized JSONL records in `<root>/_audit/form-submissions.jsonl`
- **Webhook Notifications**: Optional asynchronous `run_stop` delivery via `webhook` config (`url`, optional `events`, `secret`, `timeout`), with retry/backoff and optional HMAC signing (`X-Conductor-Signature`)
- **Safe Server Self-Update**: `/api/v1/admin/self-update` with `run-agent server update status` and `run-agent server update start --binary ...` supports deferred handoff when active root runs exist (in-place handoff is not supported on native Windows; use a source-built `run-agent` CLI for `server update` commands when checked-in binaries lag)
- **API Path Safety**: REST handlers reject project/task/run path traversal attempts and enforce path confinement within configured root directories
- **API Authentication**: Optional API key auth for server APIs (`run-agent serve --api-key ...` / `conductor --api-key ...`), with `/api/v1/health`, `/api/v1/version`, `/metrics`, and UI routes under `/ui/` kept publicly accessible for health checks and UI loading (use `/ui/` with a trailing slash)
- **Request Correlation**: API responses include `X-Request-ID`; clients can provide their own request ID for end-to-end tracing across API logs and audit records
- **Structured Observability Logs**: Centralized redacted `key=value` logging (`internal/obslog`) across startup, API, runner, message bus, and storage events
- **Storage**: Persistent run storage with structured logging; managed runs ensure `output.md` exists, and stream parsers extract assistant text for Claude/Codex/Gemini when available
- **Output Inspection**: `run-agent output` supports `--follow` for running jobs (tails live run files) plus `--tail` for quick run-output slicing; without `--follow`, `--file output` reads `output.md` first and falls back to `agent-stdout.txt` when `output.md` is missing, while `--follow --file output` tails `agent-stdout.txt`; `--file prompt` prints stored `prompt.md`
- **Activity Signals**: `run-agent list --activity` surfaces latest bus/output signals and analysis-drift hints (tunable with `--drift-after`)
- **CLI Modes**: `run-agent` for local filesystem workflows (no server required for local commands), `run-agent server ...` for API-client workflows against a running API server (`run-agent serve` or source-built `conductor` server mode; `run-agent server job submit` supports `--wait` and `--follow`), and optional `conductor` (source-built server-centric CLI) for server-first workflows; running source-built `conductor` with no subcommand starts the API server, and subcommands cover `status`, `task`, `job`, `project`, `watch`, `bus`, `workflow`, `goal`, and Cobra-generated `completion` (checked-in `./conductor` may be a compatibility shim that forwards to `run-agent`)
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
# Build conductor from source (checked-in ./conductor can be a compatibility shim)
go build -o conductor ./cmd/conductor
# Use freshly built binaries for command parity with this README.
# Checked-in/prebuilt binaries can lag this branch (for example, `./conductor` forwarding to `run-agent`).

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

# 6. Watch a task in a project until completion
./run-agent watch --root ./runs --project my-project --task task-20260222-154500-demo --timeout 30m
```

Optional source-checkout verification before running workflows:

```bash
go test ./...
golangci-lint run   # if installed
```

`scripts/start-conductor.sh` prefers a `conductor` executable when one is found (including compatibility shims) and falls back to `run-agent serve` only when no `conductor` binary is found.
If `./conductor` prints a deprecation banner or forwards to `run-agent`, rebuild it (`go build -o conductor ./cmd/conductor`) before using `start-conductor.sh` (the compatibility shim is detected as `conductor` but does not support conductor server-mode flags).
`scripts/start-conductor.sh` and `scripts/start-run-agent-monitor.sh` require a config file path (explicit `--config` or successful auto-discovery); use `scripts/start-monitor.sh` for config-optional monitor mode.

### Two Supported Working Scenarios

Conductor Loop supports the same task lifecycle in two ways:

1. **Console cloud-agent workflow**: You (or a cloud agent running in a console session) drive task submission, monitoring, and lifecycle actions with `run-agent` local commands (including `run-agent job batch` for sequential local submissions) and/or `run-agent server ...` API-client commands (`run-agent server job` supports `submit`/`list`, and `submit` supports `--attach-mode create|attach|resume`; source-built `conductor` starts the server by default and adds `conductor job submit-batch`, alongside `status`, `task`, `job`, `project`, `watch`, `bus`, `workflow`, `goal`, and Cobra-generated `completion`; use `run-agent server update ...` for self-update operations).
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
- **Node.js**: `^20.19.0 || ^22.12.0 || >=24.0.0` (for frontend development/build)
- **API Tokens**: For your chosen agents (Claude, Codex, etc.)

### Agent Integrations (configure at least one)
| Agent | Runtime Requirement | Install / Setup |
|-------|----------------------|-----------------|
| Claude | `claude` CLI in `PATH` | [Claude CLI](https://claude.ai/code) |
| Codex | `codex` CLI in `PATH` | [OpenAI Codex](https://github.com/openai/codex) |
| Gemini | `gemini` CLI in `PATH` | [Gemini CLI](https://github.com/google-gemini/gemini-cli) |
| Perplexity | REST API token | API token required |
| xAI | REST API token | API token required |

Run `run-agent validate` to verify config parsing, CLI availability, and detected CLI versions.
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

Current implementation highlights (verified on 2026-02-23 against source in `cmd/`, `internal/`, and `scripts/`; source-run CLI/help checks from `go run ./cmd/run-agent --help`, `go run ./cmd/run-agent bus discover --help`, `go run ./cmd/run-agent watch --help`, `go run ./cmd/run-agent server --help`, `go run ./cmd/run-agent server job submit --help`, `go run ./cmd/run-agent server update --help`, `go run ./cmd/conductor --help`, and `go run ./cmd/conductor job submit-batch --help`; checked-in binary checks from `./run-agent --help`, `./run-agent server --help`, and `./conductor --help`; and startup wrapper `--help` checks for `scripts/start-conductor.sh`, `scripts/start-run-agent-monitor.sh`, and `scripts/start-monitor.sh`):
- `run-agent` is the primary CLI and exposes `task`, `job`, `bus`, `list`, `status`, `watch`, `serve`, `server`, `validate`, `output`, `resume`, `stop`, `gc`, `wrap`, `shell-setup`, `workflow`, `goal`, and Cobra-generated `completion`; `task` includes `resume`/`delete`, `job` includes `batch`, and `bus` includes `discover`/`read`/`post`
- `run-agent gc` supports run cleanup plus bus/task maintenance flags: `--rotate-bus`, `--bus-max-size`, and `--delete-done-tasks` (in addition to `--older-than`, `--keep-failed`, `--dry-run`, and scoping flags)
- Task dependency support is implemented end-to-end: `run-agent task --depends-on`, `run-agent server job submit --depends-on`, `conductor job submit --depends-on`, and `conductor job submit-batch --depends-on`; dependencies are persisted in `TASK-CONFIG.yaml`, validated for cycles, and exposed as `depends_on`/`blocked_by` with `blocked` status in CLI and API views
- `run-agent bus post` resolves bus path in priority order: `--bus`, `MESSAGE_BUS`, `--project`/`--task` path resolution, then auto-discovery; message project/task/run IDs resolve via explicit flags, inferred context (resolved bus path, `RUN_FOLDER`, `TASK_FOLDER`), then `JRUN_PROJECT_ID`/`JRUN_TASK_ID`/`JRUN_ID`
- Local `run-agent` commands operate on filesystem state and do not require `run-agent serve` (`run-agent server ...` is the API-client group)
- `run-agent serve` and source-built `conductor` server mode host REST/SSE APIs and the Web UI; when port is not explicitly set by CLI/env, server startup auto-binds the next free port from the configured base port (default `14355`, up to 100 attempts)
- `run-agent serve` can start without an explicit config file (auto-discovery optional), while source-built `conductor` server mode requires `--config` or a discoverable default config
- Source-built `run-agent server ...` provides API-client command groups: `status`, `task`, `job`, `project`, `watch`, `bus`, `update`; subcommand coverage includes `task {status,list,runs,logs,stop,resume,delete}`, `job {submit,list}` (`submit` supports `--attach-mode create|attach|resume`, `--wait`, and `--follow`), `project {list,stats,gc,delete}`, `bus {read,post}` (`read` supports `--follow` SSE streaming), and `update {status,start}`
- Watch mode differs by CLI: local `run-agent watch` requires explicit `--task` IDs, while `run-agent server watch` and `conductor watch` default to all tasks in the project when `--task` is omitted
- Task lifecycle controls are available across local and server-first CLIs: `run-agent stop`, `run-agent task delete`, `run-agent server task stop|delete`, and `conductor task stop|delete`
- Project lifecycle controls are available in server-first CLIs: `run-agent server project {list,stats,gc,delete}` and `conductor project {list,stats,gc,delete}`
- Web UI requests from browsers are intentionally prevented from destructive operations (`run`/`task`/`project` deletion and project GC); use CLI/API clients for those actions
- API middleware supports request correlation (`X-Request-ID`) and writes sanitized form-submission audit records to `<root>/_audit/form-submissions.jsonl`
- Structured `key=value` runtime logs are emitted via `internal/obslog` with key/pattern-based redaction for token-like values
- Run completion webhooks are supported via `webhook` config: `run_stop` payloads are delivered asynchronously with optional `events` filtering, optional HMAC signature (`X-Conductor-Signature`), and retry attempts (failures are posted to task message bus as `WARN`)
- API handlers enforce root-bounded path resolution for project/task/run resources and reject traversal outside configured roots
- Source-built `conductor` remains an optional server-centric CLI: running `conductor` with no subcommand starts server mode; command groups include `status`, `task`, `job`, `project`, `watch`, `bus`, `workflow`, `goal`, and Cobra-generated `completion`; `conductor job` adds `submit-batch`
- `run-agent task` posts deduplicated completion summaries to the project message bus after `DONE`, with idempotent propagation state in `TASK-COMPLETE-FACT-PROPAGATION.yaml`
- Run-state liveness reconciliation heals stale dead-PID runs to `completed` when a task `DONE` marker exists (instead of leaving/locking them as stale `failed`/`running`)
- Checked-in/prebuilt binaries are not authoritative for this branch; use source-built binaries (`go build -o run-agent ./cmd/run-agent`, `go build -o conductor ./cmd/conductor`) or explicit source-run checks (`go run ./cmd/run-agent --help`, `go run ./cmd/conductor --help`) for verification (in this checkout, `./conductor` prints a deprecation banner, forwards to `run-agent`, and rejects server-mode flags such as `--root`; checked-in `./run-agent server --help` currently omits `update` while source-run help includes it)
- Release installer installs `run-agent`; server self-update client actions remain under `run-agent server update`
