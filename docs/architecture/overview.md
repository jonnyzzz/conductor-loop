# Conductor Loop Architecture Overview

This document is the primary entry point for understanding conductor-loop. It targets
developers and DevOps engineers evaluating or deploying the system for the first time.

For deeper reading, see the [Architecture Index](README.md).

---

## Table of Contents

1. [What is Conductor Loop?](#what-is-conductor-loop)
2. [Problem It Solves](#problem-it-solves)
3. [System Context Diagram](#system-context-diagram)
4. [Key Design Principles](#key-design-principles)
5. [Technology Stack](#technology-stack)
6. [Two Binaries: `run-agent` and `conductor`](#two-binaries)
7. [Storage Layout](#storage-layout)
8. [Message Bus](#message-bus)
9. [Ralph Loop (Restart Manager)](#ralph-loop)
10. [Agent Backends](#agent-backends)
11. [API Server and Web UI](#api-server-and-web-ui)
12. [16 Subsystems at a Glance](#16-subsystems-at-a-glance)
13. [Key Statistics](#key-statistics)
14. [Related Pages](#related-pages)

---

## What is Conductor Loop?

Conductor Loop is a **Go multi-agent orchestration framework** for running AI agents
as structured, restartable tasks with persistent state, cross-task messaging, and live
observability.

It manages five AI backends:

| Backend | Protocol | Examples |
|---------|----------|---------|
| Claude | CLI (stream-json) | Anthropic Claude Code |
| Codex | CLI (--json) | OpenAI Codex CLI |
| Gemini | CLI (stream-json) | Google Gemini CLI |
| Perplexity | REST (built-in adapter) | Perplexity API |
| xAI / Grok | REST (built-in adapter) | xAI Grok API |

**Core characteristics:**

- **Filesystem-backed state**: all run metadata, outputs, and messages live on disk.
  No database required.
- **Append-only message bus**: cross-task and project-level event log using
  `O_APPEND + flock`. No message queue dependency.
- **Ralph Loop**: automatic task restart and recovery — keeps agents running across
  context overflows, crashes, and SIGTERM without human intervention.
- **Optional HTTP/SSE monitoring**: `run-agent serve` or `conductor` expose the
  filesystem state over a REST+SSE API. The server is not required for task execution.
- **Two active binaries**: `run-agent` (local, filesystem-first CLI) and `conductor`
  (server-centric CLI and API server).

---

## Problem It Solves

Running AI agents on long-horizon tasks raises three hard problems:

### 1. Context Overflow and Restarts
AI agents exhaust their context windows mid-task. Conductor-loop's Ralph Loop restarts
agents automatically while preserving all prior state on disk. The agent signals
completion by creating a `DONE` file; until then, the loop keeps restarting it (up to
`maxRestarts=100` by default).

### 2. Multi-Agent Coordination and Parallel Work
Large tasks can be decomposed into independent sub-tasks executed in parallel. Agents
spawn sub-agents using `run-agent job`, passing their own run ID as `--parent-run-id`.
The parent-child lineage is tracked in `run-info.yaml` and visible in the Web UI.

### 3. Observability Without Interruption
Monitoring an in-progress agent must not require stopping it. Conductor-loop's SSE
streaming reads the live filesystem: log files are tailed directly, the message bus is
polled by ID, and run status is reconciled from on-disk PID/PGID liveness checks. The
monitoring server can be started or stopped at any time without affecting running agents.

---

## System Context Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Operators                                   │
│                                                                     │
│  Developer / DevOps engineer                                        │
│    │                                                                │
│    ├── run-agent CLI  (filesystem-direct)                           │
│    ├── conductor CLI  (server-centric)                              │
│    └── Web UI         (React dashboard, http://localhost:14355/)    │
└────────────────────────┬────────────────────────────────────────────┘
                         │ HTTP / SSE (optional)
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│             API Server  (run-agent serve / conductor)               │
│   REST endpoints · SSE streams · /metrics · Web UI serving         │
│   Auth: optional API key (Bearer / X-API-Key)                      │
└────────────┬────────────────────────────────┬───────────────────────┘
             │ reads filesystem                │ starts / stops agents
             ▼                                ▼
┌─────────────────────────┐    ┌──────────────────────────────────────┐
│   Filesystem State      │    │   Runner + Ralph Loop                │
│                         │    │   internal/runner/                   │
│  <root>/                │◄───│   • Ralph Loop (restart manager)     │
│   <project>/            │    │   • Process manager (PGID)           │
│    <task>/              │    │   • Task dependency gating           │
│     TASK.md             │    │   • Run concurrency semaphore        │
│     DONE                │    └──────────────┬───────────────────────┘
│     TASK-MESSAGE-BUS.md │                   │ spawns
│     runs/<run_id>/      │                   ▼
│      run-info.yaml      │    ┌──────────────────────────────────────┐
│      agent-stdout.txt   │    │          Agent Backends              │
│      output.md          │    │   Claude · Codex · Gemini  (CLI)     │
│      prompt.md          │    │   Perplexity · xAI         (REST)    │
└─────────────────────────┘    │                                      │
                               │  Each agent: env vars injected →     │
                               │  JRUN_PROJECT_ID, JRUN_TASK_ID,      │
                               │  JRUN_ID, MESSAGE_BUS, RUN_FOLDER,   │
                               │  TASK_FOLDER, CONDUCTOR_URL          │
                               └──────────────────────────────────────┘
```

**Key invariant:** The runner and agent processes operate entirely on the local
filesystem. The API server is a read-mostly observer — it reads filesystem state
directly and calls runner package functions only for control actions (stop, start);
the runner itself never calls back to the server.

---

## Key Design Principles

### 1. Offline-First / Filesystem as Truth

All task execution state is persisted to disk. Multiple `run-agent` processes can run
concurrently with no coordination process and no database.

```
run-agent task A  ─┐
run-agent task B  ─┼──► /data/runs/   (atomic writes, flock for shared files)
run-agent task C  ─┘
```

**Rationale:** eliminates operational dependencies; enables inspection with standard
filesystem tools (`cat`, `ls`, text editors); survives crashes without data loss.

### 2. `CONDUCTOR_URL` Is Informational

The environment variable `CONDUCTOR_URL` is injected into spawned agent processes as a
convenience so the agent can construct API URLs for posting messages or querying status.
The runner itself **never calls back to the server**.

```
Runner → spawns → Agent process
                  env: CONDUCTOR_URL=http://localhost:14355
                  (agent MAY use this; runner does NOT)
```

**Rationale:** keeps the execution path free of network dependencies; the server can
be absent, down, or on a different host without affecting task execution.

### 3. `O_APPEND + flock` for Message Bus

Writes use `O_APPEND` with an exclusive `flock` (10s timeout, exponential backoff on
contention). Reads are lockless — no lock is acquired.

**Rationale:** simple, fast, human-readable, and dependency-free. Throughput is
~37,000 messages/sec with 10 concurrent writers (fsync disabled by default).

### 4. No Auth for Local Use

API authentication is optional and disabled by default. To enable, pass `--api-key`
or set `CONDUCTOR_API_KEY`. Exempt paths: `/api/v1/health`, `/api/v1/version`,
`/metrics`, and `/ui/` (for health checks and UI loading).

**Rationale:** localhost deployment; simplest possible setup for a solo developer.
Remote deployments can opt into API key protection.

### 5. Single-Binary Deployment

Each binary is a self-contained executable. No daemon, no database, no external
services required to start running tasks.

**Rationale:** easy CI/CD integration; installable with a single `curl | bash`;
reproducible across machines.

### 6. Zero-Exit-Code Does Not Mean Done

A zero exit code by itself does not stop the Ralph Loop. The agent must create the
`DONE` file in the task directory to signal completion. If `DONE` is absent and the
restart budget remains, the loop continues regardless of exit code.

**Rationale:** agents that exhaust their context window typically exit 0; continuing
the loop allows the next run to pick up where the last left off.

---

## Technology Stack

### Backend

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24 |
| CLI framework | Cobra |
| Config parsing | gopkg.in/yaml.v3 + HashiCorp HCL |
| Error handling | github.com/pkg/errors |
| Locking | `syscall.Flock` (Unix), `LockFileEx` (Windows) |
| Process isolation | `SysProcAttr.Setsid = true` (Unix) |
| Metrics | Prometheus text format (hand-rolled, no client library) |
| Dependencies | Minimal — no ORM, no message queue, no service mesh |

### Frontend

| Component | Technology |
|-----------|-----------|
| Primary UI | React 18 + TypeScript + Vite |
| Component library | JetBrains Ring UI |
| Fallback UI | Vanilla HTML/CSS/JS (no build step) |
| Streaming | Server-Sent Events (SSE) |

### Build

```bash
# Backend
go build -o bin/run-agent  ./cmd/run-agent
go build -o bin/conductor  ./cmd/conductor

# Frontend (primary UI)
cd frontend && npm install && npm run build
# Output: frontend/dist/
```

---

## Two Binaries

### `run-agent` — Local Filesystem CLI

The full orchestration CLI. All local commands work without a running server.

```
run-agent task      — run a task with Ralph Loop restart logic
run-agent job       — submit a job (filesystem-direct) or to a server
run-agent bus       — read/post/discover message bus files
run-agent list      — list projects, tasks, runs from filesystem
run-agent output    — print or live-tail agent output files
run-agent watch     — poll until tasks reach a terminal state
run-agent gc        — garbage-collect old run directories; rotate bus
run-agent serve     — start optional monitoring/control HTTP server
run-agent resume    — remove DONE marker to restart a completed task
run-agent stop      — stop a running task by signaling its process group
run-agent monitor   — manage unchecked TODO task files (e.g. docs/dev/todos.md)
run-agent wrap      — shell wrapper for claude/codex/gemini with task setup
run-agent server …  — API-client commands against a running server
```

### `conductor` — Server-Centric CLI

An independent binary with its own Cobra command tree. Running `conductor` with no
subcommand starts the API server directly.

```
conductor                         — start API server (reads config.yaml)
conductor status                  — query server status
conductor task status|stop|delete — manage tasks via server API
conductor project list|stats|gc   — manage projects via server API
conductor job submit|submit-batch — submit jobs via server API
conductor watch                   — poll until tasks finish
conductor monitor                 — TODO-driven task automation (server mode)
conductor bus read|post           — message bus via server API
conductor workflow|goal           — higher-level workflow/goal commands
```

**Key difference:** `conductor` starts the API server from its root command.
`run-agent serve` is a subcommand that starts the same server within the `run-agent`
binary. Both serve the same REST/SSE API from the same internal code.

---

## Storage Layout

```
<storage_root>/
└── <project_id>/
    ├── PROJECT-MESSAGE-BUS.md       # Project-level event log
    └── <task_id>/
        ├── TASK.md                  # Task prompt (read by agent)
        ├── DONE                     # Completion marker (absent = keep running)
        ├── TASK-MESSAGE-BUS.md      # Task event log (append-only)
        ├── TASK-DEPENDS-ON.yaml     # Task dependency declarations (optional)
        └── runs/
            └── <run_id>/
                ├── run-info.yaml    # Run metadata (YAML, atomic writes)
                ├── agent-stdout.txt # Raw agent stdout (live-streamed via SSE)
                ├── agent-stderr.txt # Raw agent stderr
                ├── output.md        # Final output summary (written by agent)
                └── prompt.md        # Prompt used for this run (copy)
```

### Run ID Format

```
<YYYYMMDD-HHMMSS0000>-<pid>-<seq>

Example: 20260224-0832210000-73641-1
```

Generated as `timestamp + process PID + process-local atomic counter`. The counter
prevents collisions when multiple runs start within the same second from the same PID.

### Run Info Schema (`run-info.yaml`)

Key fields: `run_id`, `project_id`, `task_id`, `agent`, `pid`, `pgid`, `status`
(`running` / `completed` / `failed`), `start_time`, `end_time`, `exit_code`,
`parent_run_id`, `previous_run_id`, `error_summary`, `agent_version`,
`stdout_path`, `stderr_path`, `output_path`, `prompt_path`.

Writes use an atomic temp-file-rename pattern: write to `.tmp.{pid}`, fsync, chmod,
then `os.Rename`. Concurrent updates use a `.lock` file via `flock` with 5s timeout.

### Run-State Liveness Healing

When any read path encounters a run marked `running`, `internal/runstate.ReadRunInfo`
reconciles the status by checking whether the PID/PGID is still alive. If the process
is dead and a `DONE` marker exists, the stale `running` entry is promoted to
`completed`. This prevents the UI from showing zombie runs.

---

## Message Bus

The message bus is an **append-only file** — a standard YAML document stream separated
by `---` markers. Each message carries a unique, lexically-sortable ID.

### Message ID Format

```
MSG-<YYYYMMDD-HHMMSS>-<nanoseconds>-PID<pid5>-<seq4>

Example: MSG-20260224-083221-231902000-PID74059-0001
```

### Concurrency Model

| Path | Locking |
|------|---------|
| **Write** | `O_APPEND` + exclusive `flock` (10s timeout, 3 retries, exponential backoff) |
| **Read** | Lockless — no lock acquired on Unix; on Windows, `LockFileEx` may block reads |
| **fsync** | Optional (`WithFsync`); disabled by default for throughput (~37,000 msg/sec) |

### Rotation

- **Runtime**: `WithAutoRotate(maxBytes)` — wraps the bus and rotates to a dated
  archive file when the bus grows past the configured size.
- **Maintenance**: `run-agent gc --rotate-bus --bus-max-size 10MB` — offline rotation
  during garbage collection.

### SSE Streaming

The API server polls `ReadMessages(lastID)` at ~100ms intervals per SSE connection.
If the message bus was rotated and `lastID` is no longer found
(`ErrSinceIDNotFound`), the cursor resets to the beginning of the new file.

---

## Ralph Loop

The Ralph Loop is the restart manager for root agent tasks.

### Algorithm

```
Initialize: restartCount = 0, maxRestarts = 100

loop:
  if DONE exists → stop (agent signaled completion)
  if restartCount >= maxRestarts → stop (budget exhausted)
  if context canceled → stop (SIGTERM / user stop)

  SpawnAgent()           # create process group (Setsid), redirect stdio
  Wait for exit          # waitpid() equivalent
  Check DONE again       # check immediately after exit

  if DONE → stop
  else → restartCount++; sleep(restartDelay=1s); continue
```

**Key insight:** exit code 0 does NOT stop the loop. Only the `DONE` file stops it.

### Completion Propagation

When a task loop finishes (DONE detected), the Ralph Loop calls
`propagateTaskCompletionToProject`, which posts a deduplicated `FACT` message to
`PROJECT-MESSAGE-BUS.md`. This surfaces cross-task completion signals at the project
level.

### Process Group Management

On Unix, each agent process is started with `SysProcAttr.Setsid = true`, giving it a
new session and process group (PGID = PID). To stop the agent, the runner sends
`SIGTERM` to `-PGID` (the entire group), ensuring no child processes are orphaned.
Liveness checks use `kill(-pgid, 0)` (`ESRCH` = dead, `EPERM` = alive).

Windows uses dedicated `pgid_windows.go`, `stop_windows.go`, `wait_windows.go`
implementations. Native Windows support is experimental; WSL2 is recommended.

---

## Agent Backends

The agent interface is:

```go
type Agent interface {
    Execute(ctx context.Context, runCtx *RunContext) error
    Type() string
}
```

`RunContext` carries run/task identifiers, prompt path, working directory, stdio file
paths, and the environment map injected into the agent process.

### Environment Variables Injected into Every Agent

| Variable | Value |
|----------|-------|
| `JRUN_PROJECT_ID` | Project identifier |
| `JRUN_TASK_ID` | Task identifier |
| `JRUN_ID` | Run identifier |
| `JRUN_PARENT_ID` | Parent run ID (if spawned as sub-agent) |
| `MESSAGE_BUS` | Absolute path to `TASK-MESSAGE-BUS.md` |
| `TASK_FOLDER` | Absolute path to the task directory |
| `RUN_FOLDER` | Absolute path to the current run directory |
| `CONDUCTOR_URL` | URL of the API server (informational) |

### Backend Types

| Backend | Protocol | Notes |
|---------|----------|-------|
| `claude` | CLI, stream-json | Anthropic Claude Code; stream parser extracts assistant text |
| `codex` | CLI, --json | OpenAI Codex CLI |
| `gemini` | CLI, stream-json | Google Gemini CLI |
| `perplexity` | REST (`executeREST`) | Built-in HTTP adapter |
| `xai` | REST (`executeREST`) | Built-in xAI/Grok adapter |

After execution, the runner calls `agent.CreateOutputMD` to ensure `output.md` exists.
Stream parsers for Claude/Codex/Gemini produce a cleaned output file; if parsing
fails, a copy of `agent-stdout.txt` is used as fallback.

### Agent Diversification

The `defaults.diversification` config block enables automatic agent selection across
multiple configured agents:

```yaml
defaults:
  diversification:
    enabled: true
    strategy: round-robin   # or: weighted
    agents: [claude, codex]
    fallback_on_failure: true
```

Fallback counts are exported to Prometheus as `conductor_agent_fallbacks_total`.

---

## API Server and Web UI

The API server is started by either `run-agent serve` or `conductor`. Default port:
**14355** (tries up to 100 consecutive ports if the configured port is in use).

Default bind address: `0.0.0.0` (all interfaces).

### Authentication

- Disabled by default (no-op middleware)
- Enable: `--api-key <key>` or `CONDUCTOR_API_KEY=<key>`
- Protocol: `Authorization: Bearer <key>` or `X-API-Key: <key>`
- Exempt: `/api/v1/health`, `/api/v1/version`, `/metrics`, `/ui/`

### Key API Endpoints

| Category | Endpoint | Notes |
|----------|----------|-------|
| Projects | `GET /api/projects` | List projects |
| Projects | `GET /api/projects/{p}/stats` | Task/run counts |
| Projects | `GET /api/projects/{p}/runs/flat` | Run tree for dashboard |
| Tasks | `GET /api/projects/{p}/tasks` | Paginated (limit=50, max=500) |
| Tasks | `POST /api/projects/{p}/tasks` | Create and start a task |
| Tasks | `DELETE /api/projects/{p}/tasks/{t}` | Delete task (conflicts on active runs) |
| Runs | `GET /api/projects/{p}/tasks/{t}/runs/{r}` | Run metadata |
| Runs | `DELETE /api/projects/{p}/tasks/{t}/runs/{r}` | Delete run directory |
| Messages | `POST /api/projects/{p}/tasks/{t}/messages` | Post a message |
| Messages | `GET /api/projects/{p}/tasks/{t}/messages/stream` | SSE stream |
| Metrics | `GET /metrics` | Prometheus text format (no auth) |
| Admin | `POST /api/v1/admin/self-update` | Deferred binary self-update |

**Safety guardrails:** Browser-origin destructive actions (task/run/project delete,
project GC) are blocked with HTTP 403. Use CLI or non-browser API clients for these.

### Web UI

Served from `frontend/dist/` when `index.html` is present there; falls back to
embedded `web/src/` assets.

**Primary UI** (`frontend/dist/` — React 18 + TypeScript + Ring UI):
- Task search/filtering, run tree visualization
- Live SSE log streaming with JSON/thinking block rendering
- Message bus compose form, stop/resume controls
- Heartbeat badge (green/yellow/red based on recent agent output)
- Project stats dashboard (task/run count cards)

**Fallback UI** (`web/src/` — vanilla HTML/CSS/JS, no build step):
- Task list, run detail, real-time log via SSE, message bus feed

---

## 16 Subsystems at a Glance

| # | Subsystem | Package | Description |
|---|-----------|---------|-------------|
| 1 | Storage | `internal/storage/` | Run metadata YAML, atomic writes, in-memory run index |
| 2 | Config | `internal/config/` | YAML+HCL loading, token resolution, agent defaults |
| 3 | Message Bus | `internal/messagebus/` | Append-only event log, flock, rotation, SSE polling |
| 4 | Agent Protocol | `internal/agent/` | Agent interface, `RunContext`, stdio capture |
| 5 | Agent Backends | `internal/agent/*/` | Claude, Codex, Gemini (CLI); Perplexity, xAI (REST) |
| 6 | Runner | `internal/runner/` | Ralph Loop, process spawning, PGID, concurrency semaphore |
| 7 | API Server | `internal/api/` | REST + SSE, route definitions, middleware, path safety |
| 8 | Webhook | `internal/webhook/` | Async `run_stop` delivery, retries, HMAC signing |
| 9 | Frontend | `frontend/` + `web/src/` | React primary UI + vanilla fallback |
| 10 | `run-agent list` | `cmd/run-agent/` | Filesystem-only project/task/run listing |
| 11 | `run-agent output` | `cmd/run-agent/` | Print or live-tail agent output files |
| 12 | `run-agent watch` | `cmd/run-agent/` | Poll until tasks reach a terminal state |
| 13 | DELETE Run | `internal/api/` | `DELETE /api/projects/{p}/tasks/{t}/runs/{r}` |
| 14 | Task Search Bar | `frontend/src/` | Client-side substring filtering of the task list |
| 15 | DELETE Task | `internal/api/` + `cmd/` | Task directory removal via CLI/API |
| 16 | Project Stats | `frontend/src/ProjectStats.tsx` | Task/run count bar from `/api/projects/{p}/stats` |

For detailed documentation on each subsystem see [Subsystem Deep-Dives](../dev/subsystems.md).

---

## Key Statistics

| Metric | Value |
|--------|-------|
| Total Go lines (incl. tests) | ~59,000 |
| Test files | 111 |
| Active binaries | 2 (`run-agent`, `conductor`) |
| Documented subsystems | 16 |
| Default port | 14355 |
| Default bind address | `0.0.0.0` |
| Ralph Loop max restarts | 100 |
| Message bus throughput | ~37,000 msg/sec (fsync disabled) |
| Run concurrency default | Unlimited (set `defaults.max_concurrent_runs` to limit) |
| Root task concurrency default | Unlimited (set `defaults.max_concurrent_root_tasks` to limit) |
| Go version | 1.24 |
| Frontend | React 18 + TypeScript + JetBrains Ring UI |

---

## Related Pages

| Document | Description |
|----------|-------------|
| [Architecture Index](README.md) | Index of all architecture docs |
| [Component Reference](components.md) | Per-component interface and responsibility |
| [Architecture Decisions](decisions.md) | ADRs: why key design choices were made |
| [Task Lifecycle Data Flow](data-flow-task-lifecycle.md) | End-to-end task execution flow |
| [Message Bus Data Flow](data-flow-message-bus.md) | Message bus write/read/stream flows |
| [API Request Lifecycle](data-flow-api.md) | Request through API to filesystem |
| [Agent Integration](agent-integration.md) | Backend protocol specifications |
| [Deployment Architecture](deployment.md) | Local, Docker, and remote setups |
| [Frontend Architecture](frontend-architecture.md) | React UI component structure |
| [Observability Architecture](observability.md) | Logging, metrics, SSE, audit log |
| [Security Architecture](security.md) | Auth, path safety, HMAC webhooks |
| [Concurrency Architecture](concurrency.md) | Semaphores, flock, process groups |
| [Subsystem Deep-Dives](../dev/subsystems.md) | Detailed per-subsystem documentation |
| [Developer Architecture](../dev/architecture.md) | Implementation-level architecture |

### Source of Truth Code Paths

```
cmd/run-agent/          — run-agent entry point and subcommands
cmd/conductor/          — conductor entry point and subcommands
internal/runner/        — Ralph Loop, orchestrator, process manager
internal/storage/       — run metadata, atomic writes, index
internal/messagebus/    — append-only bus, locking, rotation
internal/api/           — REST + SSE server, routes, middleware
internal/agent/         — agent interface + all backends
internal/config/        — config loading, token resolution
internal/runstate/      — run-state liveness healing
internal/metrics/       — Prometheus counter registration
internal/webhook/       — async run_stop delivery
frontend/src/           — React 18 TypeScript primary UI
web/src/                — vanilla HTML/CSS/JS fallback UI
```

### Primary Grounding Docs

```
docs/facts/FACTS-architecture.md
docs/facts/FACTS-runner-storage.md
docs/facts/FACTS-messagebus.md
docs/facts/FACTS-agents-ui.md
docs/dev/architecture.md
docs/dev/subsystems.md
README.md
```

---

**Last Updated:** 2026-02-24
**Facts validated against:** `docs/facts/FACTS-architecture.md` (reconciliation round 2026-02-24)
