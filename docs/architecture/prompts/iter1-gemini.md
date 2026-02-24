# Component Reference

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/components.md`.

## Content Requirements
1. **Component Inventory**:
    - **CLI**: `run-agent` (primary, task execution, bus, etc.) and `conductor` (API server entry point).
    - **API Server**: `internal/api/` - REST endpoints, SSE streaming, file serving.
    - **Runner (Orchestrator)**: `internal/runner/` - Ralph loop, process spawning, task dependency management.
    - **Message Bus**: `internal/messagebus/` - Append-only event log, O_APPEND + flock, lockless reads.
    - **Storage**: `internal/storage/` - Filesystem-based run metadata (`run-info.yaml`), atomic writes.
    - **Configuration**: `internal/config/` - YAML loading, token resolution.
    - **Agent Backends**: `internal/agent/` - Protocols for Claude, Codex, Gemini, Perplexity, xAI.
    - **Frontend**: React dashboard (`frontend/`) and fallback (`web/src/`).
    - **Webhook**: `internal/webhook/` - Notification delivery.
2. **Component Details**:
    - For each component, describe its **Responsibilities**.
    - List **Key Interfaces** (e.g., `Storage`, `MessageBus`, `Agent`) and **Key Files**.
3. **Dependencies**:
    - Describe the relationships (e.g., API Server depends on Storage and Message Bus; Runner depends on Storage, Message Bus, and Agents).

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md`

## Instructions
- Read the source files to get accurate component details.
- Ensure the document is named `components.md`.
- Focus on `internal/` package structure and logical boundaries.
