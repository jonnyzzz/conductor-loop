# Task: Add WithFsync Option to Message Bus

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus_test.go
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/questions.md

## Human Decision (from QUESTIONS.md Q1)

> Should message bus writes ever be fsynced?
> **Decision (2026-02-20)**: Add `WithFsync(bool)` option, default false. Current 37K msg/sec performance is
> excellent for the primary use case. For durability-critical deployments, users can enable fsync.

## What to Implement

### 1. Add `WithFsync(enabled bool)` option

In `internal/messagebus/messagebus.go`, add a new option:

```go
// WithFsync enables or disables fsync after each message write.
// Default is false (no fsync) for maximum throughput.
// Enable for durability-critical deployments where message loss on OS crash is unacceptable.
// WARNING: fsync significantly reduces throughput (~200 msg/sec vs 37,000+ without).
func WithFsync(enabled bool) Option
```

### 2. Implementation Details

- Add `fsync bool` field to the MessageBus struct (or options struct)
- In `AppendMessage()`, after the O_APPEND write, if `fsync` is true, call `file.Sync()`
- The `file.Sync()` call should happen BEFORE releasing the lock
- Default is `false` (no fsync) — same behavior as current code

### 3. Tests Required

Add to `internal/messagebus/messagebus_test.go`:
- `TestWithFsyncOption`: Verify that WithFsync(true) option is accepted and stored
- `TestWithFsyncFalseDefault`: Verify default bus has fsync=false
- `TestFsyncWritesComplete`: With WithFsync(true), write 10 messages, verify all are readable (functional test)
- `TestWithFsyncThroughputWarning`: Optional comment in test about expected throughput difference (not a benchmark)

### 4. Config Integration (Optional)

If there is a config struct for the message bus, add a `Fsync bool` field.
However, do NOT break any existing code. This is optional if it fits naturally.

## Current MessageBus Code

The MessageBus is in `internal/messagebus/messagebus.go`. Look at the existing `Option` pattern
to understand how to add the new option correctly.

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./internal/messagebus/...` must pass
3. `go test -race ./internal/messagebus/...` must pass
4. `go vet ./...` must pass

## Output

Write your changes to the files. Then create a file at:
/Users/jonnyzzz/Work/conductor-loop/runs/session8-messagebus-fsync/output.md
with a summary of what was changed.

## Commit

Commit with message format: `feat(messagebus): add WithFsync option, default false`
