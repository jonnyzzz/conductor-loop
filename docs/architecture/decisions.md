# Architecture Decisions

This document captures the rationale behind key architecture choices in conductor-loop.

## 1) Filesystem Over Database

### Context
Conductor-loop needed persistent state for tasks, runs, message buses, and completion markers while staying easy to run locally and offline.

### Decision
Use the filesystem as the primary state store (`TASK.md`, `run-info.yaml`, `TASK-MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`, `DONE`) instead of a database.

### Consequences / Trade-offs
- Simpler operations: zero database setup, easier local debugging, and better offline behavior.
- Git-ops friendly: plain files are inspectable, reviewable, and scriptable with standard tooling.
- Trade-off: no rich query engine (queries are mostly scan/index based) and single-node scaling limits.
- Trade-off: design depends on local filesystem semantics rather than distributed storage guarantees.

## 2) `O_APPEND` + `flock` for Message Bus

### Context
Multiple processes must append to the same message bus safely without introducing external infrastructure.

### Decision
Write bus entries to append-only files using `O_APPEND` plus exclusive `flock` (with retries/backoff), while keeping reads lockless on Unix.

### Consequences / Trade-offs
- Atomic append behavior on Unix and durable file-based event history without Redis/Kafka.
- Operational simplicity: no external queue or broker dependency.
- Trade-off: Windows uses `LockFileEx` semantics, so write locks can block concurrent reads.
- Trade-off: files grow over time and require rotation/GC policy.

## 3) CLI-Wrapped Agents (Claude/Codex/Gemini)

### Context
The system integrates multiple coding agents with different capabilities and release cadences.

### Decision
Run Claude, Codex, and Gemini as isolated CLI subprocesses with redirected stdio. Keep Perplexity and xAI as in-process REST adapters.

### Consequences / Trade-offs
- Process isolation improves containment and makes failures easier to reason about per run.
- Agent updates remain decoupled from conductor-loop releases (CLI tools can evolve independently).
- I/O handling is straightforward via stdout/stderr capture and `output.md` fallback.
- Trade-off: runtime depends on CLI availability/version compatibility on the host.

## 4) YAML Over HCL for Configuration

### Context
Early designs and legacy docs used HCL, but long-term maintenance favored stronger ecosystem support in current Go tooling.

### Decision
Adopt YAML as the primary config format (`gopkg.in/yaml.v3`), with HCL retained only for legacy compatibility.

### Consequences / Trade-offs
- Better library/tooling fit and standardized config direction across docs and runtime.
- Clear precedence (`config.yaml` / `config.yml` before `config.hcl`) reduces ambiguity for new deployments.
- Trade-off: dual-format support still adds compatibility code and migration complexity.

## 5) Default Port `14355`

### Context
A default API/UI port was needed that avoids common collisions in local development environments.

### Decision
Use `14355` as the canonical default port for `run-agent serve` and `conductor`.

### Consequences / Trade-offs
- Lower collision risk than common ports such as `3000` and `8080`.
- Consistent default across CLI/server flows improves predictability.
- Trade-off: any fixed default can still conflict in specific environments, so override support remains necessary.

## 6) Process Groups (PGID) for Lifecycle Control

### Context
Agent processes can spawn child processes; stop/restart flows must clean up the full process tree, not just a single PID.

### Decision
Use process groups (`setsid`/PGID on Unix) and signal the group for termination and liveness checks.

### Consequences / Trade-offs
- Cleaner shutdown behavior and reduced orphaned subprocesses.
- Better control for Ralph-loop stop/timeout handling across agent trees.
- Trade-off: Windows parity is limited and semantics differ from Unix PGID behavior.
