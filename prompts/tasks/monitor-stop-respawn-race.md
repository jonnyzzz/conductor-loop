# Task: Fix Monitor Stop-Respawn Race in Background Monitor Loops

## Context

- **ID**: `task-20260223-155210-monitor-stop-respawn-race`
- **Priority**: P0
- **Source**: `docs/dev/todos.md` (Orchestration / Multi-Agent section)

When a user issues `run-agent stop` to terminate an active task, background monitor loops
that are watching the same task can immediately trigger a restart — overriding the stop
intent. This creates a confusing loop: the user stops a task, but it instantly respawns.

Current behavioral flow:
1. User runs `run-agent stop --project P --task T`
2. Monitor loop (e.g., `conductor watch` or background shell poller) sees task in
   `not-running` state and invokes restart logic
3. Task is restarted within seconds of the manual stop

Known problematic patterns:
- `cmd/run-agent/watch.go` polls status and can trigger restart without suppression
- There is no in-memory or filesystem signal to suppress restart after a manual stop
- `DONE` file semantics exist but are insufficient: a stopped (non-DONE) task looks
  identical to a task that finished waiting for restart

## Requirements

1. **Suppression window**: After `run-agent stop`, establish an explicit suppression period
   (configurable, default 60s) during which monitor loops must NOT restart the task.
2. **Suppression signal**: Write a `STOP-REQUESTED` marker (or equivalent) to the task
   folder that monitor loops must check before triggering a restart.
3. **Restart policy**: Document and enforce the reasoned restart policy — only restart a
   task when DONE is absent AND no suppression signal is present AND idle timeout is met.
4. **CLI flag**: `run-agent stop --no-restart` to set a permanent suppress (no auto-restart).
5. **Tests**: Add integration tests verifying that after `stop`, a polling loop does not
   restart the task within the suppression window.

## Acceptance Criteria

- After `run-agent stop`, task remains stopped for at least 60s even with an active watch loop.
- `run-agent stop --no-restart` prevents automatic restart indefinitely.
- Suppression file is cleaned up after it expires or task explicitly resumes.
- Monitor loops log the suppression skip: `"skip restart: stop-requested within window"`.
- `go test ./internal/... ./cmd/...` passes with no new race conditions.

## Verification

```bash
# Build
go build -o bin/run-agent ./cmd/run-agent && go build -o bin/conductor ./cmd/conductor

# Unit / integration
go test ./internal/runner ./cmd/run-agent -run 'TestStop|TestWatch' -count=1

# Race detector
go test -race ./internal/runner ./internal/storage -count=1

# Manual smoke test
./bin/run-agent stop --root runs --project conductor-loop --task <task-id>
sleep 5
# Verify task has NOT restarted:
./bin/run-agent list --root runs --project conductor-loop
```

## Reference Files

- `cmd/run-agent/watch.go` — background watch loop
- `internal/runner/` — runner restart logic
- `internal/storage/task.go` — task folder marker files
- `docs/dev/todos.md` — feature request origin
