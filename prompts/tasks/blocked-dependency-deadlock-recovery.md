# Task: Blocked Dependency Deadlock Recovery

## Context

- **ID**: `task-20260223-155220-blocked-dependency-deadlock-recovery`
- **Priority**: P0
- **Source**: `docs/dev/todos.md`; `docs/facts/FACTS-suggested-tasks.md`; filesystem scan 2026-02-23

Two task directories currently exist with no run history and no DONE marker:
- `task-20260222-102110-job-batch-cli`
- `task-20260222-102120-workflow-runner-cli`

`docs/dev/todos.md` marks these as `[x]` (done), but no implementation evidence exists.
`workflow-runner-cli` is gated on `job-batch-cli` per FACTS, creating a DAG deadlock:
neither task has runs, neither is DONE, and the dependency chain prevents progress.

## Root Cause

The dependency tracking system has no mechanism for:
1. **Diagnosing stuck chains**: identifying tasks that are blocked and have no running ancestors
2. **Auto-escalation**: surfacing these to the operator without manual filesystem inspection
3. **Cancellation/supersession**: marking a blocked task as cancelled or superseded without full implementation

## Requirements

### Short-term (this task)
1. **Diagnostic command**: `run-agent list --blocked` — shows tasks with no runs, no DONE,
   where all their dependents also have no active runs.
2. **Cancel/supersede command**: `run-agent task cancel --task <id> --reason "superseded by X"`
   — marks task as cancelled (writes `CANCELLED` marker with reason) without deleting task folder.
3. **Unblock command**: `run-agent task unblock --task <id>` — clears dependency gating
   so task can start even if parent is not DONE yet.
4. Resolve the two specific blocked tasks above by either executing them or cancelling
   them with documented reason.

### Medium-term
5. Add dependency validation at `run-agent job` submit time: warn if specified dependencies
   are DONE=absent, running=false (zombie dependencies).

## Acceptance Criteria

- `run-agent list --blocked --root runs --project conductor-loop` outputs blocked task chains.
- `run-agent task cancel` writes a `CANCELLED` marker file with timestamp and reason.
- `task-20260222-102110-job-batch-cli` and `task-20260222-102120-workflow-runner-cli` are
  resolved: either implement their specs or cancel with documented reason.
- `go test ./internal/... ./cmd/...` passes.

## Verification

```bash
go build -o bin/run-agent ./cmd/run-agent
./bin/run-agent list --blocked --root runs --project conductor-loop
./bin/run-agent task cancel --root runs --project conductor-loop \
  --task task-20260222-102110-job-batch-cli --reason "superseded by batch submission in conductor"
go test ./internal/storage ./cmd/run-agent -run 'TestBlocked|TestCancel' -count=1
```

## Reference Files

- `internal/storage/` — task/run storage
- `cmd/run-agent/list.go` — list command
- `docs/facts/FACTS-issues-decisions.md` — DAG design notes
- `docs/facts/FACTS-suggested-tasks.md` — blocked task inventory
