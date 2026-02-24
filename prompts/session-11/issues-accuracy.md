# Task: Update ISSUES.md to Reflect Accurate Current State

## Context

You are a research/documentation agent for the Conductor Loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

Read these files first:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — conventions
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md — current issues file
- /Users/jonnyzzz/Work/conductor-loop/MESSAGE-BUS.md — session history (for recent resolutions)

## Task

Review and update /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md to accurately reflect the current implementation state. Many issues have been resolved in sessions #5 through #10 but the document may not fully reflect this.

### What to Do

1. **Read ISSUES.md** to understand current documented state
2. **Cross-reference MESSAGE-BUS.md** for resolutions made in sessions #5-#10
3. **Verify each OPEN/PARTIALLY RESOLVED issue** by checking the actual code:
   - ISSUE-002 (Windows file locking): Check internal/messagebus/lock_windows.go exists
   - ISSUE-003 (Windows process groups): Check internal/runner/pgid_windows.go etc.
   - ISSUE-004 (CLI version compatibility): Check internal/runner/validate.go parseVersion/isVersionCompatible
   - ISSUE-005 (Runner bottleneck): Review internal/runner/job.go — is it still monolithic?
   - ISSUE-006 (Storage-MessageBus dependency): Check actual import graph
   - ISSUE-007 (Message bus lock contention): Check internal/messagebus/messagebus.go retry logic
   - ISSUE-008 (Integration validation): Check test/integration/ directory
   - ISSUE-009 (Token expiration): Check internal/runner/validate.go ValidateToken
   - ISSUE-010 (Error context): Check internal/runner/job.go tailFile/classifyExitCode
   - ISSUE-011 through ISSUE-018: Review if still applicable
   - ISSUE-019 (concurrent run-info.yaml): Check internal/storage/atomic.go file locking
   - ISSUE-020 (message bus circular dep): Check test/integration/orchestration_test.go
   - ISSUE-021 (data race in Server): Check internal/api/server.go or cmd/run-agent/serve.go for mutex

4. **Update the Summary Table** at the bottom of ISSUES.md to reflect accurate counts

5. **Add a "Session #11 Changes" section** at the bottom noting what was verified

### Code to Examine

Use `find`, `cat`, or `grep` to check:
```bash
# Check if Windows platform files exist
ls internal/messagebus/lock_windows.go 2>/dev/null
ls internal/runner/pgid_windows.go 2>/dev/null
ls internal/runner/stop_windows.go 2>/dev/null
ls internal/runner/wait_windows.go 2>/dev/null

# Check validate.go for version parsing
grep -n "parseVersion\|isVersionCompatible\|minVersions" internal/runner/validate.go

# Check retry logic in messagebus
grep -n "WithMaxRetries\|WithRetryBackoff\|ContentionStats" internal/messagebus/messagebus.go

# Check UpdateRunInfo locking
grep -n "LockExclusive\|lock" internal/storage/atomic.go

# Check Server mutex
grep -rn "sync.Mutex" internal/api/ cmd/run-agent/serve.go 2>/dev/null
```

### Quality Requirements

1. Only edit ISSUES.md (no code changes)
2. Follow the existing document format
3. Be accurate — verify with actual code before marking resolved
4. Commit with: `docs(issues): update ISSUES.md to reflect accurate session #11 state`

After completing, write a summary of what you changed.
