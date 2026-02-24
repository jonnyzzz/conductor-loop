# Task: Implement Windows Job Objects for Runner Process-Tree Control

## Context
- `docs/roadmap/technical-debt.md` tracks ISSUE-003 as PARTIALLY RESOLVED: Windows process-group support remains stubbed.
- `docs/facts/FACTS-issues-decisions.md` marks ISSUE-003 as deferred medium-term work requiring Job Objects.
- Current Windows implementation is a workaround:
  - `internal/runner/pgid_windows.go` sets `CREATE_NEW_PROCESS_GROUP` and returns PID as PGID.
  - `internal/runner/stop_windows.go` kills only the root PID.
  - `internal/runner/wait_windows.go` uses best-effort alive checks and cannot represent full process-tree state.
- Unix code relies on true process groups for child detection and termination semantics; Windows parity is currently missing.

## Requirements
- Implement Windows Job Object-based process-group management centered on `internal/runner/pgid_windows.go` (plus required wiring in related Windows runner files):
  - Create/attach each spawned run process to a Job Object.
  - Configure the Job Object for process-tree lifecycle control (including kill-on-close semantics).
  - Track/retrieve per-run Job Object context so runner stop/wait paths can act on the whole tree, not only the parent PID.
- Replace PID-as-PGID behavior with a Windows-native group abstraction that supports:
  - Process-tree liveness checks.
  - Whole-tree termination.
  - Safe cleanup of Job Object handles/resources.
- Update Windows stop/wait behavior to use Job Object APIs (e.g., `TerminateJobObject`, child/activity queries) instead of single-PID assumptions.
- Keep non-Windows behavior unchanged.
- Add Windows-focused tests (unit and/or integration) that prove descendant-process handling:
  - A parent process that spawns children is treated as one managed group.
  - Termination path kills descendants, not just the parent.
  - Liveness transitions to not-alive after group termination.
  - If native Windows execution is unavailable in CI, add mockable API layer tests plus at least one runnable Windows integration test in CI documentation/plan.

## Acceptance Criteria
- Windows runner no longer treats PGID as plain PID workaround.
- Stop/kill operations terminate the full process tree on Windows.
- Child-process liveness checks reflect Job Object state accurately enough for runner wait logic.
- Job Object handles/resources are cleaned up without leaks.
- Existing Unix tests/behavior remain unaffected.

## Verification
```bash
go test ./internal/runner -count=1

# Compile-check Windows path from non-Windows hosts
GOOS=windows GOARCH=amd64 go test ./internal/runner -c

# Run Windows-specific job-object tests on a Windows runner
go test ./internal/runner -run 'TestWindows.*(JobObject|ProcessTree|TerminateChildren|Liveness)' -count=1
```
Expected: Windows tests confirm descendant processes are terminated and wait/liveness logic observes full tree shutdown.
