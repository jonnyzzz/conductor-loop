# Task: Issues Housekeeping + ISSUE-006 Closure

## Context

You are a senior Go developer working on the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

This is an issues housekeeping task for session #25.

## Objective

Close ISSUE-006 (Storage-MessageBus Dependency Inversion) as a planning artifact, and audit all other open/partially-resolved issues to see if any can be closed.

## Background

**ISSUE-006**: "Storage-MessageBus Dependency Inversion in Phase 1"
- Status: HIGH/OPEN
- Description: "Current plan shows infra-messagebus depends on infra-storage, but this is backwards or unnecessary"
- This was a development planning concern about Phase 1 implementation parallelism
- The code is now fully implemented and the dependency is confirmed one-directional: `internal/storage/atomic.go` imports `internal/messagebus` (for `LockExclusive`)
- This is NOT circular - storage uses messagebus, messagebus does NOT use storage
- The planning concern (that it would block Phase 1 parallelism) is moot since all code is implemented
- Resolution: Mark as RESOLVED - planning artifact, implementation complete and correct

**Other issues to audit:**
- ISSUE-002: Windows File Locking - PARTIALLY RESOLVED. Has any progress been made?
- ISSUE-003: Windows Process Groups - PARTIALLY RESOLVED. Deferred items still deferred?
- ISSUE-004: CLI Version Compatibility - PARTIALLY RESOLVED. Any deferred items worth doing?
- ISSUE-009: Token Expiration - PARTIALLY RESOLVED. Any quick wins?
- ISSUE-016: Message Bus File Size - PARTIALLY RESOLVED. Will be addressed by separate agent (auto-rotate feature). Leave as-is for now.
- ISSUE-017, 018: LOW/informational - verify still accurate

## What To Do

1. Read `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md` carefully
2. Read relevant code files to verify the current implementation state
3. Mark ISSUE-006 as RESOLVED in ISSUES.md with proper explanation
4. Audit other PARTIALLY RESOLVED issues and check if any deferred items are now worth doing or should be explicitly closed as "deferred indefinitely"
5. Update the issue summary table at the bottom of ISSUES.md
6. Commit your changes: `git -C /Users/jonnyzzz/Work/conductor-loop commit -am "docs(issues): close ISSUE-006 planning artifact + audit open issues"`

## Important Notes

- Use `go build ./...` to verify nothing is broken (but you should NOT be changing any code)
- Read all relevant code files before updating ISSUES.md
- Specifically verify: `grep -rn "messagebus" /Users/jonnyzzz/Work/conductor-loop/internal/storage/` to confirm the storage→messagebus dependency direction
- When done, create a `DONE` file in the task folder: `touch $TASK_FOLDER/DONE`

## Quality Gate

- No code changes needed for this task
- ISSUES.md summary table must be updated accurately
- Commit must succeed (use absolute path: `git -C /Users/jonnyzzz/Work/conductor-loop ...`)
