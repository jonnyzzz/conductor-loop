# Conductor Loop - Architecture Documentation

This directory contains comprehensive architecture documentation for conductor-loop,
a Go multi-agent orchestration framework.

## Quick Links

| If you want to... | Read... |
| --- | --- |
| Understand what conductor-loop is | [overview.md](overview.md) |
| Find a specific component | [components.md](components.md) |
| Understand why we made certain choices | [decisions.md](decisions.md) |
| Trace a task from submit to DONE | [data-flow-task-lifecycle.md](data-flow-task-lifecycle.md) |
| Understand the message bus | [data-flow-message-bus.md](data-flow-message-bus.md) |
| Understand the REST API | [data-flow-api.md](data-flow-api.md) |
| Learn how agents are invoked | [agent-integration.md](agent-integration.md) |
| Deploy conductor-loop | [deployment.md](deployment.md) |
| Understand the frontend | [frontend-architecture.md](frontend-architecture.md) |
| Set up monitoring | [observability.md](observability.md) |
| Understand security | [security.md](security.md) |
| Understand concurrency | [concurrency.md](concurrency.md) |

## Pages

### [overview.md](overview.md)
**Purpose**: Provide the primary architecture entry point for people learning conductor-loop.

This page explains the problem space, system boundaries, and the major design principles behind the framework. It also maps the runtime shape of the platform, including the two binaries, storage model, message bus, and subsystem inventory.

**Key topics**
- Problem statement and system context diagram
- Design principles and technology stack
- Runtime split between `run-agent` and `conductor`
- Storage/message bus architecture and Ralph Loop model
- Subsystem catalog and project-level architecture stats

**Primary audience**: Architect

### [components.md](components.md)
**Purpose**: Serve as code-grounded reference documentation for core subsystems and their dependencies.

This page catalogs 16 major components with responsibilities, key types, and important cross-component interactions. It is aligned to implementation under `internal/`, `cmd/`, and `frontend/` so developers can map concepts directly to source files.

**Key topics**
- Component dependency graph across config, runner, API, storage, bus, and UI
- Detailed subsystem breakdowns and package-level responsibilities
- CLI and API feature components (`list`, `output`, `watch`, task deletion)
- Cross-cutting data types (`RunInfo`, `Message`, `Agent`, `RunContext`)
- Dependency direction and layering rules

**Primary audience**: Developer

### [decisions.md](decisions.md)
**Purpose**: Record the architectural decisions that shaped the current system design.

This page captures context, decision, rationale, alternatives, and trade-offs for core choices in storage, messaging, runtime architecture, and security defaults. It also documents current status for each decision so future changes can be evaluated against explicit prior intent.

**Key topics**
- Filesystem-first persistence model
- Message bus design (`O_APPEND` + `flock`) vs queue brokers
- CLI-wrapped agents vs direct provider APIs
- Transport/runtime choices (SSE, single-binary, default port)
- Operational semantics (`DONE` behavior, process-group management, local auth default)

**Primary audience**: Architect

### [data-flow-task-lifecycle.md](data-flow-task-lifecycle.md)
**Purpose**: Explain how a task moves through the runner from submission to completion propagation.

This page traces the full lifecycle from task acceptance and dependency checks to run creation, execution, output fallback handling, and `DONE` semantics. It includes implementation references and state transition details that connect runner behavior to storage and message bus outcomes.

**Key topics**
- End-to-end task lifecycle sequence
- Dependency gating and cycle-aware readiness checks
- Ralph Loop restart control and limits
- Run materialization, `output.md` guarantees, and failure fallback paths
- Task completion propagation into `PROJECT-MESSAGE-BUS.md`

**Primary audience**: Developer

### [data-flow-message-bus.md](data-flow-message-bus.md)
**Purpose**: Describe message creation, storage, reading, and streaming across scopes.

This page details the hierarchical project/task/run message model and the append/read/follow mechanics used by CLI and API/SSE flows. It also specifies `msg_id` format, locking behavior, and discovery rules for locating bus files.

**Key topics**
- Scope hierarchy (`project_id`, optional `task_id`, optional `run_id`)
- Append path and validation in `AppendMessage`
- Message ID structure and uniqueness semantics
- Lockless read/follow behaviors in API and CLI
- Bus file discovery and auto-resolution order

