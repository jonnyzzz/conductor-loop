# Architecture Decisions

This document explains the key architectural decisions in conductor-loop — specifically WHY the system was built the way it was.

## Decision 1: Filesystem Over Database

**Context**: Conductor Loop needed a storage backend for task metadata, logs, and message bus data.
**Decision**: All state (run metadata, message bus, task status) lives on the local filesystem.
**Rationale**:
- **Simplicity**: No database setup or maintenance required for users.
- **Portability**: The system works anywhere the binary runs.
- **Debuggability**: State can be inspected and modified with standard CLI tools (cat, grep, vim).
- **Atomicity**: Atomic rename operations provide sufficient consistency for our use case.
**Alternatives Considered**: SQLite (embedded), PostgreSQL (external).
**Trade-offs**:
- No complex query capabilities (mitigated by in-memory indexing).
- Scaling limits constrained by single-node filesystem performance.
- Local filesystem requirement means no NFS/SMB support.
**Status**: Current

## Decision 2: O_APPEND + flock Over Message Queue

**Context**: The system needed a reliable way to record and stream events between processes.
**Decision**: Message bus uses O_APPEND writes with exclusive flock + lockless reads.
**Rationale**:
- **Zero Dependency**: No external message queue (like RabbitMQ or Kafka) to deploy.
- **Performance**: High throughput (~37,000 msg/sec) with OS-buffered writes.
- **Human-Readable**: YAML format makes the bus easy to read and debug.
**Alternatives Considered**: Redis Pub/Sub, SQLite, Embedded NATS.
**Trade-offs**:
- File size grows indefinitely (mitigated by GC and auto-rotation).
- No built-in pub/sub semantics (polling required).
- Restricted to local filesystem locking semantics.
**Status**: Current

## Decision 3: CLI-Wrapped Agents Over Direct API

**Context**: Agents need to execute tasks, access files, and potentially run tools.
**Decision**: Claude, Codex, and Gemini are invoked as CLI subprocesses, not via their REST APIs directly.
**Rationale**:
- **Full Capability**: CLI tools provide built-in tool use, file access, and sandbox management that raw REST APIs don't.
- **Simpler Auth**: Environment variables handle authentication without complex token management in the runner.
- **Context Management**: Agents manage their own context window and history.
**CLI flags used**:
- `claude`: `-p --output-format stream-json --permission-mode bypassPermissions`
- `codex`: `exec --dangerously-bypass-approvals-and-sandbox --json`
- `gemini`: `--screen-reader true --approval-mode yolo --output-format stream-json`
**Alternatives Considered**: Direct REST API integration for all agents.
**Trade-offs**: Dependency on external CLI tools being installed.
**Note**: Perplexity and xAI are exceptions (REST-only) because they lack equivalent CLI tools.
**Status**: Current

## Decision 4: YAML Over HCL (Config Format Evolution)

**Context**: The system needs a configuration format for defining agents and defaults.
**Decision**: YAML is the primary config format; HCL is supported for backward compatibility.
**Rationale**:
- **Ecosystem**: YAML has broader support and tooling in the ecosystem.
- **History**: Initially chose HCL for HashiCorp-style declarative config, but reversed after validating YAML's practical advantages.
**Alternatives Considered**: TOML, JSON.
**Trade-offs**: HCL offers cleaner syntax for some hierarchical data, but YAML is more ubiquitous.
**Current behavior**: Config search order is `config.yaml` > `config.yml` > `config.hcl`.
**Status**: Current

## Decision 5: SSE Over WebSockets

**Context**: The UI needs real-time updates for logs and message bus events.
**Decision**: Server-Sent Events (SSE) for real-time streaming (not WebSockets).
**Rationale**:
- **Simplicity**: HTTP-based protocol with no complex upgrade handshake.
- **Reliability**: Automatic reconnection is built into the browser EventSource API.
- **Unidirectional**: Sufficient for log streaming; we don't need bidirectional socket communication.
**Alternatives Considered**: WebSockets, gRPC-Web.
**Trade-offs**: No bidirectional communication (mitigated by using standard REST for commands).
**Status**: Current

## Decision 6: Single Binary Deployment

**Context**: Distribution and installation need to be as simple as possible.
**Decision**: Both `run-agent` and `conductor` are single statically-linked Go binaries.
**Rationale**:
- **No Dependencies**: No runtime dependencies (Python, Node, JVM) required.
- **Easy Distribution**: `curl` and run.
- **Fast Startup**: Minimal startup overhead.
- **Frontend**: Embedded via `go:embed`; build output goes to `frontend/dist/` which is served at `/ui/`.
**Alternatives Considered**: Docker containers, Python package.
**Trade-offs**: larger binary size due to static linking and embedded assets.
**Status**: Current

## Decision 7: Port 14355 as Default

**Context**: Choosing a default port for the API server and `run-agent serve`.
**Decision**: Default port is 14355.
**Rationale**:
- **Avoid Conflicts**: Avoids common development ports (3000, 8080, 8443, 5000).
- **Memorable**: "14355" is easy enough to type.
**History**: Early spec used 8080; changed to 14355 to avoid conflicts.
**Status**: Current

## Decision 8: DONE File Semantics

**Context**: The orchestration loop needs a reliable signal to stop restarting a task.
**Decision**: Task completion is signaled by creating an empty `DONE` file in the task directory.
**Rationale**:
- **Filesystem-Native**: Survives process restarts and API server downtime.
- **Simple Control**: Can be removed manually to resume/restart a task.
- **Semantics**: Zero-exit code does NOT stop the Ralph loop; only `DONE` stops it.
**Alternatives Considered**: Database status flag, API call.
**Trade-offs**: Relies on agents correctly creating the file.
**Status**: Current

## Decision 9: No Auth for Local Use

**Context**: Securing the API while maintaining ease of use for local development.
**Decision**: Optional API key (Bearer or X-API-Key header); exempt routes include health, version, metrics, `/ui/`.
**Rationale**:
- **Local Security**: Localhost deployment is reasonably secure by network isolation.
- **Onboarding**: Mandatory auth would complicate the first-run experience.
**Enable**: Set `CONDUCTOR_API_KEY` env var or use `--api-key` flag.
**Status**: Current

## Decision 10: Process Groups (PGID) for Agent Management

**Context**: Ensuring clean termination of agent processes and their children.
**Decision**: Each agent runs in its own process group (Setsid=true on Unix).
**Rationale**:
- **Clean Cleanup**: Allows killing the entire agent subtree on stop/timeout.
- **Orphan Prevention**: Prevents orphan processes from lingering.
**Alternatives Considered**: Tracking individual PIDs.
**Trade-offs**: Windows support is limited; WSL2 is recommended.
**Status**: Current
