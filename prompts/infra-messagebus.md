# Task: Implement Message Bus

**Task ID**: infra-messagebus
**Phase**: Core Infrastructure
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement O_APPEND + flock message bus with fsync-always policy.

## Specifications
Read:
- docs/specifications/subsystem-message-bus-tools.md
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md (Problem #1)

## Required Implementation

### 1. Package Structure
Location: `internal/messagebus/`
Files:
- messagebus.go - Message struct and operations
- msgid.go - Message ID generation
- lock.go - File locking primitives

### 2. Message Struct
```go
type Message struct {
    MsgID        string            `yaml:"msg_id"`
    Timestamp    time.Time         `yaml:"ts"`
    Type         string            `yaml:"type"` // FACT, PROGRESS, DECISION, REVIEW, ERROR
    ProjectID    string            `yaml:"project_id"`
    TaskID       string            `yaml:"task_id"`
    RunID        string            `yaml:"run_id"`
    ParentMsgIDs []string          `yaml:"parents,omitempty"`
    Attachment   string            `yaml:"attachment_path,omitempty"`
    Body         string            `yaml:"-"` // After second ---
}
```

### 3. Message ID Generation
Format: `MSG-YYYYMMDD-HHMMSS-NNNNNNNNN-PIDXXXXX-SSSS`
- Nanosecond timestamp
- PID (5 digits)
- Atomic counter (4 digits, per-process)

### 4. Write Algorithm
```go
func (mb *MessageBus) AppendMessage(msg *Message) (string, error) {
    // 1. Generate msg_id
    msg.MsgID = generateMessageID()

    // 2. Serialize to YAML with --- delimiters
    data := serializeMessage(msg)

    // 3. Open with O_APPEND
    fd, err := os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

    // 4. Acquire exclusive lock (10s timeout)
    err = flockExclusive(fd, 10*time.Second)

    // 5. Write
    n, err := fd.Write(data)

    // 6. fsync
    err = fd.Sync()

    // 7. Unlock (defer)
    return msg.MsgID, nil
}
```

### 5. Read Operations
```go
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error)
func (mb *MessageBus) PollForNew(lastID string) ([]*Message, error)
```

## Tests Required
Location: `test/integration/messagebus_test.go`
- TestMessageIDUniqueness (1000 IDs)
- TestConcurrentAppend (10 processes × 100 messages = 1000 total)
- TestLockTimeout
- TestCrashRecovery (SIGKILL during write)
- TestReadWhileWriting

## Success Criteria
- All 1000 messages present (no data loss)
- All msg_ids unique
- Lock timeout works
- IntelliJ MCP review clean
- Race detector clean

## References
- THE_PROMPT_v5.md: Stage 7 (Re-run tests in IntelliJ MCP)

## Output
Log to MESSAGE-BUS.md:
- FACT: Message bus implemented
- FACT: Concurrency tests pass (1000/1000 messages)
- FACT: Zero data loss verified
