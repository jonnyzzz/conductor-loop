# Task: Define and Enforce Refresh/SSE CPU Budget

## Context

- **ID**: `task-20260223-155340-ui-refresh-churn-cpu-budget`
- **Priority**: P1
- **Source**: `docs/dev/todos.md`

The web server and browser client both exhibit high refresh/polling churn:
- **Server-side**: SSE stream polls the bus file every 100ms (tracked in `fix-sse-cpu-hotspot.md`)
- **Client-side**: React re-renders full tree on every SSE event, even if nothing changed
- **Combined**: 10 active SSE clients × 100ms poll × full tree re-render = O(N) CPU churn

There is currently no measurement or enforcement of these costs. They grow silently as
more agents are active.

## Requirements

1. **Measure baseline**: Add a Go benchmark (`internal/api/sse_bench_test.go`) that
   measures CPU/alloc for 1, 5, and 10 concurrent SSE clients for 10 seconds. Record
   baseline numbers in benchmark output comments.

2. **Server-side budget**: Enforce via CI that the per-client SSE handler CPU cost does
   not exceed `<threshold>` ns/op (set after measuring baseline). Use `go test -bench`
   with `-benchmem` and fail CI if regressions occur.

3. **Client-side budget**: Add React performance profiling to identify components that
   re-render on every SSE event unnecessarily. Memo-ize stable components. Target: zero
   full-tree re-renders for task-list SSE updates when task list is unchanged.

4. **Documentation**: Add `docs/dev/performance-budget.md` documenting the agreed budgets
   and how to run the benchmark locally.

## Acceptance Criteria

- Go benchmark exists and runs cleanly: `go test ./internal/api -bench 'BenchmarkSSE' -benchmem`
- Benchmark results are recorded in a comment block in the test file for reference.
- `npm run profile` (or equivalent) can measure React re-render frequency per SSE event.
- `docs/dev/performance-budget.md` created with current baselines.
- All existing tests still pass.

## Verification

```bash
go test ./internal/api -bench 'BenchmarkSSE' -benchmem -count=3
cd frontend && npm run build && npm test -- --run
go test ./... -count=1
```

## Reference Files

- `internal/api/sse.go` — SSE polling implementation
- `internal/api/sse_test.go` — existing SSE tests
- `frontend/src/` — React tree components
- `prompts/tasks/fix-sse-cpu-hotspot.md` — companion server-side fix
- `docs/dev/todos.md` — feature request origin
