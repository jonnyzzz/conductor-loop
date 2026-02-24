# Task: Fix Message Bus Empty-State Regression

## Context

- **ID**: `task-20260223-155300-messagebus-empty-regression-investigation`
- **Priority**: P1
- **Source**: `docs/dev/todos.md`

The Message Bus panel intermittently renders as empty (no messages shown) even when
`TASK-MESSAGE-BUS.md` contains messages. The behavior is non-deterministic:
- Reloading the page sometimes recovers, sometimes not
- SSE degradation appears to be a trigger: when SSE reconnects, the hydration logic
  may miss the "since" ID and start from wrong position
- Legacy message format (mixed markdown/YAML entries in older bus files) may cause
  the parser to return zero messages rather than a partial set

Known code locations:
- `internal/api/sse.go` — SSE message stream handler; `ErrSinceIDNotFound` handling
- `internal/messagebus/messagebus.go` — `ReadMessages(lastID)` — returns empty on parse error
- `frontend/src/` — message bus panel hydration on SSE reconnect

## Requirements

1. **Deterministic hydration**: On SSE reconnect (or initial page load), the panel must
   always show at least the last N messages (N=20 default). Empty display must be a
   genuine zero-message state, not a parse/load failure.

2. **Error recovery**: When `ReadMessages(lastID)` returns an error or empty set due to
   parse failure, fall back to reading the bus file from the beginning and returning
   whatever messages are parseable (partial recovery > empty).

3. **SSE ErrSinceIDNotFound**: When the server can't find `lastID` in bus history (rotation
   or GC), reset to start-of-file rather than returning empty.

4. **Frontend resilience**: The message bus React component should distinguish between
   "loading" (spinner) and "error" (retry button) states rather than silently showing empty.

5. **Tests**: Add a regression test that verifies the panel is non-empty after an SSE
   reconnect with an expired lastID.

## Acceptance Criteria

- After SSE reconnect with an expired lastID, the panel shows at least 1 message (not empty).
- `ReadMessages("")` always returns all parseable messages in the bus file, even if some
  entries have unknown format.
- `go test ./internal/messagebus ./internal/api -count=1` passes.
- `go build ./...` passes.

## Verification

```bash
go test ./internal/messagebus -run 'TestReadMessages|TestEmptyFallback' -count=1
go test ./internal/api -run 'TestSSEReconnect|TestBusHydration' -count=1
# Manual: open web UI, disconnect/reconnect network, verify messages still visible
```

## Reference Files

- `internal/messagebus/messagebus.go` — core read/parse logic
- `internal/api/sse.go` — SSE stream handler
- `frontend/src/components/MessageBusPanel.*` — React message bus component
- `docs/dev/todos.md` — feature request origin
