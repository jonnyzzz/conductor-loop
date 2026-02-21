# Task: ISSUE-007 — Message Bus Retry with Exponential Backoff

## Objective
Add retry logic with exponential backoff to the message bus `AppendMessage` method to handle lock contention under high concurrency (50+ agents).

## Current State
- File: `internal/messagebus/messagebus.go`
- `AppendMessage()` acquires an exclusive flock and writes. If lock acquisition fails after `lockTimeout` (default 10s), it returns `ErrLockTimeout`.
- No retry mechanism exists — a single lock timeout failure causes a permanent write failure.
- File: `internal/messagebus/lock.go` — `LockExclusive()` polls with 10ms interval until timeout.

## Requirements

### 1. Add Retry Logic to AppendMessage
In `AppendMessage()`, wrap the lock-acquire-write-unlock sequence in a retry loop:
- Default: 3 attempts
- Backoff: exponential (1st attempt immediately, 2nd after `lockTimeout`, 3rd after `2 * lockTimeout`)
- Between retries, close and reopen the file to release any stale state
- On final failure, return an error wrapping `ErrLockTimeout` with retry context

### 2. Add Configuration Options
Add two new `Option` functions:
```go
func WithMaxRetries(n int) Option    // default 3, minimum 1
func WithRetryBackoff(d time.Duration) Option  // base backoff between retries, default 100ms
```

### 3. Add Lock Contention Metrics
Add a simple counter to `MessageBus` tracking:
- Total append attempts
- Total lock contention events (retries triggered)
- Expose via method: `func (mb *MessageBus) ContentionStats() (attempts, retries int64)`
- Use `sync/atomic` for thread safety

### 4. Tests
Add tests in `internal/messagebus/messagebus_test.go`:
- `TestAppendRetryOnLockContention`: simulate lock held, verify retry succeeds after lock released
- `TestAppendExhaustsRetries`: verify all retries fail and error is returned
- `TestContentionStats`: verify counters increment correctly
- `TestWithMaxRetriesOption`: verify option is applied
- `TestWithRetryBackoffOption`: verify option is applied

## Constraints
- Do NOT modify `lock.go` or `lock_unix.go` or `lock_windows.go`
- Do NOT change existing test files in `test/integration/`
- Do NOT add new dependencies
- Keep changes minimal and focused on retry logic
- Maintain backward compatibility (default behavior with 0 options should work as before but now with retries)

## Files to Modify
- `internal/messagebus/messagebus.go` — add retry loop, options, stats
- `internal/messagebus/messagebus_test.go` — add retry tests

## Success Criteria
- `go build ./...` passes
- `go test ./internal/messagebus/ -v -count=1` passes
- `go test -race ./internal/messagebus/ -count=1` passes
