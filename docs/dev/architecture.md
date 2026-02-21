# Conductor Loop - Architecture Overview

This document provides a comprehensive overview of the conductor-loop system architecture, including component organization, data flow, design decisions, and performance considerations.

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Component Overview](#component-overview)
4. [Data Flow](#data-flow)
5. [Storage Layout](#storage-layout)
6. [Message Bus Architecture](#message-bus-architecture)
7. [Process Lifecycle](#process-lifecycle)
8. [Key Design Decisions](#key-design-decisions)
9. [Performance Considerations](#performance-considerations)
10. [Platform Support](#platform-support)

---

## System Overview

Conductor Loop is an AI agent orchestration system that manages multi-agent workflows. It provides:

- **Multi-Agent Support**: Claude, Codex (OpenAI), Gemini, Perplexity, and xAI (Grok)
- **Process Orchestration**: Automated restart loops for resilient task execution
- **Message Bus**: Append-only event log for task coordination
- **REST API + SSE**: Real-time task monitoring and control
- **Web UI**: React 18 + TypeScript dashboard (primary); plain HTML/CSS/JS fallback

**Key Statistics:**
- Backend: 11,276 lines of Go code
- 64 test files
- Frontend (primary): React 18 + TypeScript (`frontend/`, requires `npm run build`)
- Frontend (fallback): Vanilla JavaScript, no build step (`web/src/`)
- Minimal dependencies (Cobra, YAML v3, pkg/errors)

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Frontend (HTML/CSS/JS)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ TaskList │  │ RunDetail│  │LogViewer │  │MessageBus│       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
└───────┼─────────────┼─────────────┼─────────────┼──────────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                      │
                      │ HTTP/SSE
                      │
┌─────────────────────▼─────────────────────────────────────────┐
│                      API Server (REST + SSE)                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐              │
│  │  Handlers  │  │ SSE Streams│  │ Middleware │              │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘              │
└────────┼───────────────┼───────────────┼────────────────────┘
         │               │               │
         └───────────────┴───────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
         ▼               ▼               ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Storage   │  │ Message Bus │  │Orchestrator │
│  (YAML)     │  │ (O_APPEND)  │  │ (Runner)    │
└─────────────┘  └─────────────┘  └──────┬──────┘
                                          │
                                          ▼
                                  ┌───────────────┐
                                  │  Ralph Loop   │
                                  │ (Restart Mgr) │
                                  └───────┬───────┘
                                          │
                                          ▼
                                  ┌───────────────┐
                                  │Process Manager│
                                  └───────┬───────┘
                                          │
                ┌─────────────────────────┼─────────────────────────┐
                │                         │                         │
                ▼                         ▼                         ▼
        ┌──────────────┐        ┌──────────────┐        ┌──────────────┐
        │ Claude Agent │        │ Codex Agent  │        │ Other Agents │
        │   (CLI)      │        │   (API)      │        │ (Gemini/xAI) │
        └──────────────┘        └──────────────┘        └──────────────┘
```

---

## Component Overview

The system is organized into 16 major subsystems (see [Subsystem Deep-Dives](subsystems.md) for the full list). The 9 core architectural subsystems are:

### 1. Storage Layer (`internal/storage/`)

**Responsibilities:**
- Manage run metadata (YAML files)
- Atomic file operations (temp + fsync + rename)
- Run indexing and querying
- File locking for concurrent access

**Key Files:**
- `storage.go` - Main storage interface and implementation
- `atomic.go` - Atomic write operations
- `runinfo.go` - Run metadata structure

**Interface:**
```go
type Storage interface {
    CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
    UpdateRunStatus(runID string, status string, exitCode int) error
    GetRunInfo(runID string) (*RunInfo, error)
    ListRuns(projectID, taskID string) ([]*RunInfo, error)
}
```

### 2. Configuration System (`internal/config/`)

**Responsibilities:**
- Load YAML configuration
- Resolve agent tokens (direct, file, environment)
- Validate configuration
- Provide defaults

**Key Files:**
- `config.go` - Main config loader
- `tokens.go` - Token resolution
- `validation.go` - Config validation

**Configuration Structure:**
```go
type Config struct {
    Agents   map[string]AgentConfig
    Defaults DefaultConfig
    API      APIConfig
    Storage  StorageConfig
}
```

### 3. Message Bus (`internal/messagebus/`)

**Responsibilities:**
- Append-only event log
- Lockless reads, exclusive writes
- Message ID generation (lexically sortable)
- Poll-based event streaming

**Key Files:**
- `messagebus.go` - Core message bus
- `msgid.go` - Message ID generation
- `lock_unix.go`, `lock_windows.go` - Platform-specific locking

**Architecture:**
- O_APPEND for atomic writes
- flock for exclusive write lock
- OS page cache (no fsync) for high throughput
- Lockless reads for performance

### 4. Agent Protocol (`internal/agent/`)

**Responsibilities:**
- Define agent interface
- Execute agent CLI processes
- Capture stdout/stderr
- Manage process lifecycle

**Key Files:**
- `agent.go` - Agent interface
- `executor.go` - Execution logic
- `stdio.go` - I/O capture
- Backend implementations: `claude/`, `codex/`, `gemini/`, `perplexity/`, `xai/`

**Agent Interface:**
```go
type Agent interface {
    Execute(ctx context.Context, runCtx *RunContext) error
    Type() string
}
```

### 5. Agent Backends (`internal/agent/*/`)

**Supported Agents:**
1. **Claude** - Direct CLI invocation
2. **Codex** - OpenAI-compatible endpoint
3. **Gemini** - Google's Gemini API
4. **Perplexity** - Custom endpoint support
5. **xAI** - Grok backend

Each backend implements the `Agent` interface with specific API integration.

### 6. Runner Orchestration (`internal/runner/`)

**Responsibilities:**
- Task execution coordination
- Ralph loop (restart logic)
- Process spawning and management
- Child process tracking (PGID)
- Graceful shutdown

**Key Files:**
- `orchestrator.go` - Main orchestration
- `ralph.go` - Ralph loop
- `process.go` - Process spawning
- `task.go` - Task execution
- Platform-specific: `wait_*.go`, `pgid_*.go`, `stop_*.go`

**Ralph Loop Configuration:**
```go
type RalphConfig struct {
    MaxRestarts   int           // Default: 100
    WaitTimeout   time.Duration // Default: 5 minutes
    PollInterval  time.Duration // Default: 1 second
    RestartDelay  time.Duration // Default: 1 second
}
```

### 7. API Server (`internal/api/`)

**Responsibilities:**
- REST API endpoints
- Server-Sent Events (SSE) streaming
- Static file serving
- Request validation
- CORS handling

**Key Files:**
- `server.go` - HTTP server setup
- `handlers.go` - Request handlers
- `sse.go` - SSE streaming
- `routes.go` - Route definitions
- `middleware.go` - HTTP middleware

**Endpoint Categories:**
- Projects: List, get details
- Tasks: List, create, get details
- Runs: Get info, stop, read files
- Message Bus: Read/write messages, streaming
- Logs: Real-time log streaming

### 8. Webhook Notifications (`internal/webhook/`)

**Responsibilities:**
- Deliver run completion notifications to external HTTP endpoints
- HMAC-SHA256 signed payloads for authenticity
- Async delivery with retry (3 attempts, exponential backoff)

**Key Files:**
- `webhook.go` - Notifier and HTTP delivery logic
- `config.go` - Webhook configuration types

**Configuration:**
```yaml
webhook:
  url: https://your-endpoint.example.com/hook
  events: [run_completed]
  secret: your-hmac-secret
  timeout: 10s
```

### 9. Frontend UI

Two UIs are available; the conductor server serves whichever is present at startup.

#### Active (simple) UI — `web/src/`

**Technology Stack:**
- Plain HTML/CSS/JS (vanilla JavaScript, no framework)
- No npm, no build step, no TypeScript

**Key Files:**
- `web/src/index.html` - Main single-page application
- `web/src/app.js` - Application logic (vanilla JS)
- `web/src/styles.css` - Styles

**Key UI Features:**
- Task list and project sidebar
- Run detail display
- Real-time log streaming via SSE
- Message bus feed

#### Advanced React UI — `frontend/`

**Technology Stack:**
- React 18 + TypeScript
- Vite build tool
- Ring UI (JetBrains) component library

**Build:**
```bash
cd frontend && npm install && npm run build
# Output: frontend/dist/
```

**Key Features:**
- LogViewer with filtering
- RunTree visualization
- TypeScript type-safe API client

**Priority:** When `frontend/dist/index.html` is present, the conductor server serves it instead of `web/src/`. Delete or move `frontend/dist/` to fall back to the simple UI.

### 10–16. Extended Subsystems

Additional subsystems built on top of the core 9:

| # | Subsystem | Description |
|---|-----------|-------------|
| 10 | CLI: `run-agent list` | Filesystem-only listing of projects, tasks, and runs |
| 11 | CLI: `run-agent output` | Print or live-tail agent output files |
| 12 | CLI: `run-agent watch` | Poll until all specified tasks reach a terminal state |
| 13 | API: DELETE Run Endpoint | `DELETE /api/projects/{p}/tasks/{t}/runs/{r}` — remove a single run directory |
| 14 | UI: Task Search Bar | Client-side substring filtering of the task list |
| 15 | API: Task Deletion Endpoint | `DELETE /api/projects/{p}/tasks/{t}` — remove an entire task directory; CLI: `run-agent task delete` |
| 16 | UI: Project Stats Dashboard | `ProjectStats.tsx` — task/run count bar sourced from `GET /api/projects/{p}/stats` |

See [Subsystem Deep-Dives](subsystems.md) for detailed documentation on each.

---

## Data Flow

### Task Execution Flow

```
1. Frontend: POST /api/projects/{id}/tasks
   - TaskStartRequest: { task_id, prompt, project_root, attach_mode }
   │
   ▼
2. API Handler: parseTaskCreateRequest()
   - Validate input
   - Extract task parameters
   │
   ▼
3. Orchestrator.ExecuteTask()
   - Load configuration
   - Select agent (from defaults or request)
   - Resolve working directory
   │
   ▼
4. Storage.CreateRun()
   - Generate RunID: {timestamp}-{nano}-{pid}
   - Create run directory
   - Write run-info.yaml atomically
   │
   ▼
5. MessageBus.AppendMessage("task_started")
   - ExclusiveLock()
   - Append message with O_APPEND
   - fsync()
   - Unlock()
   │
   ▼
6. RalphLoop.Run()
   - Initialize restart counter
   - Enter restart loop
   │
   ▼
7. ProcessManager.SpawnAgent()
   - Create process group (setpgid)
   - Redirect stdout/stderr to files
   - Execute agent CLI
   │
   ▼
8. Agent Execution
   - Process prompt
   - Write output
   - Exit with status code
   │
   ▼
9. Wait for Completion
   - waitpid() or WaitForSingleObject()
   - Capture exit code
   - Check for restart conditions
   │
   ▼
10. Storage.UpdateRunStatus()
    - Update run-info.yaml
    - Set status, exit_code, end_time
    │
    ▼
11. MessageBus.AppendMessage("task_completed")
    - Log completion event
    │
    ▼
12. Frontend Polls: GET /api/runs/{runId}
    - Fetch updated run info
    - Display results
```

### Message Bus Streaming Flow

```
1. Frontend: EventSource("/api/projects/{id}/bus/stream")
   - Establish SSE connection
   │
   ▼
2. Server: StreamManager.Subscribe()
   - Create channel for events
   - Register client
   │
   ▼
3. Server: Poll Loop
   - Every 100ms: messagebus.PollForNew()
   - Detect new messages
   │
   ▼
4. New Messages Detected
   - Read messages from append-only log
   - Filter by project/task if needed
   │
   ▼
5. Send SSE Events
   - Format: "data: {json}\n\n"
   - Send heartbeat every 30s
   │
   ▼
6. Browser Receives Events
   - EventSource.onmessage handler
   - Parse JSON
   │
   ▼
7. Update UI
   - Re-render components
   - Show new messages in real-time
```

---

## Storage Layout

### Directory Structure

```
{storage_root}/
├── {project_id}/
│   ├── {task_id}/
│   │   ├── TASK.md                 # Task prompt and metadata
│   │   ├── DONE                    # Completion marker
│   │   ├── messagebus.yaml         # Task message bus
│   │   └── runs/
│   │       ├── {run_id}/
│   │       │   ├── run-info.yaml   # Run metadata (YAML)
│   │       │   ├── stdout          # Agent stdout
│   │       │   ├── stderr          # Agent stderr
│   │       │   ├── output.md       # Final output
│   │       │   └── messagebus.yaml # Run message bus
│   │       └── {run_id_2}/
│   │           └── ...
│   └── {task_id_2}/
│       └── ...
└── {project_id_2}/
    └── ...
```

### Run Info Schema

**File:** `run-info.yaml`

```yaml
run_id: MSG-20060102-150405-000000001-PID00123-0001
project_id: my-project
task_id: task-001
agent_type: claude
pid: 12345
pgid: 12345              # Process group ID (for child tracking)
status: running          # running, success, failed, stopped
start_time: 2026-02-05T10:00:00Z
end_time: null           # Set on completion
exit_code: null          # Set on completion
```

**Status Values:**
- `running` - Task is currently executing
- `success` - Completed successfully (exit code 0)
- `failed` - Failed with non-zero exit code
- `stopped` - Manually stopped by user

### Atomic Write Pattern

All metadata writes use the atomic pattern to prevent corruption:

```go
1. Create temporary file: run-info.yaml.tmp.{pid}
2. Write data to temp file
3. fsync() - Ensure data is on disk
4. Rename temp to final: run-info.yaml
   (Rename is atomic on POSIX systems)
```

This ensures readers always see complete, valid data even during concurrent writes.

---

## Message Bus Architecture

### Design Philosophy

The message bus uses an **append-only file** with **O_APPEND + flock** for safe concurrent writes:

- **Lockless Reads**: Readers don't block writers
- **Exclusive Writes**: Only one writer at a time (via flock)
- **High Throughput**: OS-cached writes without fsync (~37,000+ msg/sec)
- **Ordering**: Messages are totally ordered by timestamp

### Message Structure

```go
type Message struct {
    MsgID        string    // Unique ID (lexically sortable)
    Timestamp    time.Time // UTC timestamp
    Type         string    // Event type
    ProjectID    string    // Which project
    TaskID       string    // Which task (optional)
    RunID        string    // Which run (optional)
    ParentMsgIDs []string  // Parent message IDs
    Attachment   string    // Path to attached file
    Body         string    // Message content
}
```

### Message ID Format

```
MSG-{YYYYMMDD-HHMMSS}-{NANOSECONDS}-PID{PID}-{SEQUENCE}

Example: MSG-20060102-150405-000000001-PID00123-0042
```

**Properties:**
- Lexically sortable (for range queries)
- Globally unique (timestamp + PID + sequence)
- Human-readable timestamp

### Concurrency Model

**Write Path:**
```go
1. Open file: O_WRONLY | O_APPEND | O_CREATE
2. ExclusiveLock(file, timeout=10s)
3. Serialize message to YAML
4. Write to file (O_APPEND ensures atomic append)
5. fsync() - Force disk write
6. Unlock()
7. Close file
```

**Read Path (Lockless):**
```go
1. ReadFile(messagebus.yaml) - No lock needed!
2. Parse YAML documents (--- separator)
3. Filter by since_id
4. Return messages
```

**Poll Path:**
```go
1. Loop:
   - ReadMessages(since_id)
   - If empty: sleep(200ms)
   - If new messages: return
   - Check timeout
```

### File Format

**YAML with Document Separators:**

```yaml
---
msg_id: MSG-20060102-150405-000000001-PID00123-0001
ts: 2026-02-05T10:00:00Z
type: agent_started
project_id: my-project
task_id: task-001
run_id: MSG-20060102-150405-000000001
---
Agent started successfully

---
msg_id: MSG-20060102-150406-000000002-PID00123-0002
ts: 2026-02-05T10:00:01Z
type: agent_output
project_id: my-project
task_id: task-001
run_id: MSG-20060102-150405-000000001
---
Processing request...
```

---

## Process Lifecycle

### Ralph Loop (Restart Manager)

The Ralph Loop manages the lifecycle of root agent processes with automatic restart capabilities.

```
┌─────────────────────────────────────────────────────────────┐
│                      Ralph Loop                             │
│                                                             │
│  1. Initialize                                              │
│     - restartCount = 0                                      │
│     - maxRestarts = 100                                     │
│                                                             │
│  2. Restart Loop                                            │
│     ┌────────────────────────────────────────┐             │
│     │ while restartCount < maxRestarts:      │             │
│     │                                        │             │
│     │   a. SpawnAgent()                      │             │
│     │      - Create process group            │             │
│     │      - Redirect stdout/stderr          │             │
│     │      - Start agent process             │             │
│     │                                        │             │
│     │   b. Wait for completion               │             │
│     │      - waitpid() or equivalent         │             │
│     │      - Capture exit code               │             │
│     │                                        │             │
│     │   c. Check exit conditions:            │             │
│     │      - Success (exit=0) → STOP         │             │
│     │      - Fatal error → STOP              │             │
│     │      - DONE file exists → STOP         │             │
│     │      - Wait-without-restart → STOP     │             │
│     │      - Otherwise → RESTART             │             │
│     │                                        │             │
│     │   d. Delay before restart (1s)         │             │
│     │                                        │             │
│     │   e. restartCount++                    │             │
│     │                                        │             │
│     └────────────────────────────────────────┘             │
│                                                             │
│  3. Cleanup                                                 │
│     - Kill process group if needed                          │
│     - Update run status                                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Exit Conditions

**Stop (no restart):**
1. Exit code 0 (success)
2. Max restarts exceeded
3. DONE file detected in task directory
4. Wait-without-restart signal received
5. Fatal error from agent

**Restart:**
1. Non-zero exit code (< max restarts)
2. Agent crash or timeout
3. No DONE file present

### Process Group Management

**Unix/Linux/macOS:**
```go
// Set process group ID (PGID) = PID
syscall.Setpgid(0, 0)

// Kill entire process group
syscall.Kill(-pgid, syscall.SIGTERM)
```

**Benefits:**
- Kill all child processes together
- Prevent orphan processes
- Clean process tree termination

**Windows:**
- Limited support (no process groups)
- Recommendation: Use WSL2

---

## Key Design Decisions

### 1. Append-Only Message Bus

**Decision:** Use O_APPEND + flock instead of database

**Rationale:**
- Simple: No external dependencies
- Fast: Lockless reads
- Durable: fsync ensures persistence
- Human-readable: YAML format for debugging

**Trade-offs:**
- File size grows (mitigation: `WithAutoRotate` option and `run-agent gc --rotate-bus`)
- No complex queries (mitigation: simple filtering in code)
- Network filesystems may have issues (mitigation: local storage only)

### 2. YAML for Metadata

**Decision:** Use YAML instead of JSON or Protocol Buffers

**Rationale:**
- Human-readable for debugging
- Comments supported (for documentation)
- Standard library support (gopkg.in/yaml.v3)
- No compilation step needed

**Trade-offs:**
- Slower parsing than binary formats (acceptable for metadata)
- Larger file size (acceptable for small files)

### 3. File-Based Storage

**Decision:** Use filesystem instead of database

**Rationale:**
- Simple: No database setup/maintenance
- Portable: Works everywhere
- Debuggable: Can inspect with standard tools
- Atomic operations: Use rename for atomicity

**Trade-offs:**
- No complex queries (mitigation: in-memory indexing)
- Scaling limits (acceptable for single-node deployment)

### 4. Process Groups (PGID)

**Decision:** Use process groups for child tracking

**Rationale:**
- Kill all children together
- No orphan processes
- Clean shutdown

**Trade-offs:**
- Platform-specific (Unix only)
- Requires careful setup

### 5. Server-Sent Events (SSE)

**Decision:** Use SSE instead of WebSockets

**Rationale:**
- Simpler: HTTP-based (no protocol upgrade)
- Sufficient: Unidirectional updates are enough
- Reliable: Automatic reconnection

**Trade-offs:**
- No bidirectional communication (acceptable: REST for commands)
- Browser support (acceptable: modern browsers)

### 6. Minimal Dependencies

**Decision:** Avoid heavy frameworks and databases

**Rationale:**
- Easy deployment: Single binary
- Fast startup: No initialization
- Maintainable: Less code to maintain

**Trade-offs:**
- More custom code (acceptable: simple use cases)

---

## Performance Considerations

### 1. Message Bus

**Current Performance:**
- Write latency: <0.1ms (OS-cached, no fsync)
- Read latency: ~0.1-1ms (lockless)
- Throughput: ~37,000+ writes/sec measured with 10 concurrent writers

**Bottlenecks:**
- flock contention under very high concurrent load
- Linear scan on read (no indexing)

**Optimizations:**
- Lockless reads (no read blocking)
- Per-task message buses (reduce contention)
- Message indexing for large files

### 2. Storage Layer

**Current Performance:**
- Run creation: ~5-10ms (atomic write + fsync)
- Run query: ~0.1-1ms (in-memory index)
- List runs: ~1-10ms (depends on count)

**Bottlenecks:**
- File I/O for run-info.yaml reads
- Directory scanning for run discovery

**Optimizations:**
- In-memory index (RunIndex) for fast lookups
- RWMutex for concurrent reads
- Lazy loading of run details

### 3. API Server

**Current Performance:**
- Request latency: ~1-5ms (simple endpoints)
- SSE stream latency: ~100ms (poll interval)
- Concurrent clients: 10 per run (configurable)

**Bottlenecks:**
- File I/O for log tailing
- Poll frequency for SSE updates

**Optimizations:**
- Tail-only reads (last N lines)
- Configurable poll intervals
- Client limits per run

### 4. Ralph Loop

**Current Performance:**
- Restart delay: 1s (configurable)
- Poll interval: 1s (for DONE file)
- Max restarts: 100 (configurable)

**Bottlenecks:**
- Process spawning overhead (~10-100ms)
- Wait latency (depends on agent execution time)

**Optimizations:**
- Configurable delays and timeouts
- Process group for fast cleanup
- Early exit on success

### 5. Scaling Limits

**Single Node:**
- Concurrent tasks: 100+ (limited by CPU/memory)
- Concurrent agents: 50+ (limited by lock contention)
- Message bus size: ~1GB (before rotation needed)
- Storage size: Unlimited (depends on disk)

**Multi-Node:**
- Not currently supported (file-based storage)
- Future: Distributed storage backend

---

## Platform Support

### Unix/Linux/macOS

**Fully Supported:**
- Process groups (setpgid, kill -PGID)
- File locking (flock, fcntl)
- O_APPEND atomic writes
- Signal handling (SIGTERM, SIGKILL)

**Tested Platforms:**
- macOS (Darwin)
- Linux (Ubuntu, Debian, RHEL)
- BSD variants

### Windows

**Partial Support:**
- Limited process group support
- Mandatory file locks (breaks lockless reads)
- No O_APPEND guarantees (filesystem-dependent)

**Recommendations:**
- Use WSL2 for Windows users
- Native Windows support marked as experimental

**Known Issues:**
- Process cleanup may leave orphans
- File locking may block readers
- O_APPEND may not be atomic on network drives

---

## Next Steps

For more detailed information, see:

- [Subsystem Deep-Dives](subsystems.md)
- [Agent Protocol Specification](agent-protocol.md)
- [Ralph Loop Specification](ralph-loop.md)
- [Message Bus Protocol](message-bus.md)
- [Storage Layout Specification](storage-layout.md)
- [Contributing Guide](contributing.md)
- [Testing Guide](testing.md)

---

**Last Updated:** 2026-02-05
**Version:** 1.0.0
