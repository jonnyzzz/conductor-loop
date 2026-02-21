# Task: Fix Flaky TestAPIWithRealBackend Test

## Problem

`TestAPIWithRealBackend` in `test/integration/api_end_to_end_test.go` fails
intermittently (not when run in isolation, but when run with other tests via
`go test ./...`). The error is:

```
--- FAIL: TestAPIWithRealBackend (0.54s)
    testing.go:1369: TempDir RemoveAll cleanup: unlinkat .../001/project/task-api-backend: directory not empty
```

## Root Cause

In `internal/api/handlers.go` line 438-439:
```go
if s.startTasks {
    go s.startTask(req, runDir, runPrompt)
}
```

The task is started in a background goroutine. `startTask` calls `runner.RunTask`
which runs the entire ralph loop (possibly creating message bus entries, run
directories, etc.) and takes some time after the run status is set to `completed`.

The test polls `waitForTaskCompletion()` until `status == StatusCompleted`, then
returns. After the test function returns:
1. Defers run (including `defer ts.Close()`)
2. `t.TempDir()` cleanup runs → removes temp dirs

However, between when `StatusCompleted` is written to `run-info.yaml` and when
the goroutine actually exits, there may still be writes to the run directory
(message bus append, file handles). This causes `os.RemoveAll` to fail with
`ENOTEMPTY` when another goroutine concurrently creates files.

## Fix Required

Add a `sync.WaitGroup` to `Server` to track running task goroutines, and expose
a `WaitForTasks()` method. Then update the integration test to wait for all task
goroutines before the test exits.

### Changes to `internal/api/server.go`

1. Add `taskWg sync.WaitGroup` field to the `Server` struct.
2. Add a `WaitForTasks()` method:
   ```go
   // WaitForTasks waits for all background task goroutines to finish.
   // Call this in tests before temp directory cleanup to avoid races.
   func (s *Server) WaitForTasks() {
       s.taskWg.Wait()
   }
   ```

### Changes to `internal/api/handlers.go`

Wrap the goroutine to track it in the WaitGroup:
```go
if s.startTasks {
    s.taskWg.Add(1)
    go func() {
        defer s.taskWg.Done()
        s.startTask(req, runDir, runPrompt)
    }()
}
```

### Changes to `test/integration/api_end_to_end_test.go`

Register cleanup via `t.Cleanup()` that:
1. Closes the httptest.Server
2. Waits for all task goroutines via `server.WaitForTasks()`

```go
// Register cleanup BEFORE the t.TempDir() calls so cleanup runs before TempDir
// removal. The t.Cleanup functions run in LIFO order after defers.
ts := httptest.NewServer(server.Handler())
t.Cleanup(func() {
    ts.Close()
    server.WaitForTasks()
})
```

Remove the `defer ts.Close()` since the `t.Cleanup()` handles it.

**Important**: The `t.Cleanup()` function registered via `t.Cleanup()` runs
AFTER all `defer` statements AND BEFORE `t.TempDir()` cleanup. Register
`t.Cleanup()` AFTER `t.TempDir()` call so it runs first (Cleanup runs LIFO).

Wait - actually `t.Cleanup()` and `t.TempDir()` both register via `t.Cleanup()`
internally. They run in LIFO order, so if we register our cleanup AFTER the
`t.TempDir()` calls, our cleanup runs BEFORE the TempDir cleanup. That's correct.

So the order of cleanup registration matters:
1. `root := t.TempDir()` → registers TempDir cleanup (will run last)
2. `stubDir := t.TempDir()` → registers TempDir cleanup (will run second-to-last)
3. `t.Cleanup(func() { ts.Close(); server.WaitForTasks() })` → runs FIRST (LIFO)

## Additional: Fix acceptance test similarity

Also check `test/acceptance/acceptance_test.go` `startConductor()` function:
- It already has proper cleanup via `cleanupRuns()` which removes run directories
- But it doesn't call `server.WaitForTasks()` either
- After adding `WaitForTasks()`, also add it to the acceptance test cleanup in
  `startConductor()`. The `conductorHarness` struct should hold a reference to
  the `api.Server` (not just `*httptest.Server`).

Actually, the acceptance test uses `cleanupRuns()` which removes the `runs/`
subdirectory, which is a brute-force approach. It works because by that time the
goroutines have probably already exited. Don't change the acceptance test unless
it's also failing.

## Files to Change

1. `/Users/jonnyzzz/Work/conductor-loop/internal/api/server.go`
2. `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go`
3. `/Users/jonnyzzz/Work/conductor-loop/test/integration/api_end_to_end_test.go`

## Verification

After making changes:
```bash
# Run in a loop to catch flakiness (10 times)
for i in $(seq 1 10); do
    go test -count=1 ./test/integration/ 2>&1 | grep -E "PASS|FAIL|TestAPIWithRealBackend"
done

# Full test suite
go test -count=1 ./...
go test -race ./internal/... ./cmd/...
```

The test should pass consistently (all 10 iterations).

## Working Directory

All file paths are absolute. CWD: `/Users/jonnyzzz/Work/conductor-loop`

## Commit Format

```
fix(api): add WaitGroup to track task goroutines for clean test shutdown

Add taskWg sync.WaitGroup to Server struct and WaitForTasks() method so
integration tests can wait for all background task goroutines to finish
before temp directory cleanup. Fixes flaky TestAPIWithRealBackend.
```
