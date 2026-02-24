# Architecture Decisions

This page captures architecture decisions that are currently implemented in the codebase.
It aligns with `docs/decisions/*`, facts documents, `docs/dev/architecture.md`, `README.md`, and current source behavior.

## AD-01: Filesystem-First State (Instead of Database)

### Decision
Use the local filesystem as the system of record for runs, tasks, and message buses, rather than introducing a database dependency.

### Context/problem
Conductor Loop needs to run in local/offline workflows, support many direct CLI operations, and remain operable even when no API server is running.

### Rationale
- Core state (`run-info.yaml`, `TASK-MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`, `DONE`) is naturally file-scoped per task/run.
- Most operational commands (`task`, `job`, `bus`, `list`, `watch`, `gc`) work directly against filesystem state.
- Operators can inspect and recover state with standard tools without a DB service.
- Atomic file-write patterns and explicit locking cover consistency-critical paths.

### Trade-offs
- Query capabilities are limited compared to relational/document databases.
- Large-scale aggregation requires scans/caches in application code.
- Local filesystem semantics are assumed; some network filesystems are not supported for critical concurrency paths.

### Where implemented
- `internal/storage/storage.go`
- `internal/storage/atomic.go`
- `cmd/run-agent/server.go`
- `cmd/run-agent/list.go`
- `cmd/run-agent/bus.go`

## AD-02: Append-Only Message Bus with `O_APPEND` + `flock` (Instead of External Queue)

### Decision
Implement message transport as append-only files with `O_APPEND` and exclusive file locking, not a brokered queue (Kafka/RabbitMQ/etc.).

### Context/problem
Multiple processes need to post ordered task/project events safely, with minimal operational overhead and no required external infrastructure.

### Rationale
- `O_APPEND` keeps writes append-only at the file descriptor level.
- Exclusive locks serialize writers and avoid interleaved records.
- Read paths stay lock-free on Unix for high read throughput.
- Retry/backoff handles transient lock contention.
- `WithFsync` is available when durability is prioritized; default keeps throughput high.

### Trade-offs
- Message files grow over time and require rotation/GC.
- Queue semantics like consumer groups, broker replication, and ack protocols are intentionally absent.
- On native Windows, mandatory lock behavior can block concurrent reads while writers hold the lock.

### Where implemented
- `internal/messagebus/messagebus.go`
- `internal/messagebus/lock.go`
- `internal/messagebus/lock_unix.go`
- `internal/messagebus/lock_windows.go`
- `cmd/run-agent/gc.go`

## AD-03: CLI-Wrapped Agents for Claude/Codex/Gemini; REST for Perplexity/xAI

### Decision
Run `claude`, `codex`, and `gemini` through CLI process wrappers; execute `perplexity` and `xai` through in-process REST backends.

### Context/problem
The runner must support heterogeneous providers while preserving a consistent run lifecycle (`run-info`, stdout/stderr capture, bus events, timeouts, output fallback).

### Rationale
- CLI path keeps parity with vendor CLIs and local auth/tooling for Claude/Codex/Gemini.
- Runner process management gives uniform lifecycle control for CLI agents.
- REST path avoids requiring local binaries for Perplexity/xAI and integrates directly with provider APIs.
- Validation logic mirrors this split: CLI presence/version checks for CLI agents; token checks for REST agents.

### Trade-offs
- Two execution paths increase maintenance surface.
- CLI-backed agents depend on local binary installation and flag compatibility.
- REST-backed agents depend on network/API stability and token availability.

### Where implemented
- `internal/runner/job.go`
- `internal/runner/validate.go`
- `internal/agent/claude/claude.go`
- `internal/agent/codex/codex.go`
- `internal/agent/perplexity/perplexity.go`
- `internal/agent/xai/xai.go`
- `README.md`

## AD-04: YAML-Oriented Configuration with HCL Compatibility

### Decision
Treat YAML as the primary configuration format while keeping HCL parsing support for compatibility.

### Context/problem
The project needs human-editable config with clear defaults and backward compatibility with historical HCL usage.

### Rationale
- Default config discovery prefers `config.yaml` then `config.yml`, with `config.hcl` as a compatible fallback.
- Loader auto-detects by extension and maps both formats into one `Config` model.
- User-facing setup and examples are YAML-first.
- Keeping HCL support reduces migration pressure for existing setups.

### Trade-offs
- Dual-format support adds parser and test complexity.
- Documentation and behavior can drift if YAML/HCL parity is not continuously validated.
- Operators must understand precedence rules when multiple config files exist.

### Where implemented
- `internal/config/config.go`
- `internal/config/api.go`
- `internal/config/tokens.go`
- `internal/config/validation.go`
- `README.md`

## AD-05: Default Port `14355` with Auto-Bind on Non-Explicit Port Selection

### Decision
Use `14355` as the canonical default API/UI port, and when port selection is not explicit, auto-bind to the next free port (up to 100 attempts).

### Context/problem
The system needs a predictable default URL for CLI/UI workflows, but should avoid startup failure when the default/configured base port is already in use.

### Rationale
- `14355` is the shared default across server startup and client command defaults.
- API config defaults normalize host/port (`0.0.0.0:14355`) when unset.
- Server startup can fall forward to nearby free ports when users did not explicitly pin a port.
- Explicit port selection remains strict to preserve operator intent.

### Trade-offs
- Auto-bind can move the actual port away from `14355`, so callers may need to read startup output.
- Strict explicit-port mode can fail fast on conflicts.
- Network policies/firewalls may need wider allowance when using auto-bind.

### Where implemented
- `internal/config/api.go`
- `internal/api/server.go`
- `cmd/run-agent/serve.go`
- `cmd/conductor/main.go`
- `cmd/run-agent/server.go`
- `README.md`
