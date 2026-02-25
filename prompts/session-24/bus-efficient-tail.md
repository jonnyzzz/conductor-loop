# Task: Message Bus Efficient Tail Reading

## Goal

Add an efficient `ReadLastN(n int) ([]*Message, error)` method to the MessageBus type that reads
the last N messages without loading the entire file into memory. This addresses ISSUE-016
(message bus file size growth / performance degradation).

## Context

Current state:
- `ReadMessages(sinceID string)` loads the entire file with `os.ReadFile()` — slow for large files
- `bus read --tail N` CLI calls ReadMessages then slices last N — still loads everything
- For 100MB+ message bus files, this becomes prohibitively slow

File paths (absolute):
- Message bus implementation: `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go`
- Message bus tests: `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus_test.go`
- Bus CLI command: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/bus.go`
- Issues file: `/Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md`

## Requirements

### 1. Add `ReadLastN(n int) ([]*Message, error)` to MessageBus

Algorithm:
1. Open the file for reading
2. Seek to end, get file size
3. If file size <= 64KB or n <= 0, fall back to `ReadMessages("")` (read all)
4. Otherwise: seek backwards by min(64KB * ceil(n/10), fileSize) from end
5. Read from that position to end of file
6. Parse messages from the chunk (may start mid-message — find first `---\n` separator to sync)
7. If parsed message count >= n, return last N
8. If not enough messages, double the read window and retry (up to 3 doublings)
9. If still not enough after doublings, fall back to `ReadMessages("")`

Message format: each message starts with a `---` separator line. The file format is:
```
---
msg_id: MSG-...
ts: ...
type: ...
...
---
body text here

```

So to find message boundaries: scan for `\n---\n` patterns.

### 2. Update `bus read --tail N` CLI to use ReadLastN

In `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/bus.go`:
- When `tail > 0`, call `mb.ReadLastN(tail)` instead of `ReadMessages("")` + slice
- Keep existing behavior for `tail == 0` (read all messages)

### 3. Add tests for ReadLastN

In `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus_test.go`:
- Test `ReadLastN(0)` → returns all messages (same as ReadMessages(""))
- Test `ReadLastN(N)` where N >= message count → returns all messages
- Test `ReadLastN(N)` where N < message count → returns last N
- Test `ReadLastN(1)` → returns only last message
- Test with empty file → returns empty slice, no error
- Test with file just below 64KB threshold → uses full read
- Optionally: test with a large file (>64KB) to verify seek-based path

### 4. Update ISSUES.md

In ISSUE-016 section, add a note that efficient tail-based reading has been implemented:
```
- [x] Efficient ReadLastN() method for tail-based reads without loading full file (Session #24)
- [x] bus read --tail N uses ReadLastN for O(log n) read complexity (Session #24)
```

## Coding Standards

- Follow Go conventions in the file (errors.Wrap, table-driven tests)
- Keep the implementation in the existing messagebus.go file
- Export the method as `ReadLastN` (public API)
- No external dependencies beyond what's already used

## Quality Gates

Before creating the DONE file:
1. `go build ./...` must pass
2. `go test ./internal/messagebus/...` must pass
3. `go test -race ./internal/messagebus/...` must pass
4. `go test ./cmd/run-agent/...` must pass

## Done

Create the file `/Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/$JRUN_TASK_ID/DONE`
to signal completion of this task.

Also write a brief summary to output.md in $JRUN_RUN_FOLDER describing what was implemented.
