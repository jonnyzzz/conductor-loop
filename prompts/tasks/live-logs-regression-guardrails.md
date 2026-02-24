# Task: Lock Live-Log Layout/Visibility with Regression Tests

## Context

- **ID**: `task-20260223-155310-live-logs-regression-guardrails`
- **Priority**: P1
- **Source**: `docs/dev/todos.md`

Live-log visibility (the STDOUT/LOGS tab in the Web UI) has experienced repeated regressions:
- Logs stop updating mid-run without visual indication
- Log container height collapses after task state transition
- Reconnect after browser tab visibility change loses scroll position

These regressions recur because there is no test suite locking the expected behavior.
Each fix is followed by a different regression in an adjacent code path.

## Requirements

1. **Regression test suite**: Create a test file (`frontend/src/live-logs.regression.test.ts`
   or equivalent) that covers:
   - Initial log load: logs appear within 2s of task start
   - Live streaming: new lines append to bottom without re-render flash
   - Scroll behavior: auto-scroll to bottom while streaming; manual scroll pauses auto-scroll
   - Reconnect: after tab visibility hidden → visible, streaming resumes
   - Empty state: "No output yet" placeholder shown until first log line

2. **Backend test**: Integration test that verifies `GET /api/.../stdout` returns
   incremental content after each write to `agent-stdout.txt`.

3. **Visual regression** (optional, if vitest/storybook available): Snapshot test for
   log container rendered state.

4. **Fix any existing failures**: All tests must pass. If a test reveals a current bug,
   fix it as part of this task.

## Acceptance Criteria

- At least 5 new regression tests covering the behaviors listed above.
- All new tests pass in CI (`npm test --run` or equivalent).
- `go test ./internal/api -run 'TestStdout' -count=1` passes for backend coverage.
- `go build ./...` passes.

## Verification

```bash
cd frontend && npm test -- --run --reporter verbose 2>&1 | grep -E 'live.log|PASS|FAIL'
go test ./internal/api -run 'TestStdout' -count=1
```

## Reference Files

- `frontend/src/` — React components for log viewing
- `internal/api/logs.go` (or equivalent) — stdout streaming endpoint
- `docs/dev/todos.md` — feature request origin
- `docs/facts/FACTS-agents-ui.md` — agent output rendering facts
