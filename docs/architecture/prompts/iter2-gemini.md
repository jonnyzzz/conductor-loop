# Data Flow: Message Bus

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/data-flow-message-bus.md`.

## Content Requirements
1. **Write Path**:
    - Appending to `TASK-MESSAGE-BUS.md` or `PROJECT-MESSAGE-BUS.md`.
    - `O_APPEND` + exclusive `flock`.
    - Message ID generation (`MSG-<timestamp>-<pid>-<seq>`).
2. **Read Path**:
    - Lockless reads (on Unix).
    - `ReadMessages(sinceID)` polling for SSE.
    - `ReadLastN` for tailing.
3. **Hierarchy**:
    - Task Scope vs. Project Scope.
    - Discovery chain (`run-agent bus discover`).
4. **Diagram**: ASCII diagram showing Writer (Locked) vs Readers (Lockless).

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

## Instructions
- Explain the concurrency model clearly.
- Name the file `data-flow-message-bus.md`.
