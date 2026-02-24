# Sub-Agent Task: Issues Housekeeping + Docs Review (Session #26)

## Role
You are an implementation agent. Your CWD is /Users/jonnyzzz/Work/conductor-loop.

## Task
Fix the ISSUES.md summary table inconsistency and review docs/dev/ for accuracy.

## Context
The ISSUES.md summary table at the bottom shows:
- MEDIUM: 1 open, 0 partially resolved, 5 resolved
- Total: 1 open, 5 partially resolved, 16 resolved

But in Session #25, ISSUE-016 was RESOLVED (WithAutoRotate was implemented). All MEDIUM issues are now resolved.
The correct state is:
- MEDIUM: 0 open, 0 partially resolved, 6 resolved
- Total: 0 open, 5 partially resolved, 17 resolved

Also, the Session #25 summary in MESSAGE-BUS.md says "Total: 0 fully open" but the table is wrong.

## Required Actions

### 1. Fix ISSUES.md Summary Table
In /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md, update the summary table to:
```
| Severity | Open | Partially Resolved | Resolved |
|----------|------|-------------------|----------|
| CRITICAL | 0 | 2 | 4 |
| HIGH | 0 | 3 | 5 |
| MEDIUM | 0 | 0 | 6 |
| LOW | 0 | 0 | 2 |
| **Total** | **0** | **5** | **17** |
```

Add a "Session #26 Changes" section at the bottom of ISSUES.md (before "## References"):
```
### Session #26 Changes (2026-02-21)

**ISSUE-016** (MEDIUM): Table corrected — ISSUE-016 was RESOLVED in Session #25 (WithAutoRotate
implemented in afa9673) but the summary table was not updated. Correcting: MEDIUM open 1 → 0,
MEDIUM resolved 5 → 6, Total open 1 → 0, Total resolved 16 → 17.
```

### 2. Review docs/dev/ for Accuracy
Read each file in /Users/jonnyzzz/Work/conductor-loop/docs/dev/ and check for stale references:
- architecture.md — check if it matches the actual package structure
- ralph-loop.md — verify the loop algorithm description matches internal/runner/ralph.go
- storage-layout.md — check paths match actual code
- message-bus.md — check for references to features that now exist (auto-rotate, WithAutoRotate, ReadLastN)
- testing.md — verify test commands match current Makefile/go test patterns
- development-setup.md — check setup steps are still accurate
- contributing.md — check contribution guidelines
- agent-protocol.md — verify JRUN_* variables are documented (they are in the preamble now)
- adding-agents.md — check if xAI backend docs are accurate
- subsystems.md — check package paths

For each file that has stale information, fix it. Focus on:
- webhook package (internal/webhook/) — may not be mentioned yet
- WithAutoRotate option — may not be mentioned
- ReadLastN method — may not be mentioned
- /api/v1/status endpoint — may not be mentioned
- run-agent output command — may not be mentioned

### 3. Commit Changes
After making all changes:
1. Verify with: go build ./... (must pass)
2. Verify with: go test ./internal/... ./cmd/... (must all pass)
3. Commit: git add ISSUES.md docs/dev/*.md
4. Commit message: `docs(issues): fix ISSUES.md table + update dev docs for session #25 features`

### 4. Create DONE file
After the commit, create the DONE file to signal completion:
```bash
touch /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/${JRUN_TASK_ID}/DONE
```

## Quality Requirements
- go build ./... must pass
- go test ./internal/... ./cmd/... must pass
- No code changes — only documentation
