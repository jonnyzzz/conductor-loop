# Task: Fix Multi-Second UI Update Latency

## Context

Users report multi-second lag between backend state changes and visible UI updates.

Relevant paths:
- Frontend refresh and stream orchestration:
  - `frontend/src/hooks/useAPI.tsx`
  - `frontend/src/hooks/useLiveRunRefresh.ts`
  - `frontend/src/hooks/useSSE.ts`
- Tree and message rendering surfaces:
  - `frontend/src/components/TreePanel.tsx`
  - `frontend/src/components/MessageBus.tsx`
- Backend SSE defaults:
  - `internal/api/sse.go` (default poll interval)

Current tuning shows competing timings (stream + fallback polling + debounce windows), which can amplify delay under reconnect/error transitions.

## Scope

Reduce end-to-end time-to-visible-update for active runs/tasks without causing CPU regressions.

## Requirements

1. Establish baseline measurement:
- Add a reproducible latency measurement method (manual script or test harness):
  - trigger a known message/run state change
  - capture time until it appears in UI tree/message panel
- Record baseline p50/p95 in task output.

2. Optimize update pipeline:
- Review and tune the relevant polling/refresh constants in `useAPI.tsx` and `useLiveRunRefresh.ts`.
- Ensure SSE healthy state avoids redundant slow fallback behavior.
- Reduce avoidable debounce/batching delay on critical status updates.
- Keep reconnect behavior stable (no event storms / duplicate replay regressions).

3. Prevent render churn bottlenecks:
- Verify tree updates only recompute/re-render when data actually changed.
- Preserve existing stabilization helpers (`stabilizeFlatRuns`, merge helpers) while fixing stale-update delays.

4. Add regression coverage:
- Add frontend tests for update propagation timing behavior (stream event -> query cache -> rendered UI).
- Include at least one test for fallback polling path when SSE enters error state.

5. Keep CPU budget safe:
- Validate no new high-frequency loops are introduced.
- Confirm frontend build and tests pass.

## Acceptance Criteria

- Measured end-to-end UI update latency is consistently sub-second for active-run updates in normal conditions.
- Stream reconnect/fallback path does not reintroduce multi-second lag.
- No visible regression in tree correctness or message ordering.

## Verification

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Frontend tests/build
cd frontend
npm test
npm run build

# Backend SSE tests still pass
cd /Users/jonnyzzz/Work/conductor-loop
go test ./internal/api -run 'TestSSE' -count=1
```

## Key Files

- `frontend/src/hooks/useAPI.tsx`
- `frontend/src/hooks/useLiveRunRefresh.ts`
- `frontend/src/hooks/useSSE.ts`
- `frontend/src/components/TreePanel.tsx`
- `frontend/src/components/MessageBus.tsx`
- `internal/api/sse.go`
