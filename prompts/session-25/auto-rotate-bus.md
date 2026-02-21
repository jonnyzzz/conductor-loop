# Task: Auto-Rotation for Message Bus (Complete ISSUE-016)

## Context

You are a senior Go developer working on the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

This task implements automatic message bus file rotation to complete ISSUE-016.

## Background

The message bus (`internal/messagebus/messagebus.go`) is an append-only file. Currently:
- Manual rotation: `run-agent gc --rotate-bus --bus-max-size 10MB` renames the file to `.YYYYMMDD-HHMMSS.archived`
- Efficient reads: `ReadLastN(n)` seeks backwards to read last N messages
- No automatic rotation: The bus file grows indefinitely if `gc --rotate-bus` is not run manually

**SSE streaming handles rotation gracefully** (verified): The SSE stream uses `ReadMessages(lastID)`. If the bus file is rotated and `lastID` is not found, it gets `ErrSinceIDNotFound` and resets `lastID = ""`, restarting from the new file's beginning.

## Objective

Add `WithAutoRotate(maxBytes int64)` option to MessageBus that:
1. When `AppendMessage()` is called and the file size would exceed `maxBytes`
2. Automatically renames the current file to `<path>.YYYYMMDD-HHMMSS.archived`
3. Creates a new empty file at `<path>` and writes the new message to it
4. This operation happens inside the exclusive lock, so it's atomic/safe

## Implementation Plan

### 1. Add the Option to MessageBus

In `internal/messagebus/messagebus.go`, add:
```go
// WithAutoRotate configures automatic rotation when the bus file exceeds maxBytes.
// When the file size exceeds maxBytes, it is renamed to <path>.YYYYMMDD-HHMMSS.archived
// and a fresh bus file is started. Rotation occurs atomically inside the write lock.
// Set to 0 to disable (default).
func WithAutoRotate(maxBytes int64) Option {
    return func(mb *MessageBus) {
        mb.autoRotateBytes = maxBytes
    }
}
```

Add `autoRotateBytes int64` field to the `MessageBus` struct.

### 2. Implement Rotation in AppendMessage

Inside `AppendMessage()`, after acquiring the exclusive lock but before writing:
```go
if mb.autoRotateBytes > 0 {
    if fi, err := os.Stat(mb.path); err == nil && fi.Size() >= mb.autoRotateBytes {
        // rotate: rename current file to archive
        archivePath := mb.path + "." + time.Now().UTC().Format("20060102-150405") + ".archived"
        _ = os.Rename(mb.path, archivePath) // best-effort; new file will be created on next write
    }
}
```

The rename happens inside the existing exclusive lock, so it's atomic with respect to other writers.

### 3. Add extractRotateFunc (helper)

Add a private `rotateIfNeeded(path string, maxBytes int64) error` function that:
1. Stats the file
2. If size >= maxBytes, renames to archive path
3. Returns any error

### 4. Tests

In `internal/messagebus/messagebus_test.go`, add:

```go
func TestAutoRotation(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.md")

    // Create bus with 1KB auto-rotate threshold
    bus, err := NewMessageBus(path, WithAutoRotate(1024))
    require.NoError(t, err)

    // Write messages until rotation is triggered
    var rotated bool
    for i := 0; i < 100; i++ {
        msg := &Message{Type: "TEST", Body: strings.Repeat("x", 100)}
        _, err := bus.AppendMessage(msg)
        require.NoError(t, err)

        // Check if archive file appeared
        entries, _ := os.ReadDir(dir)
        for _, e := range entries {
            if strings.HasSuffix(e.Name(), ".archived") {
                rotated = true
            }
        }
        if rotated {
            break
        }
    }

    assert.True(t, rotated, "expected auto-rotation to have occurred")

    // Verify new file is smaller than threshold
    fi, err := os.Stat(path)
    require.NoError(t, err)
    assert.Less(t, fi.Size(), int64(1024), "new bus file should be under threshold")
}

func TestAutoRotationDisabled(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.md")

    // No auto-rotate (default)
    bus, err := NewMessageBus(path)
    require.NoError(t, err)

    // Write many messages
    for i := 0; i < 50; i++ {
        msg := &Message{Type: "TEST", Body: strings.Repeat("x", 100)}
        _, err := bus.AppendMessage(msg)
        require.NoError(t, err)
    }

    // No archive file should exist
    entries, _ := os.ReadDir(dir)
    for _, e := range entries {
        assert.False(t, strings.HasSuffix(e.Name(), ".archived"), "no archive expected")
    }
}
```

### 5. Update ISSUE-016 in ISSUES.md

Add to the Progress section:
```
- [x] `WithAutoRotate(maxBytes int64)` option triggers rotation on write when file exceeds threshold (Session #25)
```

Change status to RESOLVED.

## Files to Modify

1. `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go` — add option + rotation logic
2. `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus_test.go` — add tests
3. `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md` — update ISSUE-016

## Existing Code to Read First

Before implementing, read these files to understand the existing patterns:
- `internal/messagebus/messagebus.go` — full file, understand the MessageBus struct and options pattern
- `internal/messagebus/messagebus_test.go` — understand existing test patterns (TestRetryOnLockContention etc.)
- `cmd/run-agent/gc.go` — look at `rotateBusFile()` function for reference on how rotation is implemented currently

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Build must pass
go build ./...

# Tests must pass including new auto-rotate tests
go test -race ./internal/messagebus/...
go test -race ./...
```

## Commit Format

```
feat(messagebus): add WithAutoRotate option for automatic bus file rotation

When the bus file exceeds the configured threshold, it is renamed to
<path>.YYYYMMDD-HHMMSS.archived and a fresh bus file is started.
Rotation is atomic within the write lock. Closes ISSUE-016.
```

## When Done

Create `DONE` file: `touch $TASK_FOLDER/DONE`
