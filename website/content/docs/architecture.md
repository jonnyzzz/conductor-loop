---
title: "Architecture Overview"
description: "High-level structure of Conductor Loop"
weight: 30
group: "Core Docs"
---

## Runtime Shape

Conductor Loop has three primary layers:

1. Agent backends (Claude, Codex, Gemini, Perplexity, xAI)
2. Runner orchestration (Ralph Loop, run hierarchy, retries)
3. API and web monitoring UI

## Core Subsystems

- `internal/runner/`: task orchestration and lifecycle
- `internal/agent/` and `pkg/agent/`: backend protocol and agent implementations
- `pkg/storage/`: run/task/project directory layout
- `pkg/messagebus/`: append-only task/project communication stream
- `internal/api/`: REST and SSE interfaces
- `web/src/`: monitoring frontend

## Detailed Architecture Docs

- Repository architecture guide: [`docs/dev/architecture.md`](https://github.com/jonnyzzz/conductor-loop/blob/main/docs/dev/architecture.md)
- Subsystem specs: [`docs/specifications/`](https://github.com/jonnyzzz/conductor-loop/tree/main/docs/specifications)
