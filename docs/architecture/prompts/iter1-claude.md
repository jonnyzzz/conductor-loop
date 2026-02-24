# Architecture Overview

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/overview.md`.

## Content Requirements
1. **Introduction**: What is `conductor-loop`? What problem does it solve? (Orchestration of AI agents, offline-first, resilient loops).
2. **System Overview**:
   - Multi-agent support (Claude, Codex, Gemini, Perplexity, xAI).
   - Offline-first design (filesystem as single source of truth).
   - Two binaries: `run-agent` (orchestration CLI) and `conductor` (API server).
3. **Key Design Principles**:
    - **Offline-First**: No database required. State is persisted in `run-info.yaml`, `TASK.md`, `DONE`, and message bus files.
    - **Resilience**: Ralph loop manages task restarts and recovery.
    - **Simplicity**: No auth required for local use. 
    - **Independence**: `run-agent` processes are fully independent; `run-agent serve` is optional for monitoring.
4. **Technology Stack**:
    - Backend: Go 1.24 (standard library + Cobra + YAML).
    - Frontend: React 18 + TypeScript + JetBrains Ring UI (`frontend/`) with vanilla JS fallback (`web/src/`).
    - Communication: REST API + Server-Sent Events (SSE).
    - Configuration: YAML (primary) with HCL support (legacy).
5. **Key Statistics**:
    - ~59k lines of Go code.
    - 16 major subsystems.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`
- `/Users/jonnyzzz/Work/conductor-loop/README.md` (top section)

## Instructions
- Read the source files to get accurate details.
- Write a professional, high-level overview.
- Keep it concise but comprehensive.
- Ensure the document is named `overview.md`.