**Primary audience**: Developer

### [data-flow-api.md](data-flow-api.md)
**Purpose**: Document request handling from HTTP entry through middleware, handlers, and SSE streaming.

This page explains routing, authentication, CORS, path safety guards, and endpoint-specific behavior for REST and event streams. It also covers metrics scrape flow and includes sequence diagrams for important API paths.

**Key topics**
- Middleware chain ordering and responsibilities
- Route dispatch into projects/tasks/messages handlers
- Identifier validation and path traversal protections
- Optional API key enforcement and exempt endpoints
- SSE architecture and Prometheus `/metrics` request flow

**Primary audience**: Developer

### [agent-integration.md](agent-integration.md)
**Purpose**: Explain how runner orchestration integrates heterogeneous agent backends.

This page covers execution mode selection (CLI vs REST), runtime env injection, prompt preamble contracts, diversification behavior, and version validation policy. It ties each concept to runner and backend implementation so backend-specific behavior is transparent.

**Key topics**
- Agent invocation matrix and backend mapping
- Runner execution paths (`executeCLI` and `executeREST`)
- Runtime contract (environment variables and prompt preamble)
- Diversification policy for model/provider selection
- Version detection and minimum-version checks

**Primary audience**: Developer

### [deployment.md](deployment.md)
**Purpose**: Provide the operational deployment model for running conductor-loop.

This page describes server startup forms, config discovery precedence, on-disk run layout, retention/GC behavior, self-update handoff, and port binding defaults. It is grounded in executable entrypoints and runtime server/config/storage code used in production.

**Key topics**
- Single runtime with two CLI entrypoints (`run-agent serve`, `conductor`)
- Configuration precedence and environment overrides
- Storage layout and run directory structure
- Garbage collection and retention policy behavior
- Self-update flow and port binding/ops considerations

**Primary audience**: Operator

### [frontend-architecture.md](frontend-architecture.md)
**Purpose**: Describe the web UI architecture and its integration with API/SSE backends.

This page documents the frontend stack (React, TypeScript, Ring UI, Vite), application composition, and client data access patterns. It also explains SSE subscriptions, key UI capabilities, and the mapping between frontend actions and backend endpoints.

**Key topics**
- Frontend technology stack and build/runtime setup
- Component composition and app structure
- Data fetching/caching via API client and React Query
- SSE-driven live updates and event handling model
- Feature map (task lists, project stats, actions) and API interaction diagram

**Primary audience**: Developer

### [observability.md](observability.md)
**Purpose**: Define how metrics, logs, audits, and request tracing are exposed operationally.

This page outlines observability surfaces, including Prometheus metrics, structured logs with redaction, and sanitized audit trails for form submissions. It also documents request correlation and health/version endpoints used for runtime monitoring.

**Key topics**
- `/metrics` instrumentation and exported series
- Structured logging conventions in `internal/obslog`
- Audit log capture and sanitization path
- Request ID correlation behavior (`X-Request-ID`)
- Health and version observability endpoints

**Primary audience**: Operator

### [security.md](security.md)
**Purpose**: Document the current security model and trust boundaries for API and runtime operations.

This page explains authentication modes, path confinement controls, webhook signing, secret handling, and logging/audit safeguards. It also summarizes threat model assumptions and CI security posture around the repository and delivery pipeline.

**Key topics**
- Trust boundaries and threat assumptions
- API authentication and token model
- Path traversal defenses and filesystem confinement
- Webhook HMAC signing and secret usage patterns
- CI/GitHub security posture and risk summary

**Primary audience**: Architect

### [concurrency.md](concurrency.md)
**Purpose**: Describe concurrency controls across runner execution, planning queues, and message bus writes.

This page documents how Ralph Loop restart behavior, semaphores, and root-task planning limits interact to constrain throughput and isolation. It also covers message bus lock contention strategy, dependency DAG gating, and recovery behavior.

**Key topics**
- Global concurrency model and control points
- Run-level semaphore (`max_concurrent_runs`)
- Root-task planner FIFO queue (`max_concurrent_root_tasks`)
- Bus write locking, retry, and backoff behavior
- Dependency DAG cycle detection, gating, and failure recovery semantics

**Primary audience**: Developer
