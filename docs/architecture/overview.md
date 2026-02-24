# Conductor Loop Architecture Overview

## What conductor-loop is and the problem it solves
Conductor Loop is a local-first, Go-based orchestration framework for running AI agents (Claude, Codex, Gemini, Perplexity, xAI) as structured tasks with restart semantics (Ralph Loop), run lineage, and observable state.

It solves a practical coordination problem: how to run many agent executions reliably without requiring cloud control planes or databases, while still giving operators live visibility, auditability, and restart/recovery controls.

## System context: users and integrated agent types
Conductor Loop sits between human operators and agent runtimes:

| Actor | How they interact | Why it exists |
|---|---|---|
| Developer/operator | `run-agent` CLI, `conductor` CLI/server, Web UI | Start tasks/jobs, monitor progress, inspect runs, post messages |
| CLI-backed agents | Claude, Codex, Gemini | Tool-using coding/reasoning agents executed as local processes |
| REST-backed agents | Perplexity, xAI | Streaming API-based agents integrated through built-in adapters |
| UI client | React + Ring UI frontend over REST/SSE | Real-time monitoring and task/run navigation |

## Key design principles
- **Offline-first execution**: task/job execution works directly on local filesystem state; server mode is optional for API/UI access.
- **Filesystem is the source of truth**: run metadata (`run-info.yaml`), prompts/outputs, `DONE`, and message buses are persisted as files under the storage tree.
- **Local no-auth by default**: API auth is opt-in via API key; when no key is configured, API key middleware is a no-op.

## Technology stack
- **Backend**: Go `1.24` (`go.mod`), Cobra CLI, YAML v3 + HashiCorp HCL parsing.
- **Execution and orchestration**: Ralph Loop in `internal/runner` with hierarchical run/task lineage.
- **State and messaging**: filesystem storage (`internal/storage`) and append-only message bus (`internal/messagebus`).
- **API and streaming**: HTTP REST + Server-Sent Events (SSE) in `internal/api`.
- **UI**: React + TypeScript + Vite, JetBrains Ring UI (`frontend/`).
- **Configuration**: YAML and HCL config loading (`internal/config/config.go`).

## Brief architecture map
```text
Operators (CLI/UI)
    |
    v
run-agent / conductor (cmd/*)
    |
    v
Runner + Ralph Loop (internal/runner)
    |                         \
    |                          +--> Agent adapters (internal/agent/*)
    v
Filesystem state (internal/storage)
    +
Append-only message bus (internal/messagebus)
    ^
    |
API + SSE server (internal/api) <--> React + Ring UI (frontend)
```

## Related architecture pages
- [Architecture Index](README.md)
- [Component Reference](components.md)
- [Architecture Decisions](decisions.md)
- [Task Lifecycle Data Flow](data-flow-task-lifecycle.md)
- [Message Bus Data Flow](data-flow-message-bus.md)
- [API Request Lifecycle](data-flow-api.md)
- [Agent Integration Architecture](agent-integration.md)
- [Deployment Architecture](deployment.md)
- [Frontend Architecture](frontend-architecture.md)
- [Observability Architecture](observability.md)
- [Security Architecture](security.md)
- [Concurrency Architecture](concurrency.md)

## Source of truth
Primary code paths:
- `go.mod`
- `cmd/run-agent/`
- `cmd/conductor/`
- `internal/runner/`
- `internal/storage/`
- `internal/messagebus/`
- `internal/api/routes.go`
- `internal/api/sse.go`
- `internal/api/auth.go`
- `internal/config/config.go`
- `frontend/package.json`
- `frontend/src/`

Primary grounding docs:
- `docs/facts/FACTS-architecture.md`
- `docs/facts/FACTS-runner-storage.md`
- `docs/facts/FACTS-messagebus.md`
- `docs/facts/FACTS-agents-ui.md`
- `docs/dev/architecture.md`
- `docs/dev/subsystems.md`
- `README.md`
