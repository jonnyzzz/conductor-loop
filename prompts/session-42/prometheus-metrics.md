# Task: Add Prometheus-Compatible Metrics Endpoint

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

This is a Go-based multi-agent orchestration framework with a REST API server.
The task is to add a Prometheus-compatible `/metrics` endpoint for production monitoring.

## Current State

- `go build ./...` passes
- All tests pass (`go test -race ./internal/... ./cmd/...`)
- Binaries: `bin/conductor` and `bin/run-agent`
- API server: `internal/api/server.go` with routes in `internal/api/routes.go`

## Task: Add Prometheus Metrics Endpoint

### Background

The server currently has `/api/v1/status` which returns some stats. We need a proper
Prometheus-compatible `/metrics` endpoint that monitoring tools can scrape.

**Important**: Do NOT add any new external dependencies (no prometheus/client_golang).
Implement using Go's built-in `expvar` package or write the Prometheus text format manually.

### Metrics to Expose

Implement using Prometheus text format (content-type: text/plain; version=0.0.4):

```
# HELP conductor_uptime_seconds Server uptime in seconds
# TYPE conductor_uptime_seconds gauge
conductor_uptime_seconds 42.5

# HELP conductor_active_runs_total Currently active (running) agent runs
# TYPE conductor_active_runs_total gauge
conductor_active_runs_total 3

# HELP conductor_completed_runs_total Total completed agent runs since startup
# TYPE conductor_completed_runs_total counter
conductor_completed_runs_total 47

# HELP conductor_failed_runs_total Total failed agent runs since startup
# TYPE conductor_failed_runs_total counter
conductor_failed_runs_total 2

# HELP conductor_messagebus_appends_total Total message bus append operations
# TYPE conductor_messagebus_appends_total counter
conductor_messagebus_appends_total 1234

# HELP conductor_api_requests_total Total API requests by method and path
# TYPE conductor_api_requests_total counter
conductor_api_requests_total{method="GET",status="200"} 100
conductor_api_requests_total{method="POST",status="201"} 5
```

### Implementation Approach

**Step 1: Create metrics package** (`internal/metrics/metrics.go`)

Create a simple metrics registry with atomic counters:
```go
package metrics

import (
    "fmt"
    "sync/atomic"
    "time"
)

// Registry holds all metrics for the conductor server.
type Registry struct {
    startTime          time.Time
    activeRuns         atomic.Int64
    completedRuns      atomic.Int64
    failedRuns         atomic.Int64
    apiRequestsTotal   sync.Map // key: "METHOD:status_class" -> *atomic.Int64
}

func New() *Registry { return &Registry{startTime: time.Now()} }
func (r *Registry) IncActiveRuns()     { r.activeRuns.Add(1) }
func (r *Registry) DecActiveRuns()     { r.activeRuns.Add(-1) }
func (r *Registry) IncCompletedRuns()  { r.completedRuns.Add(1) }
func (r *Registry) IncFailedRuns()     { r.failedRuns.Add(1) }
func (r *Registry) RecordRequest(method string, statusCode int) { /* track by method+status */ }
func (r *Registry) Render() string     { /* write Prometheus text format */ }
```

**Step 2: Add to Server** (`internal/api/server.go`)

Add `Metrics *metrics.Registry` field to `Options` and `Server` struct.
Create a default registry if nil.

**Step 3: Wire up middleware** (`internal/api/middleware.go` or `routes.go`)

Update the logging middleware to call `s.metrics.RecordRequest(method, statusCode)`.

**Step 4: Add route** (`internal/api/routes.go`)

```go
mux.Handle("/metrics", s.wrap(s.handleMetrics))
```

**Step 5: Handler** (`internal/api/handlers_extra.go` or a new file)

```go
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "text/plain; version=0.0.4")
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, s.metrics.Render())
}
```

**Step 6: Track active/completed/failed runs**

In the run handlers (handlers_projects.go or handlers.go), when a run:
- Starts: `s.metrics.IncActiveRuns()`
- Completes: `s.metrics.DecActiveRuns(); s.metrics.IncCompletedRuns()`
- Fails: `s.metrics.DecActiveRuns(); s.metrics.IncFailedRuns()`

Look at where runs are created and completed in the API to find the right hooks.

### Files to Modify/Create

1. **New: `internal/metrics/metrics.go`** - Metrics registry
2. **New: `internal/metrics/metrics_test.go`** - Tests
3. **`internal/api/server.go`** - Add Metrics field to Options/Server
4. **`internal/api/routes.go`** - Add `/metrics` route
5. **`internal/api/handlers_extra.go`** (or new file) - Add handleMetrics
6. **`internal/api/middleware.go`** - Track API requests in metrics
7. **`docs/user/api-reference.md`** - Document `/metrics` endpoint

### Tests

1. Test that `GET /metrics` returns 200 with Prometheus format
2. Test that uptime increases over time
3. Test counter increments (completed runs, failed runs)
4. Test that wrong method returns 405

### Quality Gates

After implementation:
- `go build ./...` must pass
- `go test -race ./internal/... ./cmd/...` must pass
- All existing tests must continue to pass
- New tests must pass
- No new external dependencies added (verify with `go.sum` not changing)

## Implementation Steps

1. Read the relevant files: `internal/api/server.go`, `internal/api/routes.go`, `internal/api/middleware.go`, `internal/api/handlers_extra.go`
2. Create `internal/metrics/metrics.go` with the Registry type
3. Integrate into Server
4. Add route and handler
5. Wire up middleware for request counting
6. Add tests
7. Update docs
8. Run quality gates
9. Commit: `feat(api): add Prometheus-compatible /metrics endpoint`

## Working Directory

/Users/jonnyzzz/Work/conductor-loop

## Done Signal

Create the file `DONE` in the task directory when complete.
