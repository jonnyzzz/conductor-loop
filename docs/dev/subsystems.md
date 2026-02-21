# Subsystem Deep-Dives

This document provides detailed information about each subsystem in the conductor-loop architecture, including purpose, interfaces, implementation details, dependencies, and testing strategies.

## Table of Contents

1. [Storage Layer](#1-storage-layer)
2. [Configuration System](#2-configuration-system)
3. [Message Bus](#3-message-bus)
4. [Agent Protocol](#4-agent-protocol)
5. [Agent Backends](#5-agent-backends)
6. [Runner Orchestration](#6-runner-orchestration)
7. [API Server](#7-api-server)
8. [Frontend UI](#8-frontend-ui)
9. [Webhook Notifications](#9-webhook-notifications)
10. [CLI Commands: run-agent list](#10-cli-commands-run-agent-list)
11. [CLI Commands: run-agent output](#11-cli-commands-run-agent-output)
12. [CLI Commands: run-agent watch](#12-cli-commands-run-agent-watch)
13. [API: DELETE Run Endpoint](#13-api-delete-run-endpoint)
14. [UI: Task Search Bar](#14-ui-task-search-bar)
15. [API: Task Deletion Endpoint](#15-api-task-deletion-endpoint)
16. [UI: Project Stats Dashboard](#16-ui-project-stats-dashboard)

---

## 1. Storage Layer

**Package:** `internal/storage/`

### Purpose

The storage layer manages persistent run metadata using the filesystem. It provides:
- Run creation and metadata persistence
- Run status updates
- Run querying and listing
- Atomic file operations to prevent corruption

### Key Files

- `storage.go` - Main storage interface and `FileStorage` implementation
- `atomic.go` - Atomic write operations (temp + fsync + rename)
- `runinfo.go` - Run metadata structure and serialization

### Interface

```go
type Storage interface {
    CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
    UpdateRunStatus(runID string, status string, exitCode int) error
    GetRunInfo(runID string) (*RunInfo, error)
    ListRuns(projectID, taskID string) ([]*RunInfo, error)
}
```

### Implementation: FileStorage

**Data Structure:**
```go
type FileStorage struct {
    root     string                // Storage root directory
    now      func() time.Time      // Clock (injectable for testing)
    pid      func() int            // PID function (injectable for testing)
    runIndex map[string]string     // RunID → path mapping (in-memory index)
    mu       sync.RWMutex          // Protects runIndex
}
```

**RunID Generation:**

Reference: `internal/storage/storage.go:186-190`

```go
func (s *FileStorage) newRunID() string {
    now := s.now().UTC()
    stamp := now.Format("20060102-150405000")  // YYYYMMDDHHMMSSmmm
    return fmt.Sprintf("%s-%d", stamp, s.pid())
}
```

Format: `{YYYYMMDD-HHMMSSmmm}-{PID}`
Example: `20260205-150405123-12345`

**Run Creation Flow:**

Reference: `internal/storage/storage.go:47-83`

1. Validate inputs (projectID, taskID, agentType)
2. Generate unique RunID
3. Create directory: `{root}/{projectID}/{taskID}/runs/{runID}/`
4. Build RunInfo structure
5. Write `run-info.yaml` atomically (see atomic.go)
6. Add to in-memory index
7. Return RunInfo

**Atomic Write Pattern:**

Reference: `internal/storage/atomic.go`

```go
1. Create temp file: run-info.yaml.tmp.{pid}
2. Write YAML data to temp file
3. fsync() - Ensure data is on disk
4. Rename temp to final: run-info.yaml
   (Rename is atomic on POSIX systems)
```

This ensures readers always see complete, valid data.

**Run Index:**

Reference: `internal/storage/storage.go:173-184`

- In-memory cache: `map[string]string` (RunID → path)
- Protected by `sync.RWMutex` for concurrent access
- Populated on CreateRun and on-demand during lookups
- Speeds up GetRunInfo operations (avoid filesystem scanning)

**Run Lookup Strategy:**

Reference: `internal/storage/storage.go:154-171`

1. Check in-memory index first (fast path)
2. If not found, glob: `{root}/*/*/runs/{runID}/run-info.yaml`
3. Add to index for future lookups
4. Error if not found or multiple matches

### RunInfo Structure

```go
type RunInfo struct {
    RunID         string    // Unique run identifier
    ProjectID     string    // Project identifier
    TaskID        string    // Task identifier
    AgentType     string    // Agent backend type
    PID           int       // Process ID
    PGID          int       // Process group ID
    Status        string    // running, completed, failed
    StartTime     time.Time // UTC start time
    EndTime       time.Time // UTC end time (zero if not finished)
    ExitCode      int       // Exit code (-1 if not finished)
    ErrorSummary  string    // Optional error summary (omitempty)
    AgentVersion  string    // Detected agent CLI version (omitempty)
}
```

### Dependencies

- `gopkg.in/yaml.v3` - YAML serialization
- `github.com/pkg/errors` - Error wrapping
- Standard library: `os`, `path/filepath`, `sync`, `time`

### Testing Strategy

**Unit Tests:** `internal/storage/storage_test.go`

- Test run creation
- Test run status updates
- Test run querying
- Test concurrent access (race detector)
- Test atomic writes
- Test error conditions (invalid paths, missing files)
- Mock time and PID for deterministic RunIDs

**Integration Tests:**
- Test with real filesystem
- Test concurrent writers
- Test run index correctness

**Key Test Cases:**
```go
- CreateRun with valid inputs
- CreateRun with empty projectID/taskID
- UpdateRunStatus on existing run
- GetRunInfo by ID
- ListRuns for project/task
- Concurrent CreateRun operations (race test)
```

### Performance Characteristics

- **CreateRun:** O(1) with one fsync (~5-10ms)
- **UpdateRunStatus:** O(1) with read + fsync (~5-10ms)
- **GetRunInfo:** O(1) with index lookup (~0.1-1ms)
- **ListRuns:** O(n) where n = number of runs (~1-10ms)

### Known Limitations

1. Single-node only (file-based storage)
2. No transactional updates across multiple runs
3. Run index is not persisted (rebuilt on restart)
4. Glob-based lookup can be slow with many projects

---

## 2. Configuration System

**Package:** `internal/config/`

### Purpose

The configuration system loads, validates, and resolves YAML configuration files. It handles:
- Agent backend configuration
- Token resolution (direct, file, environment)
- API server settings
- Storage paths
- Default values

### Key Files

- `config.go` - Main config loader and structure (Reference: line 1-130)
- `tokens.go` - Token resolution logic
- `storage.go` - Storage path resolution
- `api.go` - API configuration defaults
- `validation.go` - Configuration validation

### Configuration Structure

Reference: `internal/config/config.go:13-18`

```go
type Config struct {
    Agents   map[string]AgentConfig // Named agent configurations
    Defaults DefaultConfig          // Default agent & timeout
    API      APIConfig              // HTTP server settings
    Storage  StorageConfig          // Runs directory path
}
```

### Agent Configuration

Reference: `internal/config/config.go:21-29`

```go
type AgentConfig struct {
    Type      string // claude, codex, gemini, perplexity, xai
    Token     string // Direct token (discouraged in production)
    TokenFile string // Path to file containing token
    BaseURL   string // Custom endpoint (for Perplexity, xAI)
    Model     string // Model override

    tokenFromFile bool // Internal flag (not serialized)
}
```

### Default Configuration

```go
type DefaultConfig struct {
    Agent   string // Default agent name
    Timeout int    // Default timeout in seconds
}
```

### API Configuration

```go
type APIConfig struct {
    Host string // Bind host (e.g., "0.0.0.0")
    Port int    // Bind port (e.g., 8080)
    SSE  SSEConfig
}

type SSEConfig struct {
    PollIntervalMs      int // Polling interval in ms (default: 100)
    DiscoveryIntervalMs int // Discovery interval in ms (default: 1000)
    HeartbeatIntervalS  int // Heartbeat interval in seconds (default: 30)
    MaxClientsPerRun    int // Max concurrent clients per run (default: 10)
}
```

### Configuration Loading

Reference: `internal/config/config.go:43-83`

**LoadConfig (Full Validation):**
```go
1. Read YAML file
2. Parse YAML
3. Apply agent defaults (type normalization)
4. Apply API defaults (port, SSE settings)
5. Apply token environment overrides
6. Resolve token file paths (relative to config dir)
7. Resolve storage paths (expand ~, relative paths)
8. Validate configuration (check required fields)
9. Resolve tokens (read from files if needed)
10. Return Config
```

**LoadConfigForServer (Lenient):**

Reference: `internal/config/config.go:87-119`

Same as LoadConfig but skips token validation (steps 8-9).
Used for API server startup where agent execution may be disabled.

### Token Resolution

**Precedence Order:**
1. Direct `token` field in YAML
2. `token_file` field (read from file)
3. Environment variable: `AGENT_{TYPE}_TOKEN`
4. Error if not found

**Environment Variables:**
- `AGENT_CLAUDE_TOKEN`
- `AGENT_CODEX_TOKEN`
- `AGENT_GEMINI_TOKEN`
- `AGENT_PERPLEXITY_TOKEN`
- `AGENT_XAI_TOKEN`

### Path Resolution

**Token Files:**
- Relative paths resolved relative to config file directory
- `~` expanded to user home directory
- Example: `token_file: ~/.claude/token`

**Storage Paths:**
- Relative paths resolved relative to config file directory
- `~` expanded to user home directory
- Example: `runs_dir: ~/run-agent`

### Example Configuration

```yaml
agents:
  claude:
    type: claude
    token_file: ~/.claude/token
    model: claude-3-opus

  codex:
    type: codex
    token_file: ~/.openai/token
    base_url: https://api.openai.com

  gemini:
    type: gemini
    token: ${GEMINI_API_KEY}  # Not directly supported, use env var

defaults:
  agent: claude
  timeout: 3600

api:
  host: 0.0.0.0
  port: 8080
  sse:
    poll_interval_ms: 100
    discovery_interval_ms: 1000
    heartbeat_interval_s: 30
    max_clients_per_run: 10

storage:
  runs_dir: ~/run-agent
```

### Dependencies

- `gopkg.in/yaml.v3` - YAML parsing
- Standard library: `os`, `path/filepath`, `fmt`

### Testing Strategy

**Unit Tests:** `internal/config/config_test.go`

- Test YAML parsing
- Test token resolution (direct, file, env)
- Test path resolution (relative, absolute, ~)
- Test validation (missing required fields)
- Test defaults application
- Test environment variable overrides

**Key Test Cases:**
```go
- LoadConfig with valid config
- LoadConfig with missing agent token
- LoadConfig with token_file
- LoadConfig with environment variable
- LoadConfig with relative paths
- LoadConfigForServer (skip validation)
- Token resolution precedence
```

### Performance Characteristics

- **LoadConfig:** O(1) with file I/O (~1-5ms)
- **Token Resolution:** O(1) with potential file reads (~1-10ms)
- **Validation:** O(n) where n = number of agents (~<1ms)

### Known Limitations

1. No hot-reloading (restart required for config changes)
2. No config merging (single file only)
3. No schema versioning
4. Token validation requires actual API calls (not done in config load)

---

## 3. Message Bus

**Package:** `internal/messagebus/`

### Purpose

The message bus provides an append-only event log for task coordination and monitoring. It enables:
- Multi-agent communication
- Event logging and auditing
- Real-time event streaming
- Parent-child message relationships

### Key Files

- `messagebus.go` - Core message bus implementation (Reference: line 1-367)
- `msgid.go` - Message ID generation
- `lock_unix.go` - Unix/Linux/macOS file locking
- `lock_windows.go` - Windows file locking
- `lock.go` - Lock interface

### Architecture

**Design Principles:**
1. **Append-Only:** Messages are never modified or deleted
2. **O_APPEND:** Kernel-level atomic appends on Unix
3. **Exclusive Writes:** Only one writer at a time (via flock)
4. **Lockless Reads:** Readers don't block writers
5. **Durability:** fsync after each write
6. **Ordering:** Total order by timestamp

### Message Structure

Reference: `internal/messagebus/messagebus.go:28-38`

```go
type Message struct {
    MsgID        string    // Unique message ID
    Timestamp    time.Time // UTC timestamp
    Type         string    // Event type (e.g., "agent_started")
    ProjectID    string    // Project identifier
    TaskID       string    // Task identifier (optional)
    RunID        string    // Run identifier (optional)
    ParentMsgIDs []string  // Parent message IDs (for threading)
    Attachment   string    // Path to attached file (optional)
    Body         string    // Message content (not serialized in YAML)
}
```

### Message ID Format

**Structure:** `MSG-{YYYYMMDD-HHMMSS}-{NANOSECONDS}-PID{PID}-{SEQUENCE}`

**Example:** `MSG-20060102-150405-000000001-PID00123-0042`

**Properties:**
- Lexically sortable (for range queries)
- Globally unique (timestamp + PID + sequence)
- Human-readable timestamp component

**Generation:** See `internal/messagebus/msgid.go`

### MessageBus Interface

Reference: `internal/messagebus/messagebus.go:40-46`

```go
type MessageBus struct {
    path         string              // Path to messagebus.yaml file
    now          func() time.Time    // Clock (injectable)
    lockTimeout  time.Duration       // Write lock timeout
    pollInterval time.Duration       // Poll interval
}
```

### Core Operations

#### AppendMessage

Reference: `internal/messagebus/messagebus.go:102-152`

**Flow:**
```go
1. Validate message (type, projectID required)
2. Generate unique MsgID
3. Set timestamp (UTC)
4. Serialize to YAML
5. Validate bus path (no symlinks)
6. Open file: O_WRONLY | O_APPEND | O_CREATE
7. LockExclusive (flock with timeout)
8. Append message data
9. fsync() - Force disk write
10. Unlock
11. Close file
12. Return MsgID
```

**Concurrency:** Exclusive lock ensures only one writer at a time.

#### ReadMessages

Reference: `internal/messagebus/messagebus.go:155-174`

**Flow:**
```go
1. Validate bus path
2. ReadFile (lockless - no blocking!)
3. Parse YAML documents
4. Filter messages after sinceID
5. Return messages
```

**Concurrency:** Lockless - readers don't block writers or other readers.

#### PollForNew

Reference: `internal/messagebus/messagebus.go:177-191`

**Flow:**
```go
1. Loop:
   a. Call ReadMessages(lastID)
   b. If new messages found: return them
   c. If no messages: sleep(pollInterval)
   d. Continue loop
```

**Use Case:** Server-Sent Events (SSE) streaming.

### File Format

**YAML with Document Separators:**

Reference: `internal/messagebus/messagebus.go:229-251`

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
parents:
  - MSG-20060102-150405-000000001-PID00123-0001
---
Processing request...
```

### Message Parsing

Reference: `internal/messagebus/messagebus.go:253-321`

**State Machine:**
```
stateSeekHeader → stateHeader → stateBody
                     ↑_______________|
```

1. **stateSeekHeader:** Looking for `---` separator
2. **stateHeader:** Reading YAML header until next `---`
3. **stateBody:** Reading body until next `---` or EOF

### Locking Implementation

**Unix/Linux/macOS:** `internal/messagebus/lock_unix.go`

```go
func LockExclusive(file *os.File, timeout time.Duration) error {
    // Uses syscall.Flock with LOCK_EX | LOCK_NB
    // Retries with exponential backoff until timeout
}

func Unlock(file *os.File) error {
    // Uses syscall.Flock with LOCK_UN
}
```

**Windows:** `internal/messagebus/lock_windows.go`

```go
// Uses LockFileEx with LOCKFILE_EXCLUSIVE_LOCK
// Note: Windows mandatory locks may block readers (limitation)
```

### Dependencies

- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/pkg/errors` - Error wrapping
- Standard library: `os`, `syscall`, `time`, `bufio`

### Testing Strategy

**Unit Tests:** `internal/messagebus/messagebus_test.go`

- Test message appending
- Test message reading
- Test parsing (various formats)
- Test concurrent writers (race detector)
- Test concurrent readers
- Test PollForNew behavior
- Test error conditions

**Integration Tests:**
- Test with real filesystem
- Test concurrent operations
- Test large message volumes

**Key Test Cases:**
```go
- AppendMessage with valid message
- AppendMessage with empty type/projectID
- ReadMessages with no messages
- ReadMessages with sinceID
- PollForNew blocks until new messages
- Concurrent AppendMessage operations (race test)
- Concurrent readers (race test)
- Message parsing edge cases (empty body, special characters)
```

### Performance Characteristics

- **AppendMessage:** ~1-5ms (with fsync)
- **ReadMessages:** ~0.1-1ms (lockless)
- **PollForNew:** Blocks until new messages (~100-200ms latency)
- **Throughput:** ~200-1000 writes/sec (single file)

**Bottlenecks:**
- fsync() on write (durability vs. performance trade-off)
- Lock contention with 50+ concurrent writers
- File size growth (mitigation: log rotation)

### Known Limitations

1. **Windows:** Mandatory locks may block readers
2. **Network Filesystems:** O_APPEND may not be atomic (use local storage only)
3. **Scaling:** Single file limits throughput (~1000 writes/sec)
4. **File Size:** Use `WithAutoRotate(maxBytes)` option or `run-agent gc --rotate-bus` to manage bus file growth

---

## 4. Agent Protocol

**Package:** `internal/agent/`

### Purpose

The agent protocol defines a common interface for all agent backends. It abstracts:
- Agent execution
- Runtime context
- Stdio handling
- Error handling
- Process lifecycle

### Key Files

- `agent.go` - Agent interface definition (Reference: line 1-26)
- `executor.go` - Common execution logic
- `stdio.go` - Stdio capture utilities
- Backend implementations: `claude/`, `codex/`, `gemini/`, `perplexity/`, `xai/`

### Agent Interface

Reference: `internal/agent/agent.go:7-13`

```go
type Agent interface {
    // Execute runs the agent with the given context.
    Execute(ctx context.Context, runCtx *RunContext) error

    // Type returns the agent type (claude, codex, etc.).
    Type() string
}
```

### RunContext

Reference: `internal/agent/agent.go:16-25`

```go
type RunContext struct {
    RunID       string            // Unique run identifier
    ProjectID   string            // Project identifier
    TaskID      string            // Task identifier
    Prompt      string            // User prompt
    WorkingDir  string            // Working directory for agent
    StdoutPath  string            // Path to capture stdout
    StderrPath  string            // Path to capture stderr
    Environment map[string]string // Environment variables
}
```

**Key Fields:**
- `Prompt`: User's input/request to the agent
- `WorkingDir`: Where agent should execute (usually project root)
- `StdoutPath`: File to capture agent's stdout
- `StderrPath`: File to capture agent's stderr
- `Environment`: Key-value map for environment variables (e.g., API tokens)

### Execution Contract

**Requirements for Agent Implementations:**

1. **Execute Method:**
   - Accept `context.Context` for cancellation
   - Accept `RunContext` with all necessary data
   - Return error on failure (nil on success)

2. **Stdout/Stderr:**
   - Redirect stdout to `RunContext.StdoutPath`
   - Redirect stderr to `RunContext.StderrPath`
   - Use buffered I/O for performance

3. **Working Directory:**
   - Change to `RunContext.WorkingDir` before execution
   - Restore original directory on exit (if needed)

4. **Environment Variables:**
   - Merge `RunContext.Environment` with os.Environ()
   - Ensure API tokens are set correctly

5. **Exit Codes:**
   - 0: Success
   - Non-zero: Failure
   - Return error with exit code information

6. **Cancellation:**
   - Respect `context.Context` cancellation
   - Clean up resources on cancellation
   - Return `context.Canceled` error

### Common Execution Patterns

**Process Spawning:**
```go
cmd := exec.CommandContext(ctx, "agent-cli", args...)
cmd.Dir = runCtx.WorkingDir
cmd.Env = mergeEnvironment(os.Environ(), runCtx.Environment)

stdoutFile, _ := os.Create(runCtx.StdoutPath)
defer stdoutFile.Close()
cmd.Stdout = stdoutFile

stderrFile, _ := os.Create(runCtx.StderrPath)
defer stderrFile.Close()
cmd.Stderr = stderrFile

if err := cmd.Run(); err != nil {
    // Extract exit code and return error
    return fmt.Errorf("agent execution failed: %w", err)
}
return nil
```

**API Call Pattern:**
```go
client := NewAPIClient(runCtx.Environment["API_KEY"])
response, err := client.Call(ctx, runCtx.Prompt)
if err != nil {
    return fmt.Errorf("api call failed: %w", err)
}

// Write response to stdout file
stdoutFile, _ := os.Create(runCtx.StdoutPath)
defer stdoutFile.Close()
fmt.Fprintf(stdoutFile, "%s\n", response.Text)

return nil
```

### Error Handling

**Error Types:**
1. **Validation Errors:** Missing required fields in RunContext
2. **Execution Errors:** Agent CLI failed or API call failed
3. **I/O Errors:** Cannot write to stdout/stderr files
4. **Cancellation:** Context canceled by user

**Error Wrapping:**
```go
if err := agent.Execute(ctx, runCtx); err != nil {
    return errors.Wrap(err, "execute agent")
}
```

### Dependencies

- `context` - Context for cancellation
- Standard library: `os`, `os/exec`, `fmt`

### Testing Strategy

**Unit Tests:**
- Test Execute with valid RunContext
- Test Execute with missing fields
- Test Execute with cancellation
- Test Execute with I/O errors
- Mock external CLI/API calls

**Integration Tests:**
- Test with real agent CLI (if available)
- Test stdout/stderr capture
- Test environment variable passing
- Test working directory changes

**Key Test Cases:**
```go
- Execute with valid context
- Execute with empty prompt
- Execute with invalid API token
- Execute with canceled context
- Execute captures stdout correctly
- Execute captures stderr correctly
- Execute returns correct exit code
```

### Performance Characteristics

- **Execute Latency:** Depends on agent backend (1-60 seconds)
- **Overhead:** ~10-100ms for process spawning
- **Memory:** Depends on agent output size

### Known Limitations

1. No streaming output during execution (buffered to files)
2. No progress callbacks
3. No timeout enforcement at agent level (handled by caller)

---

## 5. Agent Backends

**Packages:** `internal/agent/claude/`, `codex/`, `gemini/`, `perplexity/`, `xai/`

### Purpose

Agent backends implement the `Agent` interface for specific AI providers. Each backend handles provider-specific details like:
- API authentication
- Request formatting
- Response parsing
- Error handling
- CLI invocation

### Supported Backends

1. **Claude** (`internal/agent/claude/`)
2. **Codex** (`internal/agent/codex/`) - OpenAI-compatible
3. **Gemini** (`internal/agent/gemini/`) - Google's Gemini API
4. **Perplexity** (`internal/agent/perplexity/`)
5. **xAI** (`internal/agent/xai/`) - Grok

### Backend Implementation Pattern

Each backend follows this structure:

```go
package agentname

import (
    "context"
    "github.com/jonnyzzz/conductor-loop/internal/agent"
)

type Agent struct {
    config  Config  // Backend-specific configuration
    apiKey  string  // API authentication token
    baseURL string  // Optional custom endpoint
    model   string  // Optional model override
}

func New(config Config) (*Agent, error) {
    // Validate configuration
    // Initialize agent
    return &Agent{...}, nil
}

func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
    // 1. Prepare request (prompt, model, parameters)
    // 2. Call CLI or API
    // 3. Capture stdout/stderr
    // 4. Handle errors
    // 5. Return nil on success
}

func (a *Agent) Type() string {
    return "agentname"
}
```

### 1. Claude Backend

**Package:** `internal/agent/claude/`

**Implementation:**
- Uses Claude CLI directly (`claude` command)
- Spawns subprocess with `exec.CommandContext`
- Redirects stdout/stderr to files
- Passes API key via `ANTHROPIC_API_KEY` environment variable

**Configuration:**
```yaml
claude:
  type: claude
  token_file: ~/.claude/token
  model: claude-3-opus  # Optional
```

**Environment Variables:**
- `ANTHROPIC_API_KEY` - API authentication token

**Example Execution:**
```bash
ANTHROPIC_API_KEY=sk-... claude --prompt "Your prompt here"
```

### 2. Codex Backend (OpenAI)

**Package:** `internal/agent/codex/`

**Implementation:**
- OpenAI API-compatible
- Can use OpenAI CLI or direct API calls
- Supports custom base URLs (for proxies)

**Configuration:**
```yaml
codex:
  type: codex
  token_file: ~/.openai/token
  base_url: https://api.openai.com  # Optional
  model: gpt-4  # Optional
```

**Environment Variables:**
- `OPENAI_API_KEY` - API authentication token

### 3. Gemini Backend

**Package:** `internal/agent/gemini/`

**Implementation:**
- Uses Google's Gemini API
- Direct API calls (HTTP client)
- Requires Google API key

**Configuration:**
```yaml
gemini:
  type: gemini
  token_file: ~/.gemini/token
  model: gemini-pro  # Optional
```

**Environment Variables:**
- `GEMINI_API_KEY` - API authentication token

### 4. Perplexity Backend

**Package:** `internal/agent/perplexity/`

**Implementation:**
- Custom API endpoint support
- Direct API calls

**Configuration:**
```yaml
perplexity:
  type: perplexity
  token_file: ~/.perplexity/token
  base_url: https://api.perplexity.ai
```

**Environment Variables:**
- `PERPLEXITY_API_KEY` - API authentication token

### 5. xAI Backend (Grok)

**Package:** `internal/agent/xai/`

**Implementation:**
- xAI (Grok) API
- Custom endpoint support
- Direct API calls

**Configuration:**
```yaml
xai:
  type: xai
  token_file: ~/.xai/token
  base_url: https://api.x.ai
  model: grok-1  # Optional
```

**Environment Variables:**
- `XAI_API_KEY` - API authentication token

### Adding New Backends

See: [Adding New Agent Backends](adding-agents.md)

**Steps:**
1. Create new package: `internal/agent/newagent/`
2. Implement `Agent` interface
3. Add configuration schema
4. Update agent factory
5. Add tests
6. Document configuration

### Dependencies

**Common:**
- `context` - Context for cancellation
- `os/exec` - Process spawning
- Standard library: `os`, `fmt`, `io`

**Backend-Specific:**
- HTTP client for API-based backends
- CLI tools for CLI-based backends

### Testing Strategy

**Unit Tests:**
- Test agent initialization
- Test Execute with valid config
- Test Execute with missing API key
- Test Execute with cancellation
- Mock external API calls/CLI execution

**Integration Tests:**
- Test with real API keys (if available)
- Test stdout/stderr capture
- Test error handling

**Key Test Cases:**
```go
- New agent with valid config
- New agent with invalid config
- Execute with valid prompt
- Execute with empty prompt
- Execute with invalid API key
- Execute captures output correctly
- Execute handles API errors
```

### Performance Characteristics

**Execution Times (Typical):**
- Claude: 5-30 seconds
- Codex: 2-10 seconds
- Gemini: 3-15 seconds
- Perplexity: 2-10 seconds
- xAI: 3-15 seconds

**Factors:**
- Prompt complexity
- Model size
- Network latency
- API rate limits

### Known Limitations

1. No streaming output (buffered to files)
2. No token usage tracking
3. No cost estimation
4. Rate limits enforced by provider (not by conductor-loop)

---

## 6. Runner Orchestration

**Package:** `internal/runner/`

### Purpose

The runner orchestration subsystem coordinates agent execution with automatic restart capabilities. It handles:
- Task execution workflow
- Ralph loop (restart manager)
- Process spawning and monitoring
- Process group management
- Graceful shutdown

### Key Files

- `orchestrator.go` - Main orchestration logic (Reference: line 1-250)
- `ralph.go` - Ralph loop implementation
- `process.go` - Process spawning
- `task.go` - Task execution
- `job.go` - Job management
- Platform-specific: `wait_*.go`, `pgid_*.go`, `stop_*.go`

### Orchestrator

**Responsibilities:**
- Load configuration
- Select agent backend
- Create run metadata
- Execute Ralph loop
- Update run status
- Log events to message bus

**Structure:**
```go
type Orchestrator struct {
    rootDir    string
    cfg        *config.Config
    storage    storage.Storage
    messagebus *messagebus.MessageBus
}
```

### Ralph Loop

See detailed specification: [Ralph Loop Specification](ralph-loop.md)

**Purpose:** Manage agent process lifecycle with automatic restarts

**Configuration:**
```go
type RalphConfig struct {
    MaxRestarts  int           // Maximum restart attempts (default: 100)
    WaitTimeout  time.Duration // Timeout for wait (default: 5 minutes)
    PollInterval time.Duration // DONE file check interval (default: 1 second)
    RestartDelay time.Duration // Delay before restart (default: 1 second)
}
```

**Algorithm:**
```
1. Initialize restart counter = 0
2. Loop:
   a. Spawn agent process
   b. Wait for completion
   c. Check exit conditions:
      - Exit code 0 → STOP (success)
      - DONE file exists → STOP
      - Max restarts exceeded → STOP
      - Wait-without-restart signal → STOP
      - Fatal error → STOP
      - Otherwise → RESTART
   d. If restart: increment counter, delay, continue
3. Cleanup: Kill process group if needed
4. Update run status
```

### Process Management

**Process Group (PGID):**
- All child processes belong to same process group
- Allows killing entire process tree
- Platform-specific implementations

**Unix/Linux/macOS:**
```go
// Set process group ID = PID
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,
    Pgid:    0,  // Use PID as PGID
}

// Kill entire process group
syscall.Kill(-pgid, syscall.SIGTERM)
```

**Windows:**
```go
// Limited support (no process groups)
// Use job objects or direct process kill
// Recommendation: Use WSL2
```

### Process Spawning

**Steps:**
1. Create command: `exec.CommandContext(ctx, agentCLI, args...)`
2. Set working directory: `cmd.Dir = workingDir`
3. Set environment: `cmd.Env = environment`
4. Set process group: `cmd.SysProcAttr = ...`
5. Redirect stdout/stderr to files
6. Start process: `cmd.Start()`
7. Return PID and PGID

### Exit Conditions

**Stop (No Restart):**
1. **Success:** Exit code 0
2. **DONE File:** `TASK_FOLDER/DONE` file exists
3. **Max Restarts:** Exceeded `maxRestarts` limit
4. **Wait-Without-Restart:** Special signal received
5. **Fatal Error:** Unrecoverable error

**Restart:**
1. Non-zero exit code (within restart limit)
2. No DONE file present
3. No fatal errors

### DONE File Detection

**Purpose:** Signal task completion from within agent

**Location:** `{task_dir}/DONE`

**Usage:**
```bash
# Inside agent script
echo "Task completed" > $TASK_FOLDER/DONE
```

**Ralph Loop Behavior:**
- Checks for DONE file after each agent exit
- If found: Stop loop (no restart)
- If not found: Continue restart logic

### Dependencies

- `github.com/jonnyzzz/conductor-loop/internal/config`
- `github.com/jonnyzzz/conductor-loop/internal/storage`
- `github.com/jonnyzzz/conductor-loop/internal/messagebus`
- `github.com/jonnyzzz/conductor-loop/internal/agent`
- `github.com/pkg/errors`
- Standard library: `os`, `os/exec`, `syscall`, `time`

### Testing Strategy

**Unit Tests:** `internal/runner/orchestrator_test.go`, `ralph_test.go`, `process_test.go`

- Test orchestrator initialization
- Test task execution flow
- Test Ralph loop restart logic
- Test process spawning
- Test DONE file detection
- Test exit code handling
- Test cancellation

**Integration Tests:**
- Test with real agent execution
- Test concurrent tasks
- Test process cleanup

**Key Test Cases:**
```go
- Execute task with success exit
- Execute task with failure exit
- Execute task with DONE file
- Execute task with max restarts
- Execute task with cancellation
- Process group cleanup
- Concurrent task execution
```

### Performance Characteristics

- **Task Startup:** ~10-100ms (process spawning overhead)
- **Restart Delay:** Configurable (default: 1 second)
- **DONE File Check:** Every 1 second (configurable)

### Known Limitations

1. **Windows:** Limited process group support
2. **Orphan Processes:** Possible if process group management fails
3. **No Distributed Execution:** Single-node only
4. **No Priority Queues:** FIFO execution order

---

## 7. API Server

**Package:** `internal/api/`

### Purpose

The API server provides a REST API and Server-Sent Events (SSE) for real-time monitoring and control of tasks, runs, and the message bus.

### Key Files

- `server.go` - HTTP server setup
- `handlers.go` - Request handlers (21KB, ~40+ endpoints)
- `handlers_projects.go` - Project/task/run API handlers; includes `findProjectDir` and `findProjectTaskDir` path-resolution helpers
- `sse.go` - Server-Sent Events streaming
- `routes.go` - Route definitions
- `middleware.go` - HTTP middleware
- `discovery.go` - Project/task discovery
- `tailer.go` - Log file tailing

### Server Configuration

```go
type Options struct {
    RootDir          string           // Storage root directory
    ConfigPath       string           // Config file path
    APIConfig        config.APIConfig // API server config
    Version          string           // Version string
    Logger           *log.Logger      // Logger instance
    DisableTaskStart bool             // Disable POST /tasks
    Now              func() time.Time // Clock (injectable)
}
```

### REST Endpoints

#### Projects

- `GET /api/projects` - List all projects
- `GET /api/projects/{projectID}` - Get project details

#### Tasks

- `GET /api/projects/{projectID}/tasks` - List tasks in project (includes `run_counts` map per task)
- `GET /api/projects/{projectID}/tasks/{taskID}` - Get task details
- `POST /api/projects/{projectID}/tasks` - Start new task
- `GET /api/projects/{projectID}/tasks/{taskID}/file?name=...` - Read task file

**Start Task Request:**
```json
{
  "task_id": "task-001",
  "prompt": "Your task description",
  "project_root": "/path/to/project",
  "attach_mode": "restart"  // or "new"
}
```

#### Runs

- `GET /api/projects/{projectID}/tasks/{taskID}/runs/{runID}` - Get run info
- `GET /api/projects/{projectID}/tasks/{taskID}/runs/{runID}/file?name=...&tail=5000` - Read run file (with optional tail)
- `POST /api/projects/{projectID}/tasks/{taskID}/runs/{runID}/stop` - Stop running task

#### Message Bus

- `GET /api/projects/{projectID}/bus` - List project messages
- `GET /api/projects/{projectID}/tasks/{taskID}/bus` - List task messages
- `POST /api/projects/{projectID}/bus` - Post project message
- `POST /api/projects/{projectID}/tasks/{taskID}/bus` - Post task message

**Post Message Request:**
```json
{
  "type": "user_message",
  "body": "Message content"
}
```

#### Streaming (SSE)

- `GET /api/projects/{projectID}/tasks/{taskID}/logs/stream` - Stream task logs
- `GET /api/projects/{projectID}/bus/stream` - Stream project messages
- `GET /api/projects/{projectID}/tasks/{taskID}/bus/stream` - Stream task messages
- `GET /api/runs/stream` - Stream all runs (discovery)

### Server-Sent Events (SSE)

**SSE Configuration:**
```go
type SSEConfig struct {
    PollIntervalMs      int // Poll interval in ms (default: 100)
    DiscoveryIntervalMs int // Discovery interval in ms (default: 1000)
    HeartbeatIntervalS  int // Heartbeat interval in seconds (default: 30)
    MaxClientsPerRun    int // Max concurrent clients per run (default: 10)
}
```

**SSE Event Format:**
```
event: message
data: {"msg_id": "...", "type": "...", "body": "..."}

event: heartbeat
data: {}

event: run_discovered
data: {"run_id": "...", "project_id": "...", "task_id": "..."}
```

**SSE Flow:**
```
1. Client: EventSource("/api/projects/{id}/bus/stream")
2. Server: StreamManager.Subscribe()
3. Server: Poll Loop (every 100ms)
   - messagebus.PollForNew()
   - Detect new messages
4. Server: Send SSE events
   - "data: {json}\n\n"
5. Server: Send heartbeat (every 30s)
6. Client: Receive events
7. Client: Update UI
```

### Middleware

**CORS:**
```go
// Allow all origins for development
// Restrict in production
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

**Logging:**
```go
// Log all requests
[2026-02-05 10:00:00] GET /api/projects 200 5ms
[2026-02-05 10:00:01] POST /api/projects/my-project/tasks 201 150ms
```

**Error Handling:**
```go
// Standardized error responses
{
  "error": "error message",
  "details": "optional details"
}
```

### File Serving

**Task Files:**
- `TASK_STATE.md` - Task metadata
- `DONE` - Completion marker
- Custom files created by agents

**Run Files:**
- `run-info.yaml` - Run metadata
- `stdout` - Agent stdout
- `stderr` - Agent stderr
- `output.md` - Final output

**Tail Support:**
```
GET /api/.../file?name=stdout&tail=5000
- Returns last 5000 bytes of file
- Useful for large log files
```

### Discovery

**Project Discovery:**
- Scans `{rootDir}/*` for project directories
- Filters out hidden directories (`.`, `..`)

**Task Discovery:**
- Scans `{rootDir}/{projectID}/*` for task directories
- Checks for `TASK_STATE.md` file

**Run Discovery:**
- Scans `{rootDir}/{projectID}/{taskID}/runs/*` for run directories
- Checks for `run-info.yaml` file

### Dependencies

- `github.com/jonnyzzz/conductor-loop/internal/storage`
- `github.com/jonnyzzz/conductor-loop/internal/messagebus`
- `github.com/jonnyzzz/conductor-loop/internal/runner`
- Standard library: `net/http`, `encoding/json`, `log`

### Testing Strategy

**Unit Tests:** `internal/api/handlers_test.go`, `sse_test.go`, `server_test.go`

- Test route handlers
- Test SSE streaming
- Test middleware
- Test error handling
- Test file serving
- Mock storage and message bus

**Integration Tests:**
- Test with real server
- Test concurrent clients
- Test SSE connection management

**Key Test Cases:**
```go
- GET /api/projects returns projects
- POST /api/projects/{id}/tasks starts task
- GET /api/.../runs/{id} returns run info
- SSE stream sends events
- SSE heartbeat works
- Concurrent SSE clients
- File tail works correctly
```

### Performance Characteristics

- **Request Latency:** ~1-5ms (simple endpoints)
- **SSE Latency:** ~100ms (poll interval)
- **Concurrent Clients:** Limited per run (default: 10)
- **Throughput:** Depends on storage and message bus

### Path Resolution Helpers

**Reference:** `internal/api/handlers_projects.go`

Two helper functions resolve project and task directories for API handlers, supporting multiple common directory layouts:

```go
// findProjectDir locates <projectID> under rootDir.
// Checks: <rootDir>/<projectID>, <rootDir>/runs/<projectID>
func findProjectDir(rootDir, projectID string) (string, bool)

// findProjectTaskDir locates <projectID>/<taskID> under rootDir.
// Checks: direct, runs/ subdirectory, and walks up to 3 levels deep.
func findProjectTaskDir(rootDir, projectID, taskID string) (string, bool)
```

These helpers are used by handlers for task files, project stats, message bus, and task creation so that the API works correctly regardless of whether the root is set to the repo root or the `runs/` subdirectory.

### Known Limitations

1. **No Authentication:** Open API (secure via network)
2. **No Rate Limiting:** Clients can overwhelm server
3. **No Pagination:** Large lists may be slow
4. **No WebSocket:** SSE only (unidirectional)
5. **No Compression:** Large responses not compressed

---

## 8. Frontend UI

**Package:** `web/src/`

### Purpose

The frontend provides a web-based dashboard for monitoring and controlling tasks, runs, and the message bus in real-time.

### Technology Stack

- **Plain HTML/CSS/JS** — vanilla JavaScript, no framework
- **No npm, no build step, no TypeScript**
- Static files served directly by the API server at `/ui/`

### Key Files

- `web/src/index.html` - Main single-page application entry point
- `web/src/app.js` - Application logic (vanilla JS)
- `web/src/styles.css` - Styles

### Key UI Features

- **Left panel:** Project message bus live stream (SSE)
- **Main panel:** Task list with expandable run details
- **Tabs per run:** TASK.MD, OUTPUT, STDOUT, STDERR, RUN-INFO, MESSAGES
- **Real-time streaming:** File content via SSE (`/api/.../stream`)
- **Stop button:** Stop running tasks via POST `.../stop`
- **Message posting form:** Post messages to project or task bus
- **Auto-refresh:** Task list refreshes every 5 seconds

### Deployment

The UI is a static single-page app — no build step required. The API server
serves all files under `web/src/` at the `/ui/` path.

```
# Open in browser after starting conductor:
http://localhost:8080/ui/
```

### Performance Characteristics

- **Initial Load:** Instant (small static files, no bundle)
- **SSE Latency:** ~100ms (poll interval)

### Known Limitations

1. **No Offline Support:** Requires active server connection
2. **No Mobile Optimization:** Desktop-focused UI

---

## 9. Webhook Notifications

**Package:** `internal/webhook/`

### Purpose

The webhook package delivers run completion notifications to external HTTP endpoints. It enables:
- Integration with CI/CD pipelines and external services
- HMAC-SHA256 signed payloads for authenticity verification
- Async delivery with automatic retry on failure

### Key Files

- `webhook.go` - Notifier implementation and HTTP delivery
- `config.go` - Webhook configuration types
- `webhook_test.go` - Unit tests (11 tests)

### Configuration

```yaml
webhook:
  url: https://your-endpoint.example.com/hook
  events:
    - run_completed
  secret: your-hmac-secret
  timeout: 10s
```

### Signature Header

Every POST includes an HMAC-SHA256 signature:
```
X-Conductor-Signature: sha256=<hex>
```

### Delivery

- **Trigger:** Fires async POST after each run finalization (in `internal/runner/job.go`)
- **Retry:** 3 attempts with exponential backoff
- **Events:** Configurable; currently supports `run_completed`

### Dependencies

- Standard library: `net/http`, `crypto/hmac`, `crypto/sha256`

### Testing Strategy

**Unit Tests:** `internal/webhook/webhook_test.go` (11 tests)

- Test HMAC signature generation
- Test HTTP delivery success/failure
- Test retry logic
- Test configuration validation

---

---

## 10. CLI Commands: run-agent list

**File:** `cmd/run-agent/list.go`

### Purpose

The `run-agent list` command reads run metadata directly from the filesystem (no server required) and outputs it in human-readable table form or JSON.

### Usage

```
run-agent list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--root` | Root directory (default: `./runs` or `$RUNS_DIR`) |
| `--project` | Project ID; if set, lists tasks for that project |
| `--task` | Task ID; requires `--project`; if set, lists runs for that task |
| `--json` | Emit JSON instead of table output |

### Output Modes

**No flags (project list):** Prints one project name per line, sorted.

**`--project` only (task list):** Tab-separated table with columns:
`TASK_ID`, `RUNS`, `LATEST_STATUS`, `DONE`

**`--project --task` (run list):** Tab-separated table with columns:
`RUN_ID`, `STATUS`, `EXIT_CODE`, `STARTED`, `DURATION`

### Implementation Notes

- Root directory falls back to `RUNS_DIR` environment variable, then `./runs`.
- DONE detection: `os.Stat(taskDir/DONE)`.
- Latest status: reads `run-info.yaml` from the lexically last run directory.
- Duration shown as `"running"` for active runs; computed from `end_time - start_time` otherwise.
- JSON mode wraps output in `{"projects": [...]}`, `{"tasks": [...]}`, or `{"runs": [...]}`.

### Dependencies

- `internal/storage` (ReadRunInfo)
- `text/tabwriter` for aligned table output
- `encoding/json` for JSON output

### Testing Strategy

**Unit Tests:** `cmd/run-agent/list_test.go` (13 tests)

- Test listing projects (empty root, multiple projects)
- Test listing tasks (DONE detection, run count, latest status)
- Test listing runs (running vs. completed, duration formatting)
- Test JSON output for each mode
- Test `--task requires --project` validation

---

## 11. CLI Commands: run-agent output

**File:** `cmd/run-agent/output.go`

### Purpose

The `run-agent output` command prints output files from completed (or running) agent runs. With `--follow` / `-f`, it live-tails the output of a running job.

### Usage

```
run-agent output [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--run-dir` | Direct path to run directory (overrides `--project/--task/--run`) |
| `--root` | Root directory (default: `./runs` or `$RUNS_DIR`) |
| `--project` | Project ID |
| `--task` | Task ID |
| `--run` | Run ID (uses most recent if omitted) |
| `--tail N` | Print last N lines only (0 = all) |
| `--file` | File to print: `output` (default), `stdout`, `stderr`, `prompt` |
| `--follow`, `-f` | Follow output as it is written (for running jobs) |

### File Resolution

**`--file output` (default):** Tries `output.md` first; falls back to `agent-stdout.txt`.

**`--file stdout`:** Uses `agent-stdout.txt`.

**`--file stderr`:** Uses `agent-stderr.txt`.

**`--file prompt`:** Uses `prompt.md`.

### Follow Mode (`--follow`)

1. If the run is already complete (non-`running` status in `run-info.yaml`), prints all content and exits immediately.
2. For a running job:
   - Resolves the live file (`agent-stdout.txt` for output/stdout, `agent-stderr.txt` for stderr).
   - Waits up to `followFileWaitTimeout` (5 seconds) for the file to appear.
   - Polls every `followPollInterval` (500 ms) for new bytes, printing them immediately.
   - Exits when `run-info.yaml` status becomes non-`running`, or when no new data has arrived for `followNoDataTimeout` (60 seconds).
   - Handles SIGINT/SIGTERM gracefully.

### Implementation Notes

- Poll intervals are package-level variables (`followPollInterval`, `followFileWaitTimeout`, `followNoDataTimeout`) so tests can shorten them.
- `drainNewContent` opens the file, seeks to `offset`, copies new bytes to stdout, and returns the byte count written.
- `--tail` is implemented with a ring-buffer over a line scanner (avoids loading the entire file into memory).
- Root directory falls back to `RUNS_DIR` environment variable, then `./runs`.

### Dependencies

- `internal/storage` (ReadRunInfo, StatusRunning)
- Standard library: `os/signal`, `syscall`, `bufio`, `io`

### Testing Strategy

**Unit Tests:** `cmd/run-agent/output_follow_test.go` (6 tests)

- Follow exits immediately for a completed run
- Follow prints live data from a running job
- Follow exits when job completes
- Follow exits on no-data timeout
- Follow exits on SIGINT
- Follow waits for file to appear

---

---

## 12. CLI Commands: run-agent watch

**File:** `cmd/run-agent/watch.go`

### Purpose

The `run-agent watch` command polls the filesystem until all specified tasks reach a terminal state (completed or failed). It is useful in automation scripts that submit multiple parallel tasks and need to block until they all finish.

### Usage

```
run-agent watch --project <id> --task <id> [--task <id> ...] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--project` | Project ID (required) |
| `--task` | Task ID to watch; can be repeated for multiple tasks |
| `--root` | Root directory (default: `./runs` or `$RUNS_DIR`) |
| `--timeout` | Maximum wait time (default: 30m); exits with code 1 on timeout |
| `--json` | Output as JSON lines (one line per poll cycle) |

### Implementation Notes

- Polls every `watchPollInterval` (2 seconds; package-level for testing).
- For each task, reads the latest run directory's `run-info.yaml` to determine status.
- Both `completed` and `failed` are considered "done" (terminal).
- Text mode prints a table of tasks with elapsed time on each poll cycle.
- JSON mode emits `{"tasks":[...],"all_done":bool}` per poll cycle.
- Root directory falls back to `RUNS_DIR` environment variable, then `./runs`.

### JSON Output Schema

```json
{
  "tasks": [
    {
      "task_id": "task-20260220-140000-hello",
      "status": "completed",
      "elapsed": 65.3,
      "done": true
    }
  ],
  "all_done": true
}
```

### Dependencies

- `internal/storage` (ReadRunInfo, StatusCompleted, StatusFailed, StatusRunning)
- Standard library: `encoding/json`, `os`, `path/filepath`, `sort`, `time`

### Testing Strategy

**Unit Tests:** `cmd/run-agent/watch_test.go` (6 tests)

- Error when no `--task` flags given
- Immediate exit when task already completed
- Waits for running task to complete
- Waits for multiple tasks, exits when all done
- Timeout error when task never completes
- JSON output format and schema

---

## 13. API: DELETE Run Endpoint

**File:** `internal/api/handlers_projects.go` (`handleRunDelete`)

### Purpose

`DELETE /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}` permanently removes a completed or failed run directory from disk. This allows users to reclaim disk space for individual runs without using `run-agent gc`.

### Behavior

1. Looks up the run using the shared path-resolution helpers (`findProjectTaskDir`).
2. If the run's status is `running`, returns **409 Conflict** (stop the run first).
3. Removes the run directory via `os.RemoveAll`.
4. Returns **204 No Content** on success.

### Response Codes

| Code | Meaning |
|------|---------|
| 204 | Run directory deleted successfully |
| 404 | Run or task directory not found |
| 409 | Run is still running |
| 500 | Filesystem error |

### Web UI Integration

The React frontend (`frontend/src/RunDetail.tsx`) shows a **Delete run** button for completed and failed runs. The button is hidden while a run is in `running` status to prevent accidental deletion of live runs.

---

## 14. UI: Task Search Bar

**File:** `frontend/src/` (React frontend)

### Purpose

A search bar in the task list panel lets users filter tasks by ID substring without reloading data from the server. Filtering is purely client-side and case-insensitive.

### Behavior

- Input appears at the top of the task list.
- On each keystroke, the task list is filtered to only show tasks whose ID contains the search string (case-insensitive substring match).
- A **"Showing N of M tasks"** label is displayed below the search bar when a filter is active (N < M).
- Clearing the input restores the full list.

---

## 15. API: Task Deletion Endpoint

**File:** `internal/api/handlers_projects.go` (`handleTaskDelete`)

### Purpose

`DELETE /api/projects/{project_id}/tasks/{task_id}` permanently removes an entire task directory — including all run subdirectories, agent output, the task message bus (`TASK-MESSAGE-BUS.md`), and `TASK.md` — from disk. This is the API-level complement to `run-agent task delete`.

### Behavior

1. Scans the in-memory run list for any run whose `(projectID, taskID)` matches and whose status is `running`.
2. If any running run is found, returns **409 Conflict** (stop all runs first or use the CLI's `--force` flag).
3. Locates the task directory using `findProjectTaskDir(rootDir, projectID, taskID)`.
4. If the task directory is not found, returns **404 Not Found**.
5. Removes the task directory tree via `os.RemoveAll`.
6. Returns **204 No Content** on success.

### Response Codes

| Code | Meaning |
|------|---------|
| 204 | Task directory deleted successfully |
| 404 | Task directory not found |
| 409 | At least one run is still in `running` status |
| 500 | Filesystem error |

### CLI Wrapper

`cmd/run-agent/task_delete.go` implements `run-agent task delete`:
- `--project`, `--task` (both required)
- `--root` (default: `$RUNS_DIR` or `./runs`)
- `--force` — skips the running-run check before deleting

Without `--force`, the CLI scans `<taskDir>/runs/*/run-info.yaml` for `status: running` and exits with code 1 if any are found.

### Tests

Tests for `handleTaskDelete` are in `internal/api/handlers_projects_test.go`:
- Delete existing task with no running runs → 204
- Delete task with a running run (no force) → 409
- Delete non-existent task → 404

---

## 16. UI: Project Stats Dashboard

**File:** `frontend/src/components/ProjectStats.tsx`

### Purpose

A stats bar displayed at the top of the task list in the React UI. It gives an at-a-glance view of how many tasks and runs exist in a project and their statuses, without requiring the user to scroll through the task list.

### Component

`ProjectStats` is a pure display component:
- Receives `projectId` as a prop.
- Fetches data via the `useProjectStats(projectId)` React Query hook, which calls `GET /api/projects/{projectId}/stats`.
- Renders a horizontal bar of labeled statistics.
- Shows a loading placeholder while data is fetching; shows "Stats unavailable" on error.

### Data Source

`GET /api/projects/{project_id}/stats` (see [Section 7](#7-api-server) and [API Reference](../user/api-reference.md)).

The response fields consumed by the component:

| Field | Displayed As |
|-------|-------------|
| `total_tasks` | Tasks |
| `total_runs` | Runs |
| `running_runs` | Running (only shown when > 0) |
| `completed_runs` | Done |
| `failed_runs + crashed_runs` | Failed (only shown when > 0) |
| `message_bus_total_bytes` | Bus (human-readable: B / KB / MB) |

### Refresh

Data is fetched by React Query with the project's default stale time (typically every 10 seconds on focus, or on explicit refetch triggered by the task list refresh cycle).

### Styling

CSS classes follow the `project-stats-*` namespace:
- `project-stats-bar` — flex container for the entire bar
- `project-stats-item` — label + value pair
- `project-stats-running` — value colored for running state
- `project-stats-completed` — value colored for completed state
- `project-stats-failed` — value colored for failed state

---

## Next Steps

For more specialized documentation, see:

- [Agent Protocol Specification](agent-protocol.md)
- [Ralph Loop Specification](ralph-loop.md)
- [Message Bus Protocol](message-bus.md)
- [Storage Layout Specification](storage-layout.md)
- [Adding New Agent Backends](adding-agents.md)

---

**Last Updated:** 2026-02-21 (Session #31)
**Version:** 1.0.0
