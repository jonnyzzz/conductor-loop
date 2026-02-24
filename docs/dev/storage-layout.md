# Storage Layout Specification

This document provides a comprehensive specification of the conductor-loop storage system, covering directory structure, file formats, atomic operations, locking mechanisms, and platform compatibility.

## Table of Contents

1. [Overview](#overview)
2. [Run Directory Structure](#run-directory-structure)
3. [RunInfo Schema](#runinfo-schema)
4. [Atomic Write Pattern](#atomic-write-pattern)
5. [Parent-Child Relationships](#parent-child-relationships)
6. [File Locking Mechanisms](#file-locking-mechanisms)
7. [RunID Generation Format](#runid-generation-format)
8. [Task State Files](#task-state-files)
9. [Cleanup and Retention](#cleanup-and-retention)
10. [Query Operations](#query-operations)
11. [Platform Compatibility](#platform-compatibility)
12. [Code Examples](#code-examples)

---

## Overview

The storage layer manages persistent run metadata using a file-based storage system. It provides:

- **Run Creation and Metadata Persistence:** Atomic writes to prevent corruption
- **Run Status Updates:** Safe concurrent access with read-copy-update pattern
- **Run Querying and Listing:** Fast lookups via in-memory indexing
- **Parent-Child Tracking:** Support for hierarchical run relationships

**Design Philosophy:**
- **Simplicity:** No external database dependencies
- **Debuggability:** Human-readable YAML format
- **Atomicity:** Temp + fsync + rename for crash safety
- **Performance:** In-memory indexing for fast lookups

**Package:** `internal/storage/`

**Key Files:**
- `storage.go` - Main storage interface and FileStorage implementation
- `atomic.go` - Atomic write operations (temp + fsync + rename)
- `runinfo.go` - Run metadata structure and constants

---

## Run Directory Structure

### Complete Directory Tree

```
{storage_root}/                           # Configured via --root, CONDUCTOR_ROOT env, or storage.runs_dir
├── config.yaml  (or config.hcl)          # Global configuration (YAML checked first)
│
├── {project_id}/                         # Project root directory
│   ├── PROJECT-MESSAGE-BUS.md            # Project-level message bus (append-only)
│   ├── home-folders.md                   # Project folder configuration
│   ├── FACT-{timestamp}-{name}.md        # Project-level facts
│   │
│   └── {task_id}/                        # Task directory (pattern: task-YYYYMMDD-HHMMSS-slug)
│       ├── TASK.md                       # Task prompt
│       ├── TASK_STATE.md                 # Current task state (updated by root agent)
│       ├── DONE                          # Completion marker (empty file)
│       ├── TASK-MESSAGE-BUS.md           # Task-level message bus (append-only)
│       ├── TASK-FACTS-{timestamp}.md     # Task-level facts
│       ├── ATTACH-{timestamp}-{name}.ext # Task attachments
│       │
│       └── runs/                         # All runs for this task
│           ├── {run_id_1}/               # Individual run directory
│           │   ├── run-info.yaml         # Run metadata (YAML)
│           │   ├── agent-stdout.txt      # Agent stdout
│           │   ├── agent-stderr.txt      # Agent stderr
│           │   ├── output.md             # Final output (structured)
│           │   └── prompt.md             # Prompt used for this run
│           │
│           ├── {run_id_2}/               # Second run (restart or child)
│           │   ├── run-info.yaml
│           │   ├── agent-stdout.txt
│           │   ├── agent-stderr.txt
│           │   └── ...
│           │
│           └── {run_id_N}/               # Nth run
│               └── ...
│
└── {project_id_2}/                       # Second project
    └── ...
```

### Example with Real Data

```
~/run-agent/
├── my-project/
│   ├── PROJECT-MESSAGE-BUS.md
│   ├── home-folders.md
│   │
│   └── task-20260205-103000-example/
│       ├── TASK.md
│       ├── TASK_STATE.md
│       ├── DONE
│       ├── TASK-MESSAGE-BUS.md
│       │
│       └── runs/
│           ├── 20260205-1030450000-12345-0/
│           │   ├── run-info.yaml
│           │   ├── agent-stdout.txt
│           │   ├── agent-stderr.txt
│           │   ├── prompt.md
│           │   └── output.md
│           │
│           └── 20260205-1031451234-12345-1/
│               ├── run-info.yaml
│               ├── agent-stdout.txt
│               ├── agent-stderr.txt
│               └── output.md
│
└── another-project/
    └── task-20260205-104000-fix/
        └── runs/
            └── 20260205-1045127890-67890-0/
                └── ...
```

### Path Construction

**Project Directory:**
```
{storage_root}/{project_id}/
```

**Task Directory:**
```
{storage_root}/{project_id}/{task_id}/
```

**Run Directory:**
```
{storage_root}/{project_id}/{task_id}/runs/{run_id}/
```

**Run Info File:**
```
{storage_root}/{project_id}/{task_id}/runs/{run_id}/run-info.yaml
```

---

## RunInfo Schema

### File: `run-info.yaml`

The run-info.yaml file contains all metadata about a single agent execution.

### Complete Schema

```yaml
# Schema version (for future evolution)
version: 1

# Unique identifiers
run_id: 20260205-1030451234-12345-0
project_id: my-project
task_id: task-001

# Relationship tracking
parent_run_id: ""                         # Empty if root run
previous_run_id: ""                       # Previous run in restart chain

# Agent configuration
agent: claude                             # Agent type (claude, codex, gemini, etc.)
agent_version: "claude 2.1.49"           # Detected agent CLI version (omitempty)

# Process information
pid: 12345                                # Process ID of the agent
pgid: 12345                               # Process Group ID (for child tracking)

# Timing
start_time: 2026-02-05T10:30:45.123Z      # UTC start timestamp
end_time: 2026-02-05T10:35:50.456Z        # UTC end timestamp (zero if running)

# Completion status
exit_code: 0                              # Exit code (-1 if not finished)
status: completed                         # running, completed, failed

# Working directory and paths
cwd: /path/to/projects/my-project      # Working directory
prompt_path: ""                           # Path to prompt file (optional)
output_path: output.md                    # Path to output file
stdout_path: stdout                       # Path to stdout capture
stderr_path: stderr                       # Path to stderr capture

# Command line (for debugging)
commandline: "claude --prompt 'task description'"

# Error details (omitempty)
error_summary: ""                         # Human-readable error summary on failure
```

### Field Descriptions

#### Core Identifiers

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | int | No | Schema version (currently 1) for future evolution |
| `run_id` | string | Yes | Unique run identifier (format: `YYYYMMDD-HHMMSSMMMM-PID-SEQ`) |
| `project_id` | string | Yes | Project identifier (Java identifier rules) |
| `task_id` | string | Yes | Task identifier within project |

#### Relationships

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `parent_run_id` | string | No | Parent run ID for child runs (empty for root runs) |
| `previous_run_id` | string | No | Previous run ID in restart chain (empty for first run) |

#### Agent Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `agent` | string | Yes | Agent type: `claude`, `codex`, `gemini`, `perplexity`, `xai` |
| `agent_version` | string | No | Detected agent CLI version string (omitted for REST agents or if detection fails) |

#### Process Information

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pid` | int | Yes | Process ID of the agent process |
| `pgid` | int | Yes | Process Group ID (Unix: same as PID, Windows: same as PID) |

#### Timing

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `start_time` | timestamp | Yes | UTC start time (RFC3339 format) |
| `end_time` | timestamp | No | UTC end time (zero value if still running) |

#### Status

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `exit_code` | int | No | Process exit code (-1 if not finished, 0 = success, >0 = failure) |
| `status` | string | Yes | Run status: `running`, `completed`, `failed` |

#### File Paths

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `cwd` | string | No | Working directory where agent was executed |
| `prompt_path` | string | No | Path to prompt file (if applicable) |
| `output_path` | string | No | Path to structured output file (typically `output.md`) |
| `stdout_path` | string | No | Path to stdout capture (relative to run directory) |
| `stderr_path` | string | No | Path to stderr capture (relative to run directory) |
| `commandline` | string | No | Full command line used to start agent (for debugging) |
| `error_summary` | string | No | Human-readable error summary written on run failure (omitempty) |

### Status Values

**Status Enum:**

| Value | Description | Exit Code | Use Case |
|-------|-------------|-----------|----------|
| `running` | Agent is currently executing | -1 | Active runs |
| `completed` | Agent completed successfully | 0 | Successful completion |
| `failed` | Agent failed with error | >0 | Failed execution |

**Constants in Code:**
```go
const (
    StatusRunning   = "running"    // In progress
    StatusCompleted = "completed"  // Success
    StatusFailed    = "failed"     // Failure
)
```

### Example: Running Task

```yaml
run_id: 20260205-1030451234-12345-0
project_id: my-project
task_id: task-001
agent: claude
pid: 12345
pgid: 12345
status: running
start_time: 2026-02-05T10:30:45.123Z
end_time: 0001-01-01T00:00:00Z          # Zero value (not finished)
exit_code: -1                            # Not finished
cwd: /path/to/projects/my-project
stdout_path: stdout
stderr_path: stderr
```

### Example: Completed Task

```yaml
run_id: 20260205-1030451234-12345-0
project_id: my-project
task_id: task-001
agent: codex
pid: 12345
pgid: 12345
status: completed
start_time: 2026-02-05T10:30:45.123Z
end_time: 2026-02-05T10:35:50.456Z      # Completion time
exit_code: 0                             # Success
cwd: /path/to/projects/my-project
output_path: output.md
stdout_path: stdout
stderr_path: stderr
commandline: codex --prompt "Implement feature X"
```

### Example: Failed Task

```yaml
run_id: 20260205-1030451234-12345-0
project_id: my-project
task_id: task-001
agent: gemini
pid: 12345
pgid: 12345
status: failed
start_time: 2026-02-05T10:30:45.123Z
end_time: 2026-02-05T10:31:50.123Z      # Failed quickly
exit_code: 1                             # Error code
cwd: /path/to/projects/my-project
stderr_path: stderr
```

### Example: Child Run

```yaml
run_id: 20260205-1045127890-67890-0
parent_run_id: 20260205-1030451234-12345-0  # Parent run ID
project_id: my-project
task_id: task-001
agent: claude
pid: 67890
pgid: 67890
status: running
start_time: 2026-02-05T10:45:12.789Z
exit_code: -1
cwd: /path/to/projects/my-project
```

### Example: Restart Chain

```yaml
# First run (failed)
run_id: 20260205-1030451234-12345-0
project_id: my-project
task_id: task-001
agent: claude
status: failed
exit_code: 1
start_time: 2026-02-05T10:30:45.123Z
end_time: 2026-02-05T10:31:00.000Z

# Second run (restart)
run_id: 20260205-1031054567-12345-1
previous_run_id: 20260205-1030451234-12345-0  # Links to previous run
project_id: my-project
task_id: task-001
agent: claude
status: completed
exit_code: 0
start_time: 2026-02-05T10:31:05.456Z
end_time: 2026-02-05T10:35:00.000Z
```

---

## Atomic Write Pattern

All metadata writes use an atomic write pattern to prevent corruption during concurrent access or crashes.

### The Pattern

```
1. Create temporary file with unique name
2. Write data to temporary file
3. fsync() - Force data to disk
4. chmod() - Set correct permissions
5. Rename temporary file to final name (atomic operation)
6. Cleanup on failure
```

### Implementation Details

**Reference:** `internal/storage/atomic.go`

```go
func writeFileAtomic(path string, data []byte) error {
    dir := filepath.Dir(path)

    // 1. Create temporary file in same directory
    tmpFile, err := os.CreateTemp(dir, "run-info.*.yaml.tmp")
    if err != nil {
        return errors.Wrap(err, "create temp file")
    }
    tmpName := tmpFile.Name()

    // Cleanup on failure
    success := false
    defer func() {
        if !success {
            _ = tmpFile.Close()
            _ = os.Remove(tmpName)
        }
    }()

    // 2. Write data to temporary file
    if _, err := tmpFile.Write(data); err != nil {
        return errors.Wrap(err, "write temp file")
    }

    // 3. fsync() - Force data to disk (durability guarantee)
    if err := tmpFile.Sync(); err != nil {
        return errors.Wrap(err, "fsync temp file")
    }

    // 4. Set file permissions
    if err := tmpFile.Chmod(0o644); err != nil {
        return errors.Wrap(err, "chmod temp file")
    }

    // Close before rename
    if err := tmpFile.Close(); err != nil {
        return errors.Wrap(err, "close temp file")
    }

    // 5. Rename (atomic operation on POSIX systems)
    if err := os.Rename(tmpName, path); err != nil {
        // Windows workaround: delete + rename
        if runtime.GOOS == "windows" {
            if removeErr := os.Remove(path); removeErr == nil {
                if renameErr := os.Rename(tmpName, path); renameErr == nil {
                    success = true
                    return nil
                }
            }
        }
        return errors.Wrap(err, "rename temp file")
    }

    success = true
    return nil
}
```

### Why Atomic Writes?

**Problem:** Concurrent or interrupted writes can corrupt metadata:
```
Writer A: Opens run-info.yaml
Writer A: Writes first half of data
[CRASH or concurrent write]
Reader: Reads partial/corrupted YAML
```

**Solution:** Atomic writes ensure readers always see complete, valid data:
```
Writer A: Creates run-info.12345.yaml.tmp
Writer A: Writes all data to temp file
Writer A: fsync() ensures data is on disk
Writer A: Renames temp → run-info.yaml (atomic)
Reader: Always sees either old or new version, never partial
```

### POSIX Guarantees

On POSIX systems (Linux, macOS, BSD):
- `rename()` is atomic
- If process crashes during write, temp file remains, final file is unchanged
- If process crashes during rename, either old or new version exists, never partial

### Windows Considerations

On Windows:
- `rename()` is NOT atomic if destination exists
- Workaround: `Remove() + Rename()` (small race window)
- Recommendation: Use WSL2 for production workloads

### Temporary File Naming

**Pattern:** `run-info.*.yaml.tmp`

**Example:** `run-info.23456.yaml.tmp`

**Cleanup:** Deferred cleanup ensures temp files are removed on error.

### Read-Copy-Update Pattern

**Used for Updates:**

```go
func UpdateRunInfo(path string, update func(*RunInfo) error) error {
    // 1. Read current version
    info, err := ReadRunInfo(path)
    if err != nil {
        return err
    }

    // 2. Apply updates
    if err := update(info); err != nil {
        return err
    }

    // 3. Write new version atomically
    if err := WriteRunInfo(path, info); err != nil {
        return err
    }

    return nil
}
```

**Example Usage:**
```go
storage.UpdateRunStatus(runID, StatusCompleted, 0)
// Internally:
// 1. Read run-info.yaml
// 2. Update status, exit_code, end_time fields
// 3. Write atomically
```

---

## Parent-Child Relationships

The storage layer supports hierarchical run relationships for multi-agent workflows.

### Relationship Types

**1. Parent-Child (Spawned Agents):**
```
Parent Run: 20260205-103045123-12345
    ├─ Child Run 1: 20260205-104512789-67890 (parent_run_id set)
    └─ Child Run 2: 20260205-105012345-67891 (parent_run_id set)
```

**2. Restart Chain (Ralph Loop):**
```
First Run:  20260205-103045123-12345 (failed)
    └─ Second Run: 20260205-103105456-12345 (previous_run_id set)
        └─ Third Run: 20260205-103205789-12345 (previous_run_id set)
```

**3. Combined (Parent with Restarts):**
```
Parent Run: 20260205-103045123-12345
    └─ Child Run 1: 20260205-104512789-67890 (parent_run_id set)
        └─ Child Run 1 Restart: 20260205-104612890-67890 (parent_run_id + previous_run_id set)
```

### Schema Fields

```yaml
# Root run (no parent, first in chain)
parent_run_id: ""
previous_run_id: ""

# Child run (spawned by parent)
parent_run_id: 20260205-1030451234-12345-0
previous_run_id: ""

# Restarted run (Ralph loop)
parent_run_id: ""
previous_run_id: 20260205-1030451234-12345-0

# Child run that was restarted
parent_run_id: 20260205-1030451234-12345-0
previous_run_id: 20260205-104512789-67890
```

### Traversing Relationships

**Find Parent:**
```go
info, _ := storage.GetRunInfo(runID)
if info.ParentRunID != "" {
    parent, _ := storage.GetRunInfo(info.ParentRunID)
    // Process parent
}
```

**Find Previous Run:**
```go
info, _ := storage.GetRunInfo(runID)
if info.PreviousRunID != "" {
    previous, _ := storage.GetRunInfo(info.PreviousRunID)
    // Process previous run in chain
}
```

**Find All Runs in Restart Chain:**
```go
func GetRestartChain(storage Storage, runID string) ([]*RunInfo, error) {
    chain := []*RunInfo{}
    currentID := runID

    for currentID != "" {
        info, err := storage.GetRunInfo(currentID)
        if err != nil {
            return nil, err
        }
        chain = append(chain, info)
        currentID = info.PreviousRunID
    }

    // Reverse to get chronological order
    for i := 0; i < len(chain)/2; i++ {
        j := len(chain) - 1 - i
        chain[i], chain[j] = chain[j], chain[i]
    }

    return chain, nil
}
```

### Use Cases

**1. Wait for Child Completion (Ralph Loop):**
```go
// Check if any child runs are still running
children := findChildRuns(storage, parentRunID)
for _, child := range children {
    if child.Status == StatusRunning {
        // Wait for completion
        return waitForRun(child.RunID)
    }
}
```

**2. Calculate Total Restart Count:**
```go
chain := GetRestartChain(storage, runID)
restartCount := len(chain) - 1  // Exclude first run
```

**3. Build Run Tree:**
```go
type RunNode struct {
    Info     *RunInfo
    Children []*RunNode
}

func BuildRunTree(storage Storage, rootRunID string) (*RunNode, error) {
    info, err := storage.GetRunInfo(rootRunID)
    if err != nil {
        return nil, err
    }

    node := &RunNode{Info: info}

    // Find children
    allRuns, _ := storage.ListRuns(info.ProjectID, info.TaskID)
    for _, run := range allRuns {
        if run.ParentRunID == rootRunID {
            child, _ := BuildRunTree(storage, run.RunID)
            node.Children = append(node.Children, child)
        }
    }

    return node, nil
}
```

---

## File Locking Mechanisms

The storage layer uses file locking for exclusive write access to the message bus. Run metadata files use atomic writes instead of locks.

### Storage Files: No Locking (Atomic Writes)

**Run metadata files (`run-info.yaml`) do NOT use file locks.**

Instead, they use atomic writes:
- Readers never see partial data
- Writers never corrupt existing data
- No lock contention
- Higher performance

### Message Bus Files: Exclusive Write Locks

**Message bus files (`messagebus.yaml`) use exclusive write locks.**

**Reference:** `internal/messagebus/messagebus.go`

#### Unix/Linux/macOS Implementation

**File:** `internal/messagebus/lock_unix.go`

```go
func tryFlockExclusive(file *os.File) (bool, error) {
    // LOCK_EX: Exclusive lock (only one writer at a time)
    // LOCK_NB: Non-blocking (return immediately if lock unavailable)
    err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err == nil {
        return true, nil  // Lock acquired
    }
    if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
        return false, nil  // Lock held by another process
    }
    return false, err  // Error
}

func unlockFile(file *os.File) error {
    return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
```

**Properties:**
- `flock()` is advisory (processes can ignore it)
- `LOCK_EX` ensures only one writer at a time
- `LOCK_NB` prevents blocking (returns immediately)
- Locks are released automatically when file is closed
- Locks are NOT inherited by child processes

#### Windows Implementation

**File:** `internal/messagebus/lock_windows.go`

```go
func tryFlockExclusive(file *os.File) (bool, error) {
    handle := syscall.Handle(file.Fd())
    var overlapped syscall.Overlapped

    // LOCKFILE_EXCLUSIVE_LOCK: Exclusive lock
    // LOCKFILE_FAIL_IMMEDIATELY: Non-blocking
    err := syscall.LockFileEx(
        handle,
        syscall.LOCKFILE_EXCLUSIVE_LOCK|syscall.LOCKFILE_FAIL_IMMEDIATELY,
        0, 1, 0, &overlapped,
    )

    if err == nil {
        return true, nil
    }
    if err == syscall.ERROR_LOCK_VIOLATION {
        return false, nil  // Lock held by another process
    }
    return false, err
}

func unlockFile(file *os.File) error {
    handle := syscall.Handle(file.Fd())
    var overlapped syscall.Overlapped
    return syscall.UnlockFileEx(handle, 0, 1, 0, &overlapped)
}
```

**Windows Caveats:**
- Mandatory locks (processes MUST respect them)
- May block readers (unlike Unix advisory locks)
- Less tested in production

### Lock Acquisition with Retry

**Reference:** `internal/messagebus/messagebus.go`

```go
func LockExclusive(file *os.File, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    backoff := 10 * time.Millisecond

    for {
        // Try to acquire lock (non-blocking)
        acquired, err := tryFlockExclusive(file)
        if err != nil {
            return errors.Wrap(err, "flock exclusive")
        }
        if acquired {
            return nil  // Success
        }

        // Check timeout
        if time.Now().After(deadline) {
            return errors.New("lock timeout")
        }

        // Exponential backoff
        time.Sleep(backoff)
        backoff = backoff * 2
        if backoff > 500*time.Millisecond {
            backoff = 500 * time.Millisecond
        }
    }
}
```

**Retry Strategy:**
- Initial backoff: 10ms
- Exponential increase: 2x each retry
- Maximum backoff: 500ms
- Total timeout: 10 seconds (configurable)

### Message Bus Write Flow

```
1. Open messagebus.yaml (O_WRONLY | O_APPEND | O_CREATE)
2. LockExclusive(file, timeout=10s)
3. Serialize message to YAML
4. Write to file (O_APPEND ensures atomic append)
5. fsync() - Force disk write
6. Unlock(file)
7. Close file
```

### Message Bus Read Flow

**No locks required!**

```
1. Open messagebus.yaml (O_RDONLY)
2. Read entire file (no lock)
3. Parse YAML documents
4. Close file
```

### Concurrency Model

**Storage Layer (run-info.yaml):**
```
Writers: Atomic writes (no locks)
Readers: Direct reads (no locks)
Concurrency: Unlimited readers, unlimited writers (safe via atomic writes)
```

**Message Bus (messagebus.yaml):**
```
Writers: Exclusive lock required (one at a time)
Readers: No lock (lockless reads)
Concurrency: Unlimited readers, one writer at a time
```

### Lock Debugging

**Check for lock contention:**
```bash
# Unix: Check for flock() calls
sudo lsof /path/to/messagebus.yaml

# Strace to see lock attempts
strace -e flock ./conductor-loop
```

### Known Limitations

**Unix/Linux/macOS:**
- Advisory locks (can be ignored by non-cooperating processes)
- NFS may have unreliable locking (use local storage)
- Locks released on process exit (no cleanup needed)

**Windows:**
- Mandatory locks (may block readers)
- Less tested (recommend WSL2)
- Lock inheritance behavior differs from Unix

---

## RunID Generation Format

RunID is a unique identifier for each agent execution, designed for lexical sorting and global uniqueness.

### Format Specification

```
{YYYYMMDD-HHMMSS0000}-{PID}-{SEQ}
```

**Components:**
- `YYYYMMDD`: Year, month, day (UTC)
- `HHMMSS`: Hour, minute, second (UTC)
- `0000`: Literal suffix (fixed width, for legacy compatibility/sorting)
- `PID`: Process ID of the run-agent process
- `SEQ`: Process-local atomic counter (starts at 0, increments per run)

**Example:** `20260205-1030450000-12345-0`

### Implementation

**Reference:** `internal/runner/orchestrator.go`

```go
func newRunID(now time.Time, pid int) string {
    stamp := now.UTC().Format("20060102-1504050000")  // Seconds + literal 0000
    // + atomic seq counter appended
}
```

**Go time format:** `20060102-1504050000` (where `0000` is literal text, not fractional seconds format token).

**Collision protection:** A process-local atomic counter (`SEQ`) ensures same-second runs within the same process get unique IDs.

### Properties

**1. Lexical Sorting:**
```
20260205-1030451234-12345-0  < 20260205-1030451235-12345-1
20260205-1030451234-12345-0  < 20260205-1030460000-12345-0
```

**2. Global Uniqueness:**
- PID separates different processes at the same timestamp
- SEQ counter separates same-process same-timestamp runs (collision fix)

**3. Human Readability:**
```
20260205-1030451234-12345-0
│       │           │     │
│       │           │     └─ Sequence counter: 0
│       │           └─ Process ID: 12345
│       └─ Time: 10:30:45.1234 UTC (4 sub-second digits)
└─ Date: 2026-02-05
```

**4. Length:** 27–35 characters (varies with PID length and counter size)

### Config File Precedence

When auto-detecting configuration:
```
config.yaml > config.yml > config.hcl
```
YAML formats are checked first. Both YAML and HCL are fully supported.

### Testing

**Mock Time and PID for Deterministic Tests:**

```go
func TestRunIDGeneration(t *testing.T) {
    st, _ := NewStorage(t.TempDir())

    // Mock time
    st.now = func() time.Time {
        return time.Date(2026, 2, 5, 10, 30, 45, 123000000, time.UTC)
    }

    // Mock PID
    st.pid = func() int {
        return 12345
    }

    runID := st.newRunID()
    expected := "20260205-103045123-12345"
    if runID != expected {
        t.Errorf("expected %s, got %s", expected, runID)
    }
}
```

---

## Task State Files

Task directories contain special files for state management and completion signaling.

### TASK_STATE.md

**Purpose:** Current task state maintained by root agent

**Location:** `{storage_root}/{project_id}/{task_id}/TASK_STATE.md`

**Format:** Free-text Markdown

**Encoding:** UTF-8 without BOM (strict enforcement)

**Properties:**
- Short, current status only (not a log)
- Updated by root agent only
- Uses atomic write pattern (temp + rename)
- Overwritten on each update (not appended)

**Example Content:**
```markdown
# Task State

## Current Status
Working on feature implementation.

## Progress
- [x] Design phase complete
- [x] Implementation started
- [ ] Testing phase
- [ ] Documentation

## Next Steps
1. Complete unit tests
2. Run integration tests
3. Update documentation

Last updated: 2026-02-05 10:35:00 UTC
```

**Update Pattern:**
```bash
# Agent script updates state
cat > TASK_STATE.md.tmp << EOF
# Task State
...new content...
EOF

sync TASK_STATE.md.tmp
mv TASK_STATE.md.tmp TASK_STATE.md
```

**Read Pattern:**
```go
statePath := filepath.Join(taskDir, "TASK_STATE.md")
data, err := os.ReadFile(statePath)
if err != nil {
    // Handle missing file (task not started yet)
}
state := string(data)
```

### DONE File

**Purpose:** Completion marker created by root agent when task is complete

**Location:** `{storage_root}/{project_id}/{task_id}/DONE`

**Format:** Empty marker file (0 bytes)

**Properties:**
- Created by root agent to signal completion
- Checked by Ralph Loop after each agent exit
- If present, Ralph Loop stops (no more restarts)
- Deleting DONE file restarts the Ralph Loop on next run

**Ralph Loop Detection:**

**Reference:** `internal/runner/ralph.go:227-240`

```go
func (rl *RalphLoop) doneExists() (bool, error) {
    path := filepath.Join(rl.taskDir, "DONE")
    info, err := os.Stat(path)
    if err == nil {
        if info.IsDir() {
            return false, errors.New("DONE is a directory")
        }
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, errors.Wrap(err, "stat DONE")
}
```

**Agent Creation:**
```bash
# Inside agent script
echo "" > $TASK_DIR/DONE

# Or from agent code
os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644)
```

**Ralph Loop Behavior:**
```
Agent Exit (code=0)
  ├─ Check for DONE file
  │  ├─ If exists: STOP (task complete)
  │  └─ If not exists: Check restart conditions
  │
Agent Exit (code>0)
  ├─ Check for DONE file
  │  ├─ If exists: STOP (task complete despite error)
  │  └─ If not exists: RESTART (up to max restarts)
```

**Example Workflow:**
```bash
# Task starts
RunID: 20260205-103045123-12345
Status: running

# Task fails, no DONE file
Exit code: 1
Ralph Loop: RESTART

# New run
RunID: 20260205-103145456-12345
Status: running

# Task succeeds, creates DONE file
Exit code: 0
touch DONE
Ralph Loop: STOP (DONE file detected)

# No more restarts
```

**Deleting DONE to Restart:**
```bash
# Remove DONE file
rm ~/run-agent/my-project/task-001/DONE

# Restart task
conductor-loop task start my-project task-001

# Ralph Loop will run again
```

### Task Facts

**File Pattern:** `TASK-FACTS-{timestamp}.md`

**Location:** `{storage_root}/{project_id}/{task_id}/TASK-FACTS-{timestamp}.md`

**Format:** YAML front matter + Markdown body

**Encoding:** UTF-8 without BOM (strict enforcement)

**Example:**
```markdown
---
created_at: 2026-02-05T10:30:45Z
scope: task
run_id: 20260205-1030451234-12345-0
title: Implementation Decision
---

# Decision: Use React for Frontend

## Context
Need to choose frontend framework.

## Decision
Selected React 18 for the following reasons:
- Large ecosystem
- Strong TypeScript support
- Good performance

## Consequences
- Need to learn React hooks
- Build pipeline with Vite
```

### Attachments

**File Pattern:** `ATTACH-{timestamp}-{name}.{ext}`

**Location:** `{storage_root}/{project_id}/{task_id}/ATTACH-{timestamp}-{name}.{ext}`

**Example:**
```
ATTACH-20260205103045-screenshot.png
ATTACH-20260205104512-design-doc.pdf
ATTACH-20260205105623-error-log.txt
```

**Message Bus Reference:**
```yaml
---
msg_id: MSG-20060102-150405-000000001-PID00123-0001
type: attachment
attachment_path: ATTACH-20260205103045-screenshot.png
---
Screenshot of error
```

---

## Cleanup and Retention

### Current Policy

**Automatic cleanup is available via `run-agent gc`:**

- **Run directories:** `run-agent gc --root <root> --older-than 168h` deletes runs older than the specified duration. Use `--dry-run` to preview.
- **Message bus files:** `run-agent gc --rotate-bus --bus-max-size 10MB --root <root>` rotates bus files exceeding the size threshold. `WithAutoRotate(maxBytes)` also auto-rotates on each write when the threshold is exceeded.
- **Task state files and attachments:** No automatic cleanup; delete manually.

### Disk Space Monitoring

**Monitor storage usage:**
```bash
# Check total storage size
du -sh ~/run-agent/

# Check per-project size
du -sh ~/run-agent/*/

# Check per-task size
du -sh ~/run-agent/my-project/*/

# Check largest run directories
du -sh ~/run-agent/*/*/runs/*/ | sort -h | tail -20
```

### Manual Cleanup

**Delete old runs:**
```bash
# Delete runs older than 30 days
find ~/run-agent/*/*/runs/ -type d -mtime +30 -exec rm -rf {} +

# Delete failed runs only
find ~/run-agent/*/*/runs/ -type d -name "2025*" -exec \
  sh -c 'grep -q "status: failed" "$1/run-info.yaml" && rm -rf "$1"' _ {} \;
```

**Archive old projects:**
```bash
# Archive project to tar.gz
tar czf my-project-archive.tar.gz ~/run-agent/my-project/

# Delete archived project
rm -rf ~/run-agent/my-project/
```

### Future Retention Strategies

**Planned features (not implemented):**

1. **Configurable retention periods:**
```yaml
# Future config option
retention:
  keep_successful_runs_days: 30
  keep_failed_runs_days: 90
  keep_running_runs: true
```

2. **Automatic rotation (implemented):**
```bash
# Rotate bus files exceeding 10MB in a single pass
run-agent gc --rotate-bus --bus-max-size 10MB --root runs

# Or use WithAutoRotate for per-write automatic rotation
bus, _ := NewMessageBus(path, WithAutoRotate(10*1024*1024))
```

3. **Compression:**
```bash
# Compress old stdout/stderr files
gzip ~/run-agent/*/*/runs/*/stdout
gzip ~/run-agent/*/*/runs/*/stderr
```

### Best Practices

**For long-running production:**

1. Monitor disk space regularly
2. Archive completed projects
3. Delete very old runs manually
4. Consider mounting storage on separate disk
5. Use log rotation for stdout/stderr if files are large

---

## Query Operations

The storage layer provides efficient query operations via the Storage interface.

### Storage Interface

**Reference:** `internal/storage/storage.go:15-21`

```go
type Storage interface {
    CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
    UpdateRunStatus(runID string, status string, exitCode int) error
    GetRunInfo(runID string) (*RunInfo, error)
    ListRuns(projectID, taskID string) ([]*RunInfo, error)
}
```

### CreateRun

**Purpose:** Create a new run directory and persist initial metadata

**Signature:**
```go
func (s *FileStorage) CreateRun(
    projectID string,
    taskID string,
    agentType string,
) (*RunInfo, error)
```

**Example:**
```go
storage, _ := NewStorage("~/run-agent")

info, err := storage.CreateRun("my-project", "task-001", "claude")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created run: %s\n", info.RunID)
fmt.Printf("Status: %s\n", info.Status)
fmt.Printf("PID: %d\n", info.PID)
```

**Output:**
```
Created run: 20260205-103045123-12345
Status: running
PID: 12345
```

**What it does:**
1. Validate inputs (non-empty projectID, taskID, agentType)
2. Generate unique RunID (timestamp + PID)
3. Create directory: `{root}/{projectID}/{taskID}/runs/{runID}/`
4. Build RunInfo structure (status=running, exit_code=-1)
5. Write run-info.yaml atomically
6. Add to in-memory index
7. Return RunInfo

**Performance:** O(1) with one fsync (~5-10ms)

### GetRunInfo

**Purpose:** Load run metadata by run ID

**Signature:**
```go
func (s *FileStorage) GetRunInfo(runID string) (*RunInfo, error)
```

**Example:**
```go
info, err := storage.GetRunInfo("20260205-103045123-12345")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Run: %s\n", info.RunID)
fmt.Printf("Status: %s\n", info.Status)
fmt.Printf("Exit Code: %d\n", info.ExitCode)
fmt.Printf("Started: %s\n", info.StartTime)
fmt.Printf("Ended: %s\n", info.EndTime)
```

**What it does:**
1. Check in-memory index first (fast path)
2. If not found, glob: `{root}/*/*/runs/{runID}/run-info.yaml`
3. Read and parse YAML
4. Add to index for future lookups
5. Return RunInfo

**Performance:**
- Cache hit: O(1) (~0.1ms)
- Cache miss: O(n) where n = number of projects (~10-50ms)

**Error cases:**
- Run ID not found
- Multiple runs with same ID (should never happen)
- YAML parse error

### UpdateRunStatus

**Purpose:** Update run status, exit code, and end time

**Signature:**
```go
func (s *FileStorage) UpdateRunStatus(
    runID string,
    status string,
    exitCode int,
) error
```

**Example:**
```go
// Mark run as completed
err := storage.UpdateRunStatus(
    "20260205-103045123-12345",
    storage.StatusCompleted,
    0,
)

// Mark run as failed
err := storage.UpdateRunStatus(
    "20260205-103045123-12345",
    storage.StatusFailed,
    1,
)
```

**What it does:**
1. Validate inputs (non-empty runID, status)
2. Look up run-info.yaml path (via index or glob)
3. Read current run-info.yaml
4. Update status, exit_code, end_time fields
5. Write run-info.yaml atomically
6. Return error or nil

**Performance:** O(1) with read + fsync (~5-10ms)

**Atomicity:** Uses read-copy-update pattern with atomic write

### ListRuns

**Purpose:** List all runs for a project/task (sorted by RunID)

**Signature:**
```go
func (s *FileStorage) ListRuns(
    projectID string,
    taskID string,
) ([]*RunInfo, error)
```

**Example:**
```go
runs, err := storage.ListRuns("my-project", "task-001")
if err != nil {
    log.Fatal(err)
}

for _, run := range runs {
    fmt.Printf("%s: %s (exit=%d)\n",
        run.RunID, run.Status, run.ExitCode)
}
```

**Output:**
```
20260205-103045123-12345: completed (exit=0)
20260205-103145456-12345: failed (exit=1)
20260205-103245789-12345: running (exit=-1)
```

**What it does:**
1. Validate inputs (non-empty projectID, taskID)
2. Read directory: `{root}/{projectID}/{taskID}/runs/`
3. For each subdirectory:
   - Read run-info.yaml
   - Parse RunInfo
   - Add to list
4. Sort by RunID (lexical = chronological)
5. Return list

**Performance:** O(n) where n = number of runs (~1-10ms for 100 runs)

**Sorting:** Lexical sort on RunID → chronological order

### Advanced Query Examples

**Find all running runs:**
```go
func FindRunningRuns(storage Storage, projectID, taskID string) ([]*RunInfo, error) {
    runs, err := storage.ListRuns(projectID, taskID)
    if err != nil {
        return nil, err
    }

    running := []*RunInfo{}
    for _, run := range runs {
        if run.Status == StatusRunning {
            running = append(running, run)
        }
    }
    return running, nil
}
```

**Find latest run:**
```go
func FindLatestRun(storage Storage, projectID, taskID string) (*RunInfo, error) {
    runs, err := storage.ListRuns(projectID, taskID)
    if err != nil {
        return nil, err
    }
    if len(runs) == 0 {
        return nil, errors.New("no runs found")
    }
    // Last element (sorted by RunID)
    return runs[len(runs)-1], nil
}
```

**Count successful runs:**
```go
func CountSuccessfulRuns(storage Storage, projectID, taskID string) (int, error) {
    runs, err := storage.ListRuns(projectID, taskID)
    if err != nil {
        return 0, err
    }

    count := 0
    for _, run := range runs {
        if run.Status == StatusCompleted && run.ExitCode == 0 {
            count++
        }
    }
    return count, nil
}
```

**Find runs by time range:**
```go
func FindRunsByTimeRange(
    storage Storage,
    projectID, taskID string,
    start, end time.Time,
) ([]*RunInfo, error) {
    runs, err := storage.ListRuns(projectID, taskID)
    if err != nil {
        return nil, err
    }

    filtered := []*RunInfo{}
    for _, run := range runs {
        if run.StartTime.After(start) && run.StartTime.Before(end) {
            filtered = append(filtered, run)
        }
    }
    return filtered, nil
}
```

### In-Memory Index

**Purpose:** Fast RunID → path lookups without filesystem scanning

**Structure:**
```go
type FileStorage struct {
    root     string
    runIndex map[string]string    // RunID → run-info.yaml path
    mu       sync.RWMutex          // Protects runIndex
}
```

**Operations:**
```go
// Add to index (write lock)
func (s *FileStorage) trackRun(runID, path string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.runIndex[runID] = path
}

// Lookup from index (read lock)
func (s *FileStorage) lookupRun(runID string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    path, ok := s.runIndex[runID]
    return path, ok
}
```

**Properties:**
- Populated on CreateRun and first GetRunInfo
- Not persisted (rebuilt on restart)
- Thread-safe (RWMutex)
- O(1) lookups

**Rebuild on startup:**
```go
func (s *FileStorage) rebuildIndex() error {
    pattern := filepath.Join(s.root, "*", "*", "runs", "*", "run-info.yaml")
    matches, _ := filepath.Glob(pattern)

    for _, path := range matches {
        info, err := ReadRunInfo(path)
        if err != nil {
            continue
        }
        s.trackRun(info.RunID, path)
    }
    return nil
}
```

---

## Platform Compatibility

### POSIX Systems (Unix, Linux, macOS, BSD)

**Fully Supported Features:**

| Feature | Status | Notes |
|---------|--------|-------|
| Atomic writes (rename) | ✅ Full | POSIX guarantees atomic rename |
| File locking (flock) | ✅ Full | Advisory locks, non-blocking |
| fsync durability | ✅ Full | Guaranteed disk flush |
| Directory creation | ✅ Full | mkdir -p semantics |
| UTF-8 encoding | ✅ Full | Native support |
| Process groups (PGID) | ✅ Full | setpgid, kill -PGID |

**Tested Platforms:**
- macOS (Darwin) 11.0+
- Linux (kernel 4.0+)
  - Ubuntu 20.04+
  - Debian 10+
  - RHEL/CentOS 8+
- FreeBSD 12.0+

**File System Requirements:**
- Local filesystem (ext4, XFS, APFS, HFS+)
- NOT network filesystems (NFS, SMB) - unreliable locking

### Windows

**Partial Support:**

| Feature | Status | Notes |
|---------|--------|-------|
| Atomic writes (rename) | ⚠️ Workaround | Remove + rename (small race window) |
| File locking (LockFileEx) | ⚠️ Limited | Mandatory locks (may block readers) |
| fsync durability | ✅ Full | FlushFileBuffers |
| Directory creation | ✅ Full | CreateDirectory |
| UTF-8 encoding | ✅ Full | Requires UTF-8 mode |
| Process groups (PGID) | ❌ Limited | No process groups (use job objects) |

**Known Issues:**

1. **Rename Not Atomic:**
   - Windows rename() fails if destination exists
   - Workaround: Remove destination first (race window)
   - Recommendation: Use WSL2 for production

2. **Mandatory File Locks:**
   - LockFileEx creates mandatory locks
   - May block readers (unlike Unix advisory locks)
   - Less tested in production

3. **Process Groups:**
   - No native process group support
   - Child processes may become orphans
   - Recommendation: Use Windows job objects or WSL2

**Example Rename Workaround:**

**Reference:** `internal/storage/atomic.go:94-104`

```go
if err := os.Rename(tmpName, path); err != nil {
    if runtime.GOOS == "windows" {
        // Windows workaround: delete + rename
        if removeErr := os.Remove(path); removeErr == nil {
            if renameErr := os.Rename(tmpName, path); renameErr == nil {
                success = true
                return nil
            }
        }
    }
    return errors.Wrap(err, "rename temp file")
}
```

**Windows Recommendation:**
```
✅ Use WSL2 (Windows Subsystem for Linux 2)
   - Full POSIX compatibility
   - Native Linux kernel
   - Better performance

❌ Native Windows support marked as experimental
   - Use for development only
   - Not recommended for production
```

### Network File Systems

**NOT SUPPORTED:**

| Filesystem | Status | Issues |
|------------|--------|--------|
| NFS | ❌ Not supported | Unreliable locking, caching issues |
| SMB/CIFS | ❌ Not supported | Lock behavior varies by version |
| SSHFS | ❌ Not supported | Poor lock support |
| Cloud storage (S3, etc.) | ❌ Not supported | No locking, eventual consistency |

**Recommendation:** Always use local filesystems

### Docker/Containers

**Supported:**
- ✅ Linux containers on Linux host
- ✅ Linux containers on macOS host (Docker Desktop)
- ⚠️ Linux containers on Windows host (use WSL2 backend)

**Example Docker Setup:**
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o conductor-loop
VOLUME /data
CMD ["./conductor-loop", "--storage", "/data"]
```

**Mount local storage:**
```bash
docker run -v ~/run-agent:/data conductor-loop
```

### File System Testing

**Verify atomic writes:**
```bash
# Test atomic rename behavior
cd /tmp
touch test.txt
echo "data" > test.tmp
mv test.tmp test.txt
echo $?  # Should be 0 (success)
```

**Verify flock support:**
```bash
# Test flock (Unix only)
flock /tmp/test.lock echo "Locked command"
```

**Verify UTF-8 support:**
```bash
# Test UTF-8 encoding
echo "Test: 测试 тест" > test.txt
file test.txt  # Should show UTF-8
```

### Platform-Specific Code

**Build tags:**
```go
//go:build !windows
package messagebus
// Unix-specific implementation

//go:build windows
package messagebus
// Windows-specific implementation
```

**Runtime checks:**
```go
if runtime.GOOS == "windows" {
    // Windows-specific workaround
} else {
    // POSIX implementation
}
```

---

## Code Examples

### Complete Storage Workflow

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/jonnyzzz/conductor-loop/internal/storage"
)

func main() {
    // Initialize storage
    st, err := storage.NewStorage("~/run-agent")
    if err != nil {
        log.Fatal(err)
    }

    // Create a new run
    info, err := st.CreateRun("my-project", "task-001", "claude")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created run: %s\n", info.RunID)
    fmt.Printf("Status: %s\n", info.Status)
    fmt.Printf("PID: %d\n", info.PID)

    // Simulate agent execution
    time.Sleep(5 * time.Second)

    // Update run status
    err = st.UpdateRunStatus(info.RunID, storage.StatusCompleted, 0)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Run marked as completed")

    // Retrieve run info
    updated, err := st.GetRunInfo(info.RunID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Final status: %s\n", updated.Status)
    fmt.Printf("Exit code: %d\n", updated.ExitCode)
    fmt.Printf("Duration: %s\n", updated.EndTime.Sub(updated.StartTime))

    // List all runs for task
    runs, err := st.ListRuns("my-project", "task-001")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\nTotal runs: %d\n", len(runs))
    for _, run := range runs {
        fmt.Printf("  %s: %s (exit=%d)\n",
            run.RunID, run.Status, run.ExitCode)
    }
}
```

### Atomic Write Example

```go
package main

import (
    "fmt"
    "github.com/jonnyzzz/conductor-loop/internal/storage"
)

func updateTaskState(taskDir string, state string) error {
    // Create RunInfo
    info := &storage.RunInfo{
        RunID:     "20260205-103045123-12345",
        ProjectID: "my-project",
        TaskID:    "task-001",
        AgentType: "claude",
        PID:       12345,
        PGID:      12345,
        Status:    storage.StatusRunning,
        StartTime: time.Now().UTC(),
        ExitCode:  -1,
    }

    // Write atomically
    path := filepath.Join(taskDir, "runs", info.RunID, "run-info.yaml")
    if err := storage.WriteRunInfo(path, info); err != nil {
        return err
    }

    fmt.Println("Run info written atomically")
    return nil
}
```

### Query Patterns

```go
package main

import (
    "fmt"
    "time"
    "github.com/jonnyzzz/conductor-loop/internal/storage"
)

func queryExamples(st storage.Storage) {
    projectID := "my-project"
    taskID := "task-001"

    // Find all running runs
    runs, _ := st.ListRuns(projectID, taskID)
    running := []*storage.RunInfo{}
    for _, run := range runs {
        if run.Status == storage.StatusRunning {
            running = append(running, run)
        }
    }
    fmt.Printf("Running runs: %d\n", len(running))

    // Find latest run
    if len(runs) > 0 {
        latest := runs[len(runs)-1]
        fmt.Printf("Latest run: %s (%s)\n", latest.RunID, latest.Status)
    }

    // Count by status
    completed, failed := 0, 0
    for _, run := range runs {
        switch run.Status {
        case storage.StatusCompleted:
            completed++
        case storage.StatusFailed:
            failed++
        }
    }
    fmt.Printf("Stats: %d completed, %d failed\n", completed, failed)

    // Find runs in last hour
    since := time.Now().Add(-1 * time.Hour)
    recent := []*storage.RunInfo{}
    for _, run := range runs {
        if run.StartTime.After(since) {
            recent = append(recent, run)
        }
    }
    fmt.Printf("Recent runs (1h): %d\n", len(recent))
}
```

### Parent-Child Traversal

```go
package main

import (
    "fmt"
    "github.com/jonnyzzz/conductor-loop/internal/storage"
)

func buildRunTree(st storage.Storage, rootRunID string) error {
    // Get root run
    root, err := st.GetRunInfo(rootRunID)
    if err != nil {
        return err
    }

    fmt.Printf("Root: %s (%s)\n", root.RunID, root.Status)

    // Find children
    runs, _ := st.ListRuns(root.ProjectID, root.TaskID)
    for _, run := range runs {
        if run.ParentRunID == rootRunID {
            fmt.Printf("  Child: %s (%s)\n", run.RunID, run.Status)

            // Find grandchildren
            for _, child := range runs {
                if child.ParentRunID == run.RunID {
                    fmt.Printf("    Grandchild: %s (%s)\n",
                        child.RunID, child.Status)
                }
            }
        }
    }

    return nil
}
```

---

## API Task Summary Response

The `GET /api/projects/{projectID}/tasks` endpoint includes a `run_counts` field in each task object. This is a map from run status to integer count, computed at query time from the task's runs:

```json
{
  "id": "task-001",
  "project_id": "my-project",
  "status": "running",
  "run_count": 5,
  "run_counts": {
    "running": 1,
    "completed": 3,
    "failed": 1
  }
}
```

This field enables the frontend to show per-status badges without a separate API call.

## See Also

- [Architecture Overview](architecture.md)
- [Subsystem Deep-Dives](subsystems.md)
- [Ralph Loop Specification](ralph-loop.md)
- [Agent Protocol](agent-protocol.md)
- [Message Bus Architecture](../specifications/subsystem-storage-layout.md)

---

**Last Updated:** 2026-02-23 (facts-validated)
**Version:** 1.1.0
**Package:** `internal/storage/`
