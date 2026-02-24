# Conductor Loop — Architecture Documentation

This directory contains architecture documentation for **conductor-loop**, a Go multi-agent orchestration framework. Start with [`overview.md`](overview.md) if you are new to the system.

---

## Table of Contents

### Overview

| Document | Description |
|----------|-------------|
| [overview.md](overview.md) | Primary entry point: what conductor-loop is, design principles, storage model, Ralph Loop, and 16-subsystem inventory. |
| [decisions.md](decisions.md) | Architecture Decision Records (ADRs) explaining the rationale behind filesystem-first storage, message bus design, agent protocol choices, and more. |

### Data Flow

| Document | Description |
|----------|-------------|
| [data-flow-task-lifecycle.md](data-flow-task-lifecycle.md) | End-to-end sequence from task submission through Ralph Loop restarts to `DONE` completion and project-level propagation. |
| [data-flow-message-bus.md](data-flow-message-bus.md) | Message append, lockless read, and SSE streaming flows across project and task bus scopes. |
| [data-flow-api.md](data-flow-api.md) | HTTP request lifecycle: middleware chain, routing, path safety, REST handlers, and SSE streaming. |

### Subsystems

| Document | Description |
|----------|-------------|
| [components.md](components.md) | Per-component interface, responsibility, and dependency reference for all major subsystems (`runner`, `storage`, `messagebus`, `api`, `agent`, etc.). |
| [agent-integration.md](agent-integration.md) | How the runner invokes CLI backends (Claude, Codex, Gemini) and REST backends (Perplexity, xAI), including env injection and diversification. |
| [frontend-architecture.md](frontend-architecture.md) | React 18 primary UI and vanilla JS fallback: build pipeline, Go binary embedding, and SSE-driven live updates. |
| [concurrency.md](concurrency.md) | Semaphores, `flock` write locking, FIFO root-task planner queue, and dependency DAG cycle detection and gating. |

### Operations

| Document | Description |
|----------|-------------|
| [deployment.md](deployment.md) | Single-binary deployment model, on-disk directory hierarchy, garbage collection, self-update flow, and Docker/compose setup. |
| [observability.md](observability.md) | Prometheus metrics, structured logging with redaction, audit log, request ID correlation, and health/version endpoints. |
| [security.md](security.md) | Authentication modes, path traversal defenses, HMAC webhook signing, secret handling, and threat model summary. |

---

## Quick Links

| If you want to… | Read… |
|-----------------|-------|
| Understand what conductor-loop is | [overview.md](overview.md) |
| Find a specific component | [components.md](components.md) |
| Understand why key choices were made | [decisions.md](decisions.md) |
| Trace a task from submit to DONE | [data-flow-task-lifecycle.md](data-flow-task-lifecycle.md) |
| Understand the message bus | [data-flow-message-bus.md](data-flow-message-bus.md) |
| Understand the REST API | [data-flow-api.md](data-flow-api.md) |
| Learn how agents are invoked | [agent-integration.md](agent-integration.md) |
| Deploy conductor-loop | [deployment.md](deployment.md) |
| Understand the frontend | [frontend-architecture.md](frontend-architecture.md) |
| Set up monitoring | [observability.md](observability.md) |
| Understand security | [security.md](security.md) |
| Understand concurrency | [concurrency.md](concurrency.md) |
