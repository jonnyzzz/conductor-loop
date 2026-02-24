# Data Flow Review Notes - B

## Overview
Reviewed `data-flow-task-lifecycle.md`, `data-flow-message-bus.md`, and `data-flow-api.md`.

## Findings

### 1. Contradictions
- **Locking:** The API and Message Bus documents are consistent regarding locking. `data-flow-message-bus.md` specifies that reads are lockless on Unix, and `data-flow-api.md` implicitly follows this by not requiring locks for its SSE and REST read paths.
- **Path Resolution:** There is a slight difference in how paths are resolved: CLI uses upward auto-discovery (`data-flow-message-bus.md`), while the API uses a depth-pruned downward walk (`data-flow-api.md`). This appears to be an intentional design choice for server-side vs. client-side discovery but is worth noting.

### 2. Missing Steps / Clarifications
- **Task Lifecycle (Phase 3 - Execution):** While Phase 5 mentions "Fact Propagation," Phase 3 could be more explicit about Agents posting `FACT`, `PROGRESS`, and `DECISION` messages to the bus. Currently, it only mentions runner-posted lifecycle events (`RUN_START`, etc.). Adding this would better ground the source of the data propagated in Phase 5.
- **Task Lifecycle (Phase 3 - Output):** The document states the runner "guarantees `output.md` exists," but it could clarify that the Agent is the primary writer of the content, while the runner provides the fallback/guarantee.
- **API (Windows Locking):** `data-flow-message-bus.md` mentions that readers on Windows can block due to mandatory locking. `data-flow-api.md` does not mention this, which could impact API responsiveness on Windows. Adding a brief note about OS-specific blocking behavior in the API doc would be beneficial.

### 3. Fact Propagation
- **Confirmation:** `data-flow-task-lifecycle.md` correctly identifies Fact Propagation in Phase 5. It describes the synthesis of task-level facts into the project-level bus, which aligns with the hierarchical structure described in `data-flow-message-bus.md`.

## Conclusion
Data Flow: LGTM (with minor suggestions for Phase 3 detail).
