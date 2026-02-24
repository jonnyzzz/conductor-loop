# Task: Implement Windows Shared-Lock Readers for Message Bus

## Context
- `docs/roadmap/technical-debt.md` tracks ISSUE-002 as still PARTIALLY RESOLVED: Windows mandatory locking still blocks lockless readers.
- `docs/facts/FACTS-issues-decisions.md` marks ISSUE-002 as critical and deferred for medium-term Windows reader behavior.
- `internal/messagebus/lock_windows.go` currently exposes only exclusive lock behavior (`LockFileEx` with `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY`).
- `internal/messagebus/messagebus.go:393-411` reads with `os.ReadFile` and no lock acquisition. On Windows this can fail/block when a writer holds the lock.
- The result is platform drift: Unix/macOS lockless reads work, while Windows can stall message polling.

## Requirements
- Implement Windows reader locking with shared-lock semantics and timeout/retry behavior in `internal/messagebus/lock_windows.go` (and minimal supporting changes where needed):
  - Add Windows shared lock acquisition path (`LockFileEx` shared mode, fail-immediately + retry loop).
  - Reuse existing timeout model (`ErrLockTimeout`, poll interval, retry-until-deadline semantics) so reader and writer behavior is consistent.
- Update message bus read path to use the Windows shared lock flow before reading bus content, while preserving current Unix behavior.
- Ensure all lock handles are correctly released on success and error paths.
- Keep API compatibility for existing call sites where possible (avoid broad refactors outside message bus locking/read code).
- Add tests covering concurrent read/write behavior under Windows locking rules:
  - Reader blocks/retries while exclusive writer lock is held.
  - Reader succeeds after writer unlocks.
  - Reader times out with `ErrLockTimeout` when lock cannot be acquired.
  - If native Windows runtime tests are not available in CI, add syscall-abstraction/mock tests that validate retry/timeout semantics deterministically.

## Acceptance Criteria
- Windows message-bus readers no longer rely on lockless file reads when writer locks are active.
- Concurrent read/write on Windows behaves predictably: retries occur, timeout is bounded, and successful reads resume after lock release.
- `ErrLockTimeout` is surfaced for reader lock acquisition timeout cases.
- Unix behavior is unchanged (no regression in existing message bus tests).
- New/updated tests protect against regression of ISSUE-002 behavior.

## Verification
```bash
go test ./internal/messagebus -count=1

# Compile-check Windows path from non-Windows hosts
GOOS=windows GOARCH=amd64 go test ./internal/messagebus -c

# Run Windows-specific locking tests on a Windows runner (or equivalent CI)
go test ./internal/messagebus -run 'TestWindows.*(SharedLock|ReadRetry|ReadTimeout|ConcurrentReadWrite)' -count=1
```
Expected: Windows reader tests show retry/timeout behavior and successful post-unlock reads; baseline messagebus tests remain green.
