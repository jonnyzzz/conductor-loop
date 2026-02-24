# Architecture Documentation Index

This directory documents the implemented architecture of Conductor Loop from system, component, data-flow, deployment, operations, and decision viewpoints.

## Page Index

| Page | Description |
| --- | --- |
| `overview.md` | System context, design principles, and high-level architecture map. |
| `components.md` | Component boundaries, responsibilities, interfaces, and dependency graph. |
| `decisions.md` | Key architecture decisions and rationale/trade-offs. |
| `data-flow-task-lifecycle.md` | End-to-end task lifecycle from submit to completion propagation. |
| `data-flow-message-bus.md` | Message bus write/read/follow paths, IDs, and scope hierarchy. |
| `data-flow-api.md` | REST/SSE request lifecycle, middleware chain, and auth/path protections. |
| `agent-integration.md` | Agent backend execution modes, env contract, diversification, version checks. |
| `deployment.md` | Deployment topology, config precedence, storage layout, GC, self-update, ports. |
| `frontend-architecture.md` | React + Ring UI structure, data access, SSE model, and UI features. |
| `observability.md` | Metrics, structured logging, audit logging, request correlation, health/version. |
| `security.md` | Authentication model, path confinement, token handling, webhook signing, CI posture. |
| `concurrency.md` | Ralph loop, semaphores, planner queue, bus lock contention, dependency DAG gating. |

## Iteration 5 Validation Notes

- Cross-page consistency pass completed for defaults, ports, auth behavior, agent-mode split, and storage/message-bus semantics.
- Source references were re-checked against current implementation in `cmd/`, `internal/`, and `frontend/`.
- `FACTS-*` coverage is represented across these pages:
  - architecture and component model: `overview.md`, `components.md`, `decisions.md`
  - runner/storage lifecycle and DONE semantics: `data-flow-task-lifecycle.md`, `deployment.md`, `concurrency.md`
  - message-bus protocol and streaming: `data-flow-message-bus.md`, `data-flow-api.md`
  - agent backends and UI: `agent-integration.md`, `frontend-architecture.md`
  - ops/security/telemetry: `observability.md`, `security.md`
