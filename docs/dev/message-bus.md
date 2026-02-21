# Message Bus Protocol

This document provides a comprehensive specification of the message bus implementation in conductor-loop, covering the protocol design, concurrency model, file format, and implementation details.

## Table of Contents

1. [Overview](#overview)
2. [Design Philosophy](#design-philosophy)
3. [Message Bus Protocol](#message-bus-protocol)
4. [O_APPEND + flock Design](#o_append--flock-design)
5. [Message ID Generation](#message-id-generation)
6. [Concurrency Guarantees](#concurrency-guarantees)
7. [Write Durability Model](#write-durability-model)
8. [Message Format](#message-format)
9. [Read Operations](#read-operations)
10. [Write Operations](#write-operations)
11. [Race Condition Handling](#race-condition-handling)
12. [Platform Differences](#platform-differences)
13. [Performance Characteristics](#performance-characteristics)
14. [Known Limitations](#known-limitations)
15. [Implementation Reference](#implementation-reference)

---

## Overview

The message bus is an **append-only event log** that enables multi-agent coordination, event streaming, and task monitoring in conductor-loop. It provides an ordered stream of messages with strong inter-process consistency guarantees.

**Key Features:**
- Append-only log (messages never modified or deleted)
- Lockless reads (readers never block writers or other readers)
- Exclusive writes (one writer at a time via flock)
- High-throughput OS-cached writes (~37,000+ msg/sec measured)
- Lexically sortable message IDs
- Human-readable YAML format
- Platform-specific optimizations (Unix/Windows)

**Package:** `internal/messagebus/`

**Key Files:**
- `messagebus.go` - Core message bus implementation (367 lines)
- `msgid.go` - Message ID generation (19 lines)
- `lock.go` - Platform-agnostic locking interface (51 lines)
- `lock_unix.go` - Unix/Linux/macOS flock implementation (24 lines)
- `lock_windows.go` - Windows LockFileEx implementation (28 lines)

---

## Design Philosophy

The message bus design prioritizes **simplicity**, **throughput**, and **debuggability** over disk-level durability.

### Core Principles

1. **No External Dependencies**
   - Uses filesystem for storage (no database required)
   - YAML format for human readability
   - Standard library + minimal dependencies

2. **High Throughput via OS Cache**
   - Writes go to OS page cache (no fsync)
   - OS guarantees immediate read-after-write visibility across processes
   - Atomic appends via `O_APPEND` + `flock`

3. **Simple Concurrency Model**
   - Lockless reads: Readers never block
   - Exclusive writes: One writer at a time via `flock`
   - No lock contention on reads

4. **Human-Debuggable**
   - YAML format (not binary)
   - Readable message IDs with timestamps
   - Can inspect files with standard tools (`cat`, `less`, `grep`)

5. **Append-Only**
   - Messages never modified or deleted
   - Total ordering by timestamp
   - Audit log for debugging

### Trade-offs

**Advantages:**
- Simple implementation (~500 lines of code)
- No database setup/maintenance
- Easy to debug (human-readable files)
- High throughput (~37,000+ writes/sec measured)
- Lockless reads for high read throughput

**Disadvantages:**
- File size grows unbounded (need rotation)
- No complex queries (linear scan)
- Not durable against OS crash (messages may be lost before page-cache flush)
- Network filesystems may have issues (use local storage)

---

## Message Bus Protocol

### Message Structure

**Reference:** `internal/messagebus/messagebus.go:28-38`

```go
type Message struct {
    MsgID        string    `yaml:"msg_id"`          // Unique message ID
    Timestamp    time.Time `yaml:"ts"`              // UTC timestamp
    Type         string    `yaml:"type"`            // Event type
    ProjectID    string    `yaml:"project_id"`      // Project identifier
    TaskID       string    `yaml:"task_id"`         // Task identifier (optional)
    RunID        string    `yaml:"run_id"`          // Run identifier (optional)
    ParentMsgIDs []string  `yaml:"parents,omitempty"` // Parent messages (threading)
    Attachment   string    `yaml:"attachment_path,omitempty"` // Path to file
    Body         string    `yaml:"-"`               // Message body (not in YAML header)
}
```

**Field Descriptions:**

- **MsgID**: Unique identifier (lexically sortable, see [Message ID Generation](#message-id-generation))
- **Timestamp**: UTC timestamp (set automatically on append)
- **Type**: Event type (e.g., `agent_started`, `agent_output`, `agent_completed`)
- **ProjectID**: Required. Which project this message belongs to
- **TaskID**: Optional. Which task this message belongs to
- **RunID**: Optional. Which run this message belongs to
- **ParentMsgIDs**: Optional. Parent message IDs for threading (request-response relationships)
- **Attachment**: Optional. Path to attached file (relative to storage root)
- **Body**: Optional. Message content (stored separately in YAML document, not in header)

### Message Bus Interface

**Reference:** `internal/messagebus/messagebus.go:40-46`

```go
type MessageBus struct {
    path         string              // Path to messagebus.yaml file
    now          func() time.Time    // Clock (injectable for testing)
    lockTimeout  time.Duration       // Exclusive lock timeout (default: 10s)
    pollInterval time.Duration       // Poll interval for PollForNew (default: 200ms)
}
```

**Core Operations:**

```go
// Create a new message bus
func NewMessageBus(path string, opts ...Option) (*MessageBus, error)

// Append a message and return its msg_id
func (mb *MessageBus) AppendMessage(msg *Message) (string, error)

// Read messages after sinceID (lockless)
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error)

// Block until new messages appear (polling)
func (mb *MessageBus) PollForNew(lastID string) ([]*Message, error)
```

---

## O_APPEND + flock Design

The message bus uses a **two-layer concurrency control** mechanism:

1. **O_APPEND**: Kernel-level atomic appends (Unix guarantee)
2. **flock**: Advisory exclusive lock for write serialization

### Why O_APPEND?

On Unix systems, the `O_APPEND` flag provides a **kernel-level guarantee** that writes are atomic and appended to the end of the file, even with concurrent writers.

**Reference:** `internal/messagebus/messagebus.go:132`

```go
file, err := os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
```

**POSIX Guarantee (IEEE Std 1003.1):**
> "If the O_APPEND flag of the file status flags is set, the file offset shall be set to the end of the file prior to each write and no intervening file modification operation shall occur between changing the file offset and the write operation."

**What this means:**
- Each `write()` syscall is atomic
- Multiple concurrent appends won't interleave
- No "torn writes" (partial data from multiple writers)

**Limitations:**
- Only guaranteed on **local filesystems** (not NFS, SMB, etc.)
- Only guaranteed on **Unix/Linux/macOS** (Windows has weaker guarantees)
- Does not prevent message interleaving (hence need for flock)

### Why flock?

While `O_APPEND` prevents torn writes, it doesn't prevent **message interleaving**. Consider two concurrent appends:

```
Writer A: write("---\nmsg_id: 1\n")
Writer B: write("---\nmsg_id: 2\n")
Writer A: write("body: A\n")
Writer B: write("body: B\n")
```

**Without flock, file could become:**
```yaml
---
msg_id: 1
---
msg_id: 2
body: A
body: B
```

This **corrupts the message format** (message 1 has no body separator).

**Solution: Exclusive lock during entire message write**

**Reference:** `internal/messagebus/messagebus.go:138-143`

```go
if err := LockExclusive(file, mb.lockTimeout); err != nil {
    return "", fmt.Errorf("lock message bus: %w", err)
}
defer func() {
    _ = Unlock(file)
}()
```

**flock characteristics:**
- **Advisory lock**: Processes must cooperate (not enforced by kernel)
- **Per-process**: Lock held by process, released on file close
- **Non-blocking mode**: `LOCK_NB` flag with retry loop
- **Timeout**: Exponential backoff up to `lockTimeout` (default: 10 seconds)

### Write Flow with O_APPEND + flock

```
1. Open file with O_APPEND
2. Acquire exclusive lock (flock with timeout)
3. Write message (header + body separator) to OS page cache
4. Release lock
5. Close file
```

**Key invariant:** Only one writer can hold the lock at a time, ensuring messages are written atomically.

---

## Message ID Generation

Message IDs are **unique, lexically sortable, and human-readable** identifiers.

### Format Specification

**Reference:** `internal/messagebus/msgid.go:13-18`

```go
func GenerateMessageID() string {
    now := time.Now().UTC()
    seq := atomic.AddUint32(&msgSequence, 1) % 10000
    pid := os.Getpid() % 100000
    return fmt.Sprintf("MSG-%s-%09d-PID%05d-%04d",
        now.Format("20060102-150405"),
        now.Nanosecond(),
        pid,
        seq)
}
```

**Format:** `MSG-{YYYYMMDD-HHMMSS}-{NANOSECONDS}-PID{PID}-{SEQUENCE}`

**Example:** `MSG-20260205-143052-000123456-PID12345-0042`

**Components:**

1. **Prefix**: `MSG-` (fixed identifier)
2. **Date-Time**: `YYYYMMDD-HHMMSS` (e.g., `20260205-143052`)
   - ISO 8601 compact format
   - UTC timezone
   - Lexically sortable
3. **Nanoseconds**: `000000000-999999999` (9 digits)
   - Sub-second precision
   - Zero-padded for sorting
4. **PID**: `PID00000-PID99999` (5 digits)
   - Process ID modulo 100000
   - Distinguishes messages from different processes
5. **Sequence**: `0000-9999` (4 digits)
   - Atomic counter per process
   - Wraps at 10000
   - Distinguishes messages within same nanosecond

### Uniqueness Guarantees

**Global Uniqueness:** Combination of `(timestamp, nanosecond, PID, sequence)` is globally unique.

**Analysis:**

1. **Different seconds**: Sorted by date-time component
2. **Same second, different nanoseconds**: Sorted by nanosecond component
3. **Same nanosecond, different processes**: Distinguished by PID
4. **Same nanosecond, same process**: Distinguished by atomic sequence counter

**Edge Cases:**

- **Clock skew**: If system clock moves backward, messages may appear out of order
- **PID reuse**: After PID wraps (unlikely in single session), combined with timestamp makes collision extremely unlikely
- **Sequence wrap**: At 10,000 messages per nanosecond, wrap is possible but timestamp will have advanced
- **Concurrent appends**: Sequence counter is atomic (`atomic.AddUint32`)

### Lexical Sorting

Message IDs are **lexically sortable** as strings, which enables:
- Efficient range queries (`since_id` parameter)
- Chronological ordering without parsing
- Simple string comparison for filtering

**Example ordering:**
```
MSG-20260205-143052-000123456-PID12345-0001
MSG-20260205-143052-000123457-PID12345-0002
MSG-20260205-143052-000123457-PID12346-0001
MSG-20260205-143053-000000001-PID12345-0003
```

### Human Readability

The timestamp component is human-readable without parsing:
- `20260205` → February 5, 2026
- `143052` → 14:30:52 (2:30:52 PM)

This simplifies debugging and log analysis.

---

## Concurrency Guarantees

The message bus provides **strong concurrency guarantees** for both reads and writes.

### Lockless Reads

**Key Property:** Readers never acquire locks.

**Reference:** `internal/messagebus/messagebus.go:155-174`

```go
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error) {
    // ... validation ...
    data, err := os.ReadFile(mb.path)  // No lock!
    if err != nil {
        if os.IsNotExist(err) {
            return []*Message{}, nil
        }
        return nil, errors.Wrap(err, "read message bus")
    }
    messages, err := parseMessages(data)
    if err != nil {
        return nil, err
    }
    return filterSince(messages, sinceID)
}
```

**Why lockless reads work:**

1. **Append-only**: Existing messages never change
2. **O_APPEND atomicity**: New messages are appended atomically
3. **Readers see consistent prefix**: Readers may see partial file, but all complete messages are valid

**Consistency model:**

- **Read uncommitted**: Readers may see messages before `fsync()` completes
- **Prefix consistency**: Readers always see a valid prefix of the log
- **No torn reads**: YAML parser handles incomplete trailing messages gracefully

**Benefits:**

- High read throughput (no lock contention)
- Multiple concurrent readers
- No reader starvation
- Real-time monitoring without blocking writes

### Exclusive Writes

**Key Property:** Only one writer at a time.

**Reference:** `internal/messagebus/lock.go:16-27`

```go
func LockExclusive(file *os.File, timeout time.Duration) error {
    return flockExclusive(file, timeout)
}

func flockExclusive(file *os.File, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for {
        locked, err := tryFlockExclusive(file)  // LOCK_EX | LOCK_NB
        if err != nil {
            return errors.Wrap(err, "flock")
        }
        if locked {
            return nil
        }
        if time.Now().After(deadline) {
            return ErrLockTimeout
        }
        time.Sleep(10 * time.Millisecond)  // Exponential backoff
    }
}
```

**Unix implementation (flock):**

**Reference:** `internal/messagebus/lock_unix.go:10-19`

```go
func tryFlockExclusive(file *os.File) (bool, error) {
    err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err == nil {
        return true, nil  // Lock acquired
    }
    if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
        return false, nil  // Lock held by another process, retry
    }
    return false, err  // Actual error
}
```

**Lock timeout mechanism:**

1. Try to acquire lock (non-blocking)
2. If held by another process: sleep 10ms and retry
3. Repeat until lock acquired or timeout (default: 10 seconds)
4. Return `ErrLockTimeout` if timeout exceeded

**Guarantees:**

- **Mutual exclusion**: Only one writer at a time
- **Progress**: Writer eventually acquires lock (no deadlock)
- **Fairness**: Not guaranteed (first-come may not be first-served)

### Write Ordering

Messages are **totally ordered** by the order in which writes complete.

**Happens-before relationship:**

```
Write1.lock → Write1.write → Write1.unlock
  ↓
  happens-before
  ↓
Write2.lock → Write2.write → Write2.unlock
```

**Guarantee:** If `Write1` completes before `Write2` starts, `Message1` appears before `Message2` in the log.

### Race-Free Read-After-Write

**Scenario:** Writer appends message, then immediately reads.

```go
msgID, err := bus.AppendMessage(msg)    // Write
messages, err := bus.ReadMessages("")    // Read
```

**Question:** Is the new message guaranteed to be visible?

**Answer:** **Yes**, because:

1. `AppendMessage()` writes to the OS page cache while holding `flock`
2. OS page cache is shared across all processes on the same host
3. `ReadMessages()` reads from the same OS page cache
4. Read-after-write consistency is guaranteed within a single host

**Exception:** Network filesystems (NFS, SMB) may have weaker consistency. Use local storage for message bus.

---

## Write Durability Model

Writes go to the OS page cache and are **not fsynced**. This is an intentional performance trade-off.

### Design Decision: No fsync

**Problem:** `fsync()` serializes writes to disk, limiting throughput to ~50–1000 writes/sec on typical hardware.

**Solution:** Skip `fsync()`. Writes are held in the OS page cache and flushed asynchronously. The OS guarantees that all processes reading the same file path see the written data immediately (read-after-write consistency within the same host).

**Write sequence:**

```
1. Open file (O_WRONLY|O_APPEND|O_CREATE)
2. flock() — acquire exclusive lock
3. Write data (to OS page cache)
4. Unlock
5. Return success
```

**Throughput:** ~37,000+ messages/sec measured with 10 concurrent writers on macOS.

### Consistency Guarantees

- **Visibility**: Other processes reading the same file will see new messages immediately (OS page cache is shared).
- **Ordering**: `flock` ensures messages are appended in a serialized, total order.
- **Durability**: Messages survive process crashes but **may be lost** on OS crash or power failure before the page cache is flushed (typically within seconds).

### When Does Data Reach Disk?

The OS flushes dirty pages on a configurable schedule (typically every 5–30 seconds) or when memory pressure triggers a flush. For the message bus use case (agent coordination and logging), this is acceptable — transient message loss on hard crash is recoverable since agents can re-post their state.

### Network Filesystem Warning

On NFS, SMB, or other network filesystems, read-after-write consistency is **not guaranteed** without additional synchronization. Use local storage for the message bus.

---

## Message Format

Messages are stored in **YAML with document separators** (`---`).

### File Format Specification

**Reference:** `internal/messagebus/messagebus.go:229-251`

```go
func serializeMessage(msg *Message) ([]byte, error) {
    header, err := yaml.Marshal(msg)
    if err != nil {
        return nil, errors.Wrap(err, "marshal message")
    }
    var buf bytes.Buffer
    buf.WriteString("---\n")         // Header separator
    buf.Write(header)                 // YAML header (metadata)
    if len(header) == 0 || header[len(header)-1] != '\n' {
        buf.WriteByte('\n')
    }
    buf.WriteString("---\n")         // Body separator
    if msg.Body != "" {
        buf.WriteString(msg.Body)     // Body content
    }
    if !strings.HasSuffix(msg.Body, "\n") {
        buf.WriteByte('\n')
    }
    return buf.Bytes(), nil
}
```

### Example Message File

```yaml
---
msg_id: MSG-20260205-143052-000123456-PID12345-0001
ts: 2026-02-05T14:30:52.123456Z
type: agent_started
project_id: my-project
task_id: task-001
run_id: run-001
---
Agent started successfully

---
msg_id: MSG-20260205-143053-000234567-PID12345-0002
ts: 2026-02-05T14:30:53.234567Z
type: agent_output
project_id: my-project
task_id: task-001
run_id: run-001
parents:
  - MSG-20260205-143052-000123456-PID12345-0001
---
Processing request: implement new feature

---
msg_id: MSG-20260205-143055-000345678-PID12345-0003
ts: 2026-02-05T14:30:55.345678Z
type: agent_completed
project_id: my-project
task_id: task-001
run_id: run-001
parents:
  - MSG-20260205-143053-000234567-PID12345-0002
---
Task completed successfully
```

### Document Structure

Each message consists of **two YAML documents**:

1. **Header document**: Metadata (msg_id, timestamp, type, etc.)
2. **Body document**: Message content (arbitrary text)

**Separator:** `---` (YAML document separator)

### Why Two Documents?

**Design rationale:**

1. **Metadata separation**: Header is structured (YAML object), body is unstructured (text)
2. **Body flexibility**: Body can contain arbitrary text (including YAML-like content) without escaping
3. **Partial parsing**: Can parse headers without parsing bodies (efficiency)
4. **Human readability**: Clear visual separation

### Field Serialization

**YAML tags in Message struct:**

```go
type Message struct {
    MsgID        string    `yaml:"msg_id"`
    Timestamp    time.Time `yaml:"ts"`
    Type         string    `yaml:"type"`
    ProjectID    string    `yaml:"project_id"`
    TaskID       string    `yaml:"task_id"`
    RunID        string    `yaml:"run_id"`
    ParentMsgIDs []string  `yaml:"parents,omitempty"`  // Omitted if empty
    Attachment   string    `yaml:"attachment_path,omitempty"`  // Omitted if empty
    Body         string    `yaml:"-"`  // Not serialized in header
}
```

**Field naming:**
- `msg_id` (not `MsgID`) - snake_case for consistency with other YAML files
- `ts` (not `Timestamp`) - short name for brevity
- `parents` (not `ParentMsgIDs`) - shorter field name

**Omitempty:** Fields with `omitempty` tag are omitted if empty/zero value.

---

## Read Operations

Read operations are **lockless** and support filtering.

### ReadMessages Flow

**Reference:** `internal/messagebus/messagebus.go:155-174`

```
1. Validate message bus path
2. ReadFile (lockless - no flock!)
3. Parse YAML documents (state machine)
4. Filter messages after sinceID
5. Return message slice
```

**Code:**

```go
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error) {
    // 1. Validation
    if err := validateBusPath(mb.path); err != nil {
        return nil, errors.Wrap(err, "validate message bus path")
    }

    // 2. Read entire file (lockless)
    data, err := os.ReadFile(mb.path)
    if err != nil {
        if os.IsNotExist(err) {
            return []*Message{}, nil  // Empty bus
        }
        return nil, errors.Wrap(err, "read message bus")
    }

    // 3. Parse messages
    messages, err := parseMessages(data)
    if err != nil {
        return nil, err
    }

    // 4. Filter by sinceID
    return filterSince(messages, sinceID)
}
```

### Message Parsing State Machine

**Reference:** `internal/messagebus/messagebus.go:253-321`

**States:**
1. `stateSeekHeader` - Looking for header separator (`---`)
2. `stateHeader` - Reading YAML header
3. `stateBody` - Reading body content

**State transitions:**

```
stateSeekHeader → (see "---") → stateHeader
stateHeader     → (see "---") → stateBody
stateBody       → (see "---") → stateHeader (next message)
stateBody       → (EOF)       → append message and done
```

**Parsing algorithm:**

```go
func parseMessages(data []byte) ([]*Message, error) {
    reader := bufio.NewReader(bytes.NewReader(data))
    state := stateSeekHeader
    var headerBuf, bodyBuf bytes.Buffer
    var current *Message
    messages := make([]*Message, 0)

    for {
        line, err := reader.ReadString('\n')
        if err != nil && err != io.EOF {
            return nil, errors.Wrap(err, "read message bus")
        }
        if err == io.EOF && line == "" {
            break
        }

        trimmed := strings.TrimRight(line, "\r\n")

        switch state {
        case stateSeekHeader:
            if trimmed == "---" {
                state = stateHeader
                headerBuf.Reset()
            }

        case stateHeader:
            if trimmed == "---" {
                // Parse header
                var msg Message
                if err := yaml.Unmarshal(headerBuf.Bytes(), &msg); err != nil {
                    // Invalid header, skip and continue
                    headerBuf.Reset()
                    state = stateHeader
                    break
                }
                current = &msg
                bodyBuf.Reset()
                state = stateBody
            } else {
                headerBuf.WriteString(line)
            }

        case stateBody:
            if trimmed == "---" {
                // End of body, append message
                if current != nil {
                    current.Body = finalizeBody(bodyBuf.Bytes())
                    messages = append(messages, current)
                }
                headerBuf.Reset()
                state = stateHeader
            } else {
                bodyBuf.WriteString(line)
            }
        }

        if err == io.EOF {
            break
        }
    }

    // Handle final message (no trailing separator)
    if state == stateBody && current != nil {
        bodyBytes := bodyBuf.Bytes()
        if len(bodyBytes) > 0 && bodyBytes[len(bodyBytes)-1] == '\n' {
            current.Body = finalizeBody(bodyBytes)
            messages = append(messages, current)
        }
    }

    return messages, nil
}
```

**Resilience:** Parser handles malformed messages gracefully:
- Invalid YAML header → skip and continue parsing
- Missing body separator → treat as partial message (ignore)
- Extra separators → handled as state transitions

### Filtering by sinceID

**Reference:** `internal/messagebus/messagebus.go:333-346`

```go
func filterSince(messages []*Message, sinceID string) ([]*Message, error) {
    if strings.TrimSpace(sinceID) == "" {
        return messages, nil  // No filter, return all
    }

    // Linear search for sinceID
    for i, msg := range messages {
        if msg != nil && msg.MsgID == sinceID {
            if i+1 >= len(messages) {
                return []*Message{}, nil  // No messages after sinceID
            }
            return messages[i+1:], nil  // Return messages after sinceID
        }
    }

    // sinceID not found
    return nil, fmt.Errorf("since id %q not found: %w", sinceID, ErrSinceIDNotFound)
}
```

**Behavior:**

- `sinceID = ""` → Return all messages
- `sinceID = "MSG-..."` → Return messages **after** (not including) sinceID
- `sinceID` not found → Return `ErrSinceIDNotFound` error

**Use case:** Server-Sent Events (SSE) streaming with `Last-Event-ID` header.

### PollForNew

**Reference:** `internal/messagebus/messagebus.go:176-191`

```go
func (mb *MessageBus) PollForNew(lastID string) ([]*Message, error) {
    for {
        messages, err := mb.ReadMessages(lastID)
        if err != nil {
            return nil, err
        }
        if len(messages) > 0 {
            return messages, nil  // New messages found
        }
        time.Sleep(mb.pollInterval)  // Poll interval (default: 200ms)
    }
}
```

**Behavior:**

1. Call `ReadMessages(lastID)`
2. If new messages found: return immediately
3. If no new messages: sleep for `pollInterval` (default: 200ms)
4. Repeat until new messages appear

**Use case:** SSE streaming endpoint blocks on `PollForNew()` to wait for new events.

**Note:** This is a **busy-wait polling** approach, not event-driven (no inotify/fsnotify).

---

## Write Operations

Write operations acquire an **exclusive lock** for the entire write.

### AppendMessage Flow

**Reference:** `internal/messagebus/messagebus.go:102-152`

```
1. Validate message (type, projectID required)
2. Generate unique MsgID
3. Set timestamp (UTC)
4. Serialize to YAML (header + body)
5. Validate bus path (no symlinks)
6. Open file (O_WRONLY | O_APPEND | O_CREATE)
7. LockExclusive (flock with timeout)
8. Append message data to OS page cache
9. Unlock
10. Close file
11. Return MsgID
```

**Code:**

```go
func (mb *MessageBus) AppendMessage(msg *Message) (string, error) {
    // 1. Validation
    if msg == nil {
        return "", errors.New("message is nil")
    }
    if strings.TrimSpace(msg.Type) == "" {
        return "", errors.New("message type is empty")
    }
    if strings.TrimSpace(msg.ProjectID) == "" {
        return "", errors.New("project id is empty")
    }

    // 2. Generate MsgID
    msg.MsgID = GenerateMessageID()

    // 3. Set timestamp
    if msg.Timestamp.IsZero() {
        msg.Timestamp = mb.now().UTC()
    } else {
        msg.Timestamp = msg.Timestamp.UTC()
    }

    // 4. Serialize
    data, err := serializeMessage(msg)
    if err != nil {
        return "", errors.Wrap(err, "serialize message")
    }

    // 5. Validate path
    if err := validateBusPath(mb.path); err != nil {
        return "", errors.Wrap(err, "validate message bus path")
    }

    // 6. Open file
    file, err := os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
    if err != nil {
        return "", errors.Wrap(err, "open message bus")
    }
    defer file.Close()

    // 7. Lock
    if err := LockExclusive(file, mb.lockTimeout); err != nil {
        return "", fmt.Errorf("lock message bus: %w", err)
    }
    defer func() {
        _ = Unlock(file)
    }()

    // 8. Append (to OS page cache)
    if err := appendEntry(file, data); err != nil {
        return "", errors.Wrap(err, "write message")
    }

    // 9-10. Unlock and close (deferred)

    // 11. Return MsgID
    return msg.MsgID, nil
}
```

### Append Entry Logic

**Reference:** `internal/messagebus/messagebus.go:193-213`

```go
func appendEntry(file *os.File, data []byte) error {
    // Check if file is empty
    info, err := file.Stat()
    if err != nil {
        return errors.Wrap(err, "stat message bus")
    }

    // Add separator newline if file is non-empty
    if info.Size() > 0 {
        if _, err := file.Write([]byte("\n")); err != nil {
            return errors.Wrap(err, "write separator")
        }
    }

    // Write message data
    if err := writeAll(file, data); err != nil {
        return err
    }

    return nil
}
```

**Separator logic:**
- First message: No leading newline
- Subsequent messages: Leading newline to separate from previous message

**Result:** Messages separated by single blank line.

### Atomic Write Helper

**Reference:** `internal/messagebus/messagebus.go:215-227`

```go
func writeAll(w io.Writer, data []byte) error {
    for len(data) > 0 {
        n, err := w.Write(data)
        if err != nil {
            return errors.Wrap(err, "write message data")
        }
        if n == 0 {
            return errors.New("short write")
        }
        data = data[n:]  // Advance by bytes written
    }
    return nil
}
```

**Purpose:** Handle short writes (partial writes) by retrying until all data is written.

**Note:** With `O_APPEND`, writes should be atomic up to `PIPE_BUF` (4096 bytes on Linux), but this provides extra safety.

---

## Race Condition Handling

The message bus handles several potential race conditions.

### Race 1: Concurrent Appends

**Scenario:** Two writers append messages concurrently.

```
Writer A: AppendMessage(msg1)
Writer B: AppendMessage(msg2)
```

**Solution:** Exclusive lock (`flock`) ensures serial execution.

```
Timeline:
t0: A locks file
t1: A writes msg1 (to OS page cache)
t2: A unlocks file
t3: B locks file (was blocked at t0)
t4: B writes msg2 (to OS page cache)
t5: B unlocks file
```

**Result:** `msg1` appears before `msg2` in file (total order).

### Race 2: Read During Write

**Scenario:** Reader reads while writer is appending.

```
Writer: AppendMessage(msg)
Reader: ReadMessages("")
```

**Possible interleavings:**

**Case 1: Reader before write**
```
Reader: read() → sees old data
Writer: lock → write → fsync → unlock
```
Result: Reader sees old messages (consistent prefix).

**Case 2: Reader during write**
```
Writer: lock → write (in progress)
Reader: read() → sees old data + partial new message
```
Result: Parser handles incomplete message gracefully (ignores partial message).

**Case 3: Reader after write**
```
Writer: lock → write → fsync → unlock
Reader: read() → sees old + new data
```
Result: Reader sees all messages including new one.

**Key insight:** Readers never see corrupted data due to:
1. **O_APPEND atomicity**: Writes are atomic at kernel level
2. **Parser resilience**: Incomplete messages are ignored
3. **Append-only**: Old messages never change

### Race 3: Delete During Read

**Scenario:** File is deleted while reader is reading.

```
Reader: open() → read()
Writer: delete file
Reader: read() (continued)
```

**Unix behavior:**
- **File descriptor remains valid** after deletion (inode still exists)
- Reader can continue reading until close()
- File is actually deleted when last fd is closed

**Result:** Reader successfully reads old data, but file is gone afterward.

**Mitigation:** Don't delete message bus files (append-only design).

### Race 4: Lock Timeout

**Scenario:** Writer holds lock for > 10 seconds (default timeout).

```
Writer A: lock → write (very slow)
Writer B: lock (retry) → lock (retry) → timeout → error
```

**Result:** Writer B returns `ErrLockTimeout` error.

**Mitigation:** Configure longer timeout if needed:

```go
bus, err := NewMessageBus(path, WithLockTimeout(30 * time.Second))
```

---

## Platform Differences

The message bus has **platform-specific** implementations for locking.

### Unix/Linux/macOS (flock)

**Reference:** `internal/messagebus/lock_unix.go`

**Build constraint:** `//go:build !windows`

```go
func tryFlockExclusive(file *os.File) (bool, error) {
    err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err == nil {
        return true, nil
    }
    if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
        return false, nil  // Lock held, retry
    }
    return false, err
}

func unlockFile(file *os.File) error {
    return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
```

**flock characteristics:**

- **Advisory lock**: Not enforced by kernel (processes must cooperate)
- **Per-file descriptor**: Lock released when fd is closed
- **Inheritance**: Not inherited across `fork()` or `exec()`
- **Network filesystems**: May not work on NFS (depends on NFS version)

**Flags:**
- `LOCK_EX`: Exclusive lock (mutual exclusion)
- `LOCK_NB`: Non-blocking (return immediately if lock held)
- `LOCK_UN`: Unlock

**Error codes:**
- `EWOULDBLOCK`: Lock held by another process (expected on contention)
- `EAGAIN`: Same as `EWOULDBLOCK` on some systems
- Other errors: Unexpected (I/O error, invalid fd, etc.)

### Windows (LockFileEx)

**Reference:** `internal/messagebus/lock_windows.go`

**Build constraint:** `//go:build windows`

```go
func tryFlockExclusive(file *os.File) (bool, error) {
    handle := syscall.Handle(file.Fd())
    var overlapped syscall.Overlapped
    err := syscall.LockFileEx(handle,
        syscall.LOCKFILE_EXCLUSIVE_LOCK|syscall.LOCKFILE_FAIL_IMMEDIATELY,
        0, 1, 0, &overlapped)
    if err == nil {
        return true, nil
    }
    if err == syscall.ERROR_LOCK_VIOLATION {
        return false, nil  // Lock held, retry
    }
    return false, err
}

func unlockFile(file *os.File) error {
    handle := syscall.Handle(file.Fd())
    var overlapped syscall.Overlapped
    return syscall.UnlockFileEx(handle, 0, 1, 0, &overlapped)
}
```

**LockFileEx characteristics:**

- **Mandatory lock**: Enforced by kernel (blocks all access)
- **Per-file handle**: Lock released when handle is closed
- **Inheritance**: Not inherited across process creation
- **Region locking**: Can lock specific byte ranges (we lock 1 byte at offset 0)

**Flags:**
- `LOCKFILE_EXCLUSIVE_LOCK`: Exclusive lock
- `LOCKFILE_FAIL_IMMEDIATELY`: Non-blocking

**Error codes:**
- `ERROR_LOCK_VIOLATION`: Lock held by another process

### Platform Comparison

| Feature | Unix (flock) | Windows (LockFileEx) |
|---------|--------------|---------------------|
| Lock type | Advisory | Mandatory |
| Readers blocked? | No | Yes |
| Lock scope | Entire file | Byte range (we use 1 byte) |
| Inheritance | Not inherited | Not inherited |
| Network FS | May not work on NFS | Works on SMB |
| Performance | Fast | Fast |

**Key difference:** Windows locks are **mandatory** and **may block readers**!

### Windows Limitation: Reader Blocking

**Problem:** On Windows, `LockFileEx` is a mandatory lock that blocks **all access** to the locked region, including reads.

**Impact on conductor-loop:**
- Readers may be blocked while writer holds lock (~1-5ms)
- Degrades "lockless reads" design goal
- Multiple readers still don't block each other (when no writer)

**Mitigation:**
- Keep write time short (already done)
- Consider using memory-mapped files (more complex)
- Document limitation in Windows support

**Recommendation:** Use **WSL2** on Windows for full Unix semantics.

### O_APPEND on Windows

**POSIX guarantee:** Writes with `O_APPEND` are atomic.

**Windows reality:**
- `O_APPEND` is emulated by Go runtime (not native Win32)
- Atomicity depends on filesystem (NTFS is generally safe)
- Network filesystems may not be atomic

**Recommendation:** Use local NTFS filesystems only.

---

## Performance Characteristics

### Write Performance

**Typical latency:**
- Lock acquisition: 0.1-10ms (depends on contention)
- Serialize message: <0.1ms
- Write to file: <0.1ms (buffered)
- fsync(): 0.1-20ms (dominant cost, depends on storage)
- Unlock: <0.1ms

**Total write latency:** ~1-30ms (dominated by fsync)

**Throughput:**
- **Sequential writes**: 50-1000 messages/sec (limited by fsync)
- **Concurrent writers**: Similar (lock serializes writes)

**Bottlenecks:**
1. **fsync() latency**: Cannot avoid (durability requirement)
2. **Lock contention**: With 10+ concurrent writers, contention increases
3. **Single file**: All projects/tasks share same file (contention)

### Read Performance

**Typical latency:**
- Read entire file: 0.1-10ms (depends on file size)
- Parse YAML: 0.1-5ms (depends on message count)
- Filter: <0.1ms (linear scan)

**Total read latency:** ~0.5-20ms

**Throughput:**
- **Concurrent readers**: 100-1000 reads/sec (no lock contention)
- **Scalability**: Linear with CPU cores (no shared locks)

**Bottlenecks:**
1. **File size**: Larger files = slower reads (linear scan)
2. **Parsing cost**: Proportional to message count
3. **Memory allocation**: Slice allocation for messages

### Scaling Limits

**File size:**
- **Tested**: Up to 100 MB (~100,000 messages)
- **Practical limit**: 1 GB (~1 million messages)
- **Mitigation**: Log rotation (not implemented)

**Concurrent writers:**
- **Tested**: 10 concurrent writers
- **Practical limit**: 50 writers (lock contention becomes significant)
- **Mitigation**: Per-task message buses (already done)

**Concurrent readers:**
- **No limit**: Lockless reads scale linearly with cores

### Performance Optimizations

**Current optimizations:**
1. Lockless reads (no lock on read path)
2. O_APPEND for atomic writes (kernel-level)
3. Per-task message buses (reduce contention)
4. Configurable poll interval (tune latency vs CPU)

**Potential optimizations (not implemented):**
1. **Batch writes**: Multiple messages per fsync (~10x throughput)
2. **Memory-mapped files**: Faster reads, more complex
3. **Index file**: Faster filtering by ID (at cost of complexity)
4. **Log rotation**: Limit file size (need reader cooperation)
5. **Binary format**: Faster parsing than YAML (lose human-readability)

---

## Known Limitations

### 1. File Size Growth

**Issue:** Files grow unbounded (append-only, no deletion).

**Impact:**
- Slower reads (parse entire file)
- Disk space usage
- Eventually hits filesystem limits

**Implemented Mitigations:**
- Per-task message buses (limits individual file size)
- `WithAutoRotate(maxBytes int64)` option: automatically renames the bus file to
  `<path>.YYYYMMDD-HHMMSS.archived` when a write would exceed the threshold;
  the next write creates a fresh file. SSE streaming handles rotation via
  `ErrSinceIDNotFound` reset.
- `run-agent gc --rotate-bus --bus-max-size 10MB --root runs`: rotate all bus
  files exceeding the threshold in a single pass.
- `ReadLastN(n int)` method: efficient tail-only reads using a 64KB seek window
  (doubles up to 3× before falling back to a full read) — avoids loading the
  entire file for small queries.

**Usage:**
```go
// Automatic rotation on write
bus, _ := NewMessageBus(path, WithAutoRotate(10*1024*1024))

// Read last N messages efficiently
messages, _ := bus.ReadLastN(100)
```

### 2. No Complex Queries

**Issue:** Can only filter by `sinceID` (linear scan).

**Missing features:**
- Filter by type
- Filter by time range
- Filter by parent message
- Count messages

**Mitigation:** Filter in application code after reading.

**Example:**
```go
messages, _ := bus.ReadMessages("")
filtered := make([]*Message, 0)
for _, msg := range messages {
    if msg.Type == "agent_output" {
        filtered = append(filtered, msg)
    }
}
```

### 3. Network Filesystems

**Issue:** O_APPEND and flock may not work correctly on network filesystems.

**Affected systems:**
- NFS (especially NFSv2/NFSv3)
- SMB/CIFS
- Distributed filesystems (GlusterFS, etc.)

**Recommendation:** **Always use local filesystems** for message bus.

**Detection:** Not automatic (may silently corrupt data).

### 4. No Transactional Semantics

**Issue:** Cannot atomically append to multiple message buses.

**Scenario:**
```go
// Want atomic: append to both buses or neither
bus1.AppendMessage(msg1)  // May succeed
bus2.AppendMessage(msg2)  // May fail
```

**Mitigation:** Not possible with current design.

**Alternative:** Use single message bus with multiple projects/tasks.

### 5. Lock Timeout Errors

**Issue:** Writers may timeout if lock is held too long.

**Scenario:**
- Slow storage (HDD with many concurrent writes)
- Long-running writers (should never happen, but bugs exist)

**Mitigation:** Increase timeout:
```go
bus, _ := NewMessageBus(path, WithLockTimeout(30 * time.Second))
```

**Trade-off:** Longer timeout = longer wait on stuck writers.

### 6. No Garbage Collection

**Issue:** Deleted/obsolete messages remain in file forever.

**Impact:** Wasted disk space and read time.

**Mitigation:** Manual log rotation (see limitation #1).

### 7. Polling-Based Event Notification

**Issue:** `PollForNew()` uses busy-wait polling (not event-driven).

**Impact:**
- CPU usage (even when idle)
- Latency (poll interval = 200ms by default)

**Alternative:** Use `inotify` (Linux) or `FSEvents` (macOS) for event-driven notification.

**Trade-off:** More complex, platform-specific.

### 8. Windows Reader Blocking

**Issue:** Mandatory locks on Windows may block readers.

**Impact:** Degrades "lockless reads" design goal.

**Mitigation:** Use WSL2 for full Unix semantics.

---

## Implementation Reference

### Message Bus Creation

```go
// Create message bus with default options
bus, err := NewMessageBus("/path/to/messagebus.yaml")

// Create with custom options
bus, err := NewMessageBus("/path/to/messagebus.yaml",
    WithLockTimeout(30 * time.Second),
    WithPollInterval(100 * time.Millisecond),
    WithClock(time.Now),  // Injectable clock for testing
)
```

### Appending Messages

```go
msg := &Message{
    Type:      "agent_started",
    ProjectID: "my-project",
    TaskID:    "task-001",
    RunID:     "run-001",
    Body:      "Agent started successfully",
}

msgID, err := bus.AppendMessage(msg)
if err != nil {
    // Handle error (lock timeout, I/O error, etc.)
}
// msgID: MSG-20260205-143052-000123456-PID12345-0001
```

### Reading Messages

```go
// Read all messages
messages, err := bus.ReadMessages("")

// Read messages after specific ID
messages, err := bus.ReadMessages("MSG-20260205-143052-000123456-PID12345-0001")

// Check for "since ID not found" error
if errors.Is(err, ErrSinceIDNotFound) {
    // Handle missing ID (client out of sync)
}
```

### Polling for New Messages

```go
lastID := ""
for {
    messages, err := bus.PollForNew(lastID)
    if err != nil {
        // Handle error
        break
    }

    for _, msg := range messages {
        // Process new message
        fmt.Printf("New message: %s\n", msg.Body)
        lastID = msg.MsgID
    }
}
```

### Message Threading (Parent-Child)

```go
// Parent message
parentID, _ := bus.AppendMessage(&Message{
    Type:      "agent_query",
    ProjectID: "my-project",
    Body:      "Please implement feature X",
})

// Child message (response)
_, _ = bus.AppendMessage(&Message{
    Type:         "agent_response",
    ProjectID:    "my-project",
    ParentMsgIDs: []string{parentID},
    Body:         "Feature X implemented successfully",
})
```

### Testing with Mock Clock

```go
// Inject mock clock for deterministic timestamps
var now time.Time
bus, _ := NewMessageBus(path, WithClock(func() time.Time {
    return now
}))

now = time.Date(2026, 2, 5, 14, 30, 0, 0, time.UTC)
msgID1, _ := bus.AppendMessage(msg1)

now = now.Add(1 * time.Second)
msgID2, _ := bus.AppendMessage(msg2)

// msgID1 and msgID2 have predictable timestamps
```

### Error Handling

```go
msgID, err := bus.AppendMessage(msg)
if err != nil {
    switch {
    case errors.Is(err, ErrLockTimeout):
        // Lock timeout (10 seconds elapsed)
        // Retry or return error to client

    case strings.Contains(err.Error(), "symlink"):
        // Path is a symlink (security check failed)
        // Fix configuration

    case strings.Contains(err.Error(), "fsync"):
        // Disk error (out of space, I/O error)
        // Check disk health

    default:
        // Other error (validation, I/O, etc.)
    }
}
```

---

## Testing Considerations

### Unit Tests

**Reference:** `internal/messagebus/messagebus_test.go`

Key test cases:
- Validation (nil bus, nil message, empty fields)
- Append and read single message
- Append and read multiple messages
- Read with `sinceID` filtering
- `ErrSinceIDNotFound` handling
- Concurrent appends (race detector)
- Concurrent reads (race detector)
- Lock timeout
- Symlink rejection
- Message ID uniqueness
- Polling behavior

### Integration Tests

Test with real filesystem:
- Multiple concurrent writers
- Multiple concurrent readers
- Large files (100 MB+)
- Cross-platform (Unix + Windows)

### Performance Tests

Benchmark:
- Write throughput (messages/sec)
- Read throughput (messages/sec)
- Latency distribution (p50, p95, p99)
- Lock contention with N writers

---

## Summary

The message bus provides a **simple, durable, and debuggable** append-only log for multi-agent coordination.

**Key strengths:**
- Strong durability (fsync on every write)
- Lockless reads (high read throughput)
- Human-readable format (YAML)
- No external dependencies (filesystem only)
- Platform-specific optimizations

**Key trade-offs:**
- File size grows unbounded (need rotation)
- Write throughput limited by fsync (~1000 msg/sec)
- No complex queries (linear scan only)
- Polling-based event notification (not event-driven)

**Design philosophy:** Choose **simplicity** and **correctness** over **maximum performance**.

---

**Last Updated:** 2026-02-21
**Version:** 1.0.0
