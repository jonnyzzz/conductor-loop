# Task: Add Max Concurrent Runs Semaphore to Runner

## Context

The conductor-loop runner has no limit on concurrent runs. Multiple tasks can spawn unlimited
agent processes simultaneously, potentially exhausting system resources. The config has no
`max_concurrent_runs` field.

## Goal

Add a configurable `max_concurrent_runs` limit that uses a Go channel semaphore to queue excess
runs. When at capacity, new runs block (waiting for a slot) rather than being rejected.

## Required Reading (read ALL before making any changes)

1. /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — commit format, code style
2. /Users/jonnyzzz/Work/conductor-loop/internal/config/config.go — Config structure, DefaultConfig
3. /Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go — Orchestrator, RunJob, RunTask
4. /Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go — runJob internals
5. /Users/jonnyzzz/Work/conductor-loop/internal/metrics/metrics.go — metrics Registry (add new metric)
6. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go — health endpoint (add queue_depth)
7. /Users/jonnyzzz/Work/conductor-loop/docs/user/configuration.md — config docs to update
8. /Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md — CLI docs to update

## Implementation Steps

### Step 1: Config field (`internal/config/config.go`)

Add `MaxConcurrentRuns int` to `DefaultConfig`:

```go
type DefaultConfig struct {
    Agent             string `yaml:"agent"`
    Timeout           int    `yaml:"timeout"`
    MaxConcurrentRuns int    `yaml:"max_concurrent_runs"`
}
```

Default is 0 (unlimited — existing behavior). Also add HCL parsing in `parseHCLConfig`
under the `defaults` block:
```go
if n, ok := m["max_concurrent_runs"].(int); ok {
    cfg.Defaults.MaxConcurrentRuns = n
}
```

### Step 2: Semaphore in Orchestrator (`internal/runner/orchestrator.go`)

Add a `sem chan struct{}` field to the `Orchestrator` struct (or wherever RunJob/RunTask live).
Read the existing code carefully before modifying — understand the constructor and struct layout.

Initialize in the constructor (after reading cfg):
```go
var sem chan struct{}
if cfg != nil && cfg.Defaults.MaxConcurrentRuns > 0 {
    sem = make(chan struct{}, cfg.Defaults.MaxConcurrentRuns)
}
```

Add helper methods:
```go
// acquireSem blocks until a run slot is available or ctx is cancelled.
func (o *Orchestrator) acquireSem(ctx context.Context) error {
    if o.sem == nil {
        return nil // unlimited
    }
    select {
    case o.sem <- struct{}{}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// releaseSem releases a run slot.
func (o *Orchestrator) releaseSem() {
    if o.sem != nil {
        <-o.sem
    }
}
```

### Step 3: Wire into RunJob and RunTask

In `RunJob` (and `RunTask` if they're separate entry points), acquire the semaphore early
(after validating options but before creating run directory), release via defer:

```go
if err := o.acquireSem(ctx); err != nil {
    return fmt.Errorf("acquire run slot: %w", err)
}
defer o.releaseSem()
```

Important: make sure the defer fires even on early error returns.

### Step 4: Expose queue depth in metrics

In `internal/metrics/metrics.go`, add a gauge metric `conductor_queued_runs_total` for
how many runs are currently waiting for a slot.

Use an `atomic.Int64` counter:
- Increment BEFORE selecting on semaphore channel
- Decrement AFTER acquiring slot (whether success or ctx cancel)

Expose it in the `/metrics` endpoint output alongside other metrics.

Add a method to the Registry:
```go
// RecordWaitingRun increments/decrements the queued run gauge.
func (r *Registry) RecordWaitingRun(delta int64) {
    r.queuedRuns.Add(delta)
}
```

### Step 5: Expose in health endpoint (optional enhancement)

If `/api/v1/health` or `/api/v1/status` returns a JSON body, add a `queued_runs` field.
Check `internal/api/handlers.go` to see the current health response shape and add:
```json
{
  "status": "ok",
  "queued_runs": 0,
  ...
}
```

Only add if the health handler already returns a JSON struct (not a simple `{"status":"ok"}`).

### Step 6: Tests

Write tests (new file or extend existing) for the semaphore behavior:

**Test: unlimited (max=0)**
- Two goroutines call RunJob concurrently; both proceed without blocking
- No semaphore contention

**Test: limited (max=1)**
- Start a mock "long" run that holds the slot
- Second run blocks until first completes
- Verify ordering (first completes before second starts)

**Test: context cancellation while waiting**
- Max=1, slot taken by goroutine A
- Goroutine B tries to acquire with already-cancelled context
- B returns ctx.Err() immediately

**Test: semaphore released on error**
- Trigger an error in RunJob after acquiring semaphore
- Verify semaphore slot is released (can start another run after error)

Tests can use the Orchestrator directly with a minimal mock setup. Look at how existing
`orchestrator_test.go` (if it exists) or `job_test.go` structures tests.

Use `go test -race` to verify no data races.

### Step 7: Documentation

Update `docs/user/configuration.md` under the `defaults:` section:

```yaml
defaults:
  agent: claude
  timeout: 300
  max_concurrent_runs: 4  # 0 = unlimited (default)
```

Explain the queuing behavior: runs wait for a slot rather than being rejected.

## Code Style Requirements

- Channel-based semaphore (idiomatic Go — no sync.Mutex for this)
- No new external dependencies
- All exported symbols have godoc comments
- Table-driven tests where applicable

## Quality Gates (run ALL before committing)

```bash
go build ./...
go test ./internal/runner/ ./internal/config/ ./internal/metrics/
go test -race ./internal/runner/
go vet ./...
```

All must pass. Fix any failures before committing.

## Commit

Single commit with message:
```
feat(runner): add max_concurrent_runs semaphore for run throttling
```

Include in commit body: brief description of the semaphore approach and default behavior.
