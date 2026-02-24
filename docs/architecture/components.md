# Component Reference

This document details the major software components of the `conductor-loop` system, their responsibilities, key interfaces, and dependencies. The system is designed with a "filesystem-first" philosophy, where components interact primarily through shared files and atomic operations.

## Component Inventory

The core logic resides in `internal/`, while entry points are in `cmd/`.

### 1. CLI Entry Points (`cmd/`)

**Components:**
- **`run-agent`**: The primary CLI tool for task execution, orchestration, and management. It handles job submission (`run-agent job`), direct task execution (`run-agent task`), bus interactions (`run-agent bus`), and garbage collection (`run-agent gc`).
- **`conductor`**: A dedicated entry point for the API server. It starts the REST/SSE server directly and is distinct from the `run-agent` CLI.

**Responsibilities:**
- Parse command-line arguments (Cobra).
- Load configuration.
- Instantiate and wire together internal components (Storage, MessageBus, Runner).
- `run-agent serve` and `conductor` expose the API server.

**Key Files:**
- `cmd/run-agent/main.go`
- `cmd/conductor/main.go`

---

### 2. Storage Layer (`internal/storage/`)

**Responsibilities:**
- Manage persistent run metadata (`run-info.yaml`) on the filesystem.
- Ensure data integrity via atomic file writes (write-sync-rename).
- Maintain an in-memory index of runs for fast lookups.
- Handle run status updates (`running`, `completed`, `failed`).

**Key Interface:**
```go
type Storage interface {
    CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
    UpdateRunStatus(runID string, status string, exitCode int) error
    GetRunInfo(runID string) (*RunInfo, error)
    ListRuns(projectID, taskID string) ([]*RunInfo, error)
}
```

**Key Files:**
- `storage.go`: Main `FileStorage` implementation and interface definition.
- `atomic.go`: Primitives for atomic file writing (`os.CreateTemp` + `os.Rename`).
- `runinfo.go`: Struct definition for `RunInfo` and YAML serialization.

---

### 3. Message Bus (`internal/messagebus/`)

**Responsibilities:**
- Provide an append-only, chronologically ordered event log for projects and tasks.
- Guarantee atomic writes using `O_APPEND` and exclusive file locking (`flock`).
- Support lockless reads for high-throughput monitoring.
- Generate globally unique, lexically sortable message IDs.

**Key Interface:**
```go
type MessageBus interface {
    AppendMessage(msg *Message) (string, error)
    ReadMessages(path string, sinceID string) ([]*Message, error)
}
```

**Key Files:**
- `messagebus.go`: Core logic for appending and reading messages.
- `msgid.go`: ID generation logic (`MSG-<timestamp>-<pid>-<seq>`).
- `lock_unix.go` / `lock_windows.go`: Platform-specific file locking.

---

### 4. Configuration (`internal/config/`)

**Responsibilities:**
- Load and validate system configuration from YAML files.
- Resolve API tokens from multiple sources (direct, file, environment variables).
- Provide default settings for agents, timeouts, and concurrency.
- Resolve paths (e.g., expanding `~`).

**Key Structures:**
- `Config`: Root configuration object.
- `AgentConfig`: Per-agent settings (token, model, url).

**Key Files:**
- `config.go`: Loader logic (`LoadConfig`) and struct definitions.
- `tokens.go`: Logic to resolve tokens from files or env vars (`CONDUCTOR_AGENT_<NAME>_TOKEN`).

---

### 5. Agent Backends (`internal/agent/`)

**Responsibilities:**
- Define the abstract protocol for AI agents.
- Implement specific backends for supported providers (Claude, Codex, Gemini, Perplexity, xAI).
- Execute agent processes, capturing `stdout` and `stderr`.
- Inject execution context (environment variables, working directory).

**Key Interface:**
```go
type Agent interface {
    Execute(ctx context.Context, runCtx *RunContext) error
    Type() string
}
```

**Key Files:**
- `agent.go`: Interface definition and `RunContext` struct.
- `executor.go`: Common process spawning and I/O handling logic.
- `claude/`, `codex/`, `gemini/`, `perplexity/`, `xai/`: Backend implementations.

---

### 6. Runner / Orchestrator (`internal/runner/`)

**Responsibilities:**
- Orchestrate the lifecycle of a task run.
- Implement the "Ralph Loop" (automatic restart logic).
- Manage process groups (PGID) for reliable process termination.
- Detect task completion via the `DONE` file.
- Enforce concurrency limits and handle task dependencies.

**Key Files:**
- `orchestrator.go`: High-level coordination (wiring storage, bus, and agent).
- `ralph.go`: The restart loop logic (`maxRestarts`, `waitTimeout`).
- `process.go`: Low-level process spawning and PGID management.
- `task.go`: Task execution flow and dependency resolution.

---

### 7. API Server (`internal/api/`)

**Responsibilities:**
- Expose system state via a REST API.
- Provide real-time updates via Server-Sent Events (SSE).
- Serve static assets for the frontend UI.
- Handle request validation and log tailing.

**Key Files:**
- `server.go`: HTTP server initialization and routing.
- `handlers.go`: API endpoint implementations.
- `sse.go`: SSE stream management and client registration.
- `handlers_projects.go`: Project and task-related endpoints.

---

### 8. Frontend

**Responsibilities:**
- Provide a visual interface for monitoring and control.
- Stream logs and message bus events in real-time.

**Components:**
- **Primary (`frontend/`)**: A modern React 18 + TypeScript application. Built into `frontend/dist`.
- **Fallback (`web/src/`)**: A lightweight, vanilla HTML/JS dashboard served if the React build is absent.

---

### 9. Webhook (`internal/webhook/`)

**Responsibilities:**
- Deliver asynchronous notifications to external systems (e.g., on run completion).
- Sign payloads using HMAC-SHA256 for security.
- Handle retries with exponential backoff.

**Key Files:**
- `webhook.go`: Notification delivery and signing logic.

---

## Component Dependencies

The system is layered, with `cmd/` depends on everything, and `internal/` packages having specific relationships:

1.  **Runner (`internal/runner/`) depends on:**
    -   **Storage**: To create runs and update their status (`running` -> `completed`).
    -   **Message Bus**: To publish lifecycle events (`task_started`, `task_completed`).
    -   **Agent**: To execute the actual AI workload.
    -   **Config**: For limits, timeouts, and agent selection.

2.  **API Server (`internal/api/`) depends on:**
    -   **Storage**: To read run metadata and list projects/tasks.
    -   **Message Bus**: To read/poll for messages and stream them via SSE.
    -   **Runner**: To trigger "stop" actions (though it primarily observes state).
    -   **Config**: For server settings (port, host).

3.  **Agent (`internal/agent/`) depends on:**
    -   **Config**: For API tokens and model settings.

4.  **CLI (`cmd/`) depends on:**
    -   All of the above to assemble the application.

```mermaid
graph TD
    CLI[CLI (cmd/)] --> Runner
    CLI --> API
    
    API[API Server] --> Storage
    API --> MessageBus
    API --> Runner
    
    Runner[Orchestrator] --> Storage
    Runner --> MessageBus
    Runner --> Agent
    
    Agent[Agent Backends] --> Config
    Runner --> Config
    API --> Config
    
    subgraph Data Layer
    Storage[Storage (Filesystem)]
    MessageBus[Message Bus (O_APPEND)]
    end
```
