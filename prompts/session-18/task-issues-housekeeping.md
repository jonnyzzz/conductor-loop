# Task: ISSUES.md and QUESTIONS.md Housekeeping

## Overview

Update ISSUES.md and related documentation to reflect the actual current state of implementation.
Several issues and questions are marked incorrectly (open when resolved, or missing resolution details).

## Context

Read these files first:
- `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md` — conventions
- `/Users/jonnyzzz/Work/conductor-loop/Instructions.md` — project structure
- `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md` — current issues list
- `/Users/jonnyzzz/Work/conductor-loop/QUESTIONS.md` — design questions
- `/Users/jonnyzzz/Work/conductor-loop/MESSAGE-BUS.md` — session history (last 200 lines are most relevant)

## Specific Changes to Make

### 1. Mark ISSUE-015 as RESOLVED in ISSUES.md

ISSUE-015 "Run Directory Accumulation Without Cleanup" is currently marked as OPEN.
The `run-agent gc` command was implemented in session #11-12. Verify by checking:
- `cmd/run-agent/gc_test.go` — gc command tests
- `cmd/run-agent/gc.go` or similar — gc command implementation

Once verified, update ISSUE-015:
- Change Status from `OPEN` to `RESOLVED`
- Add `Resolved: 2026-02-20`
- Add resolution notes: "Implemented `run-agent gc` command with --root, --older-than, --dry-run, --project, --keep-failed flags"
- Update the issue summary table at the bottom (MEDIUM open: 6 → 5, MEDIUM resolved: 0 → 1)

### 2. Update QUESTIONS.md spec file answers

Review `docs/specifications/subsystem-message-bus-tools-QUESTIONS.md` to verify which answers have been implemented. Check:
- Q2: POST /api/v1/messages — verify in `internal/api/routes.go` (it exists, "POST /api/v1/messages")
- Q4: RUN_START/RUN_STOP/RUN_CRASH events — verify in `internal/messagebus/messagebus.go` or related files
- Q5: SSE message stream with id field — verify in `internal/api/sse.go`

For each Q that has been implemented, add an "Implementation (date): ..." note below the Answer in the file.

### 3. Update env-contract QUESTIONS file

Review `docs/specifications/subsystem-env-contract-QUESTIONS.md`:
- The TODO about CLAUDECODE env var research — add a brief note: "CLAUDECODE is set by the Claude CLI when it runs; sub-agents inherit it. No special handling needed in run-agent."
- The TODO2 about agent env integration tests — this is already tracked in ISSUES.md if applicable; otherwise add a note that Docker-based env comparison tests are deferred.

### 4. Update ISSUES.md summary section

After the changes above, update the session notes at the bottom of ISSUES.md:
Add a new section:
```
### Session #18 Changes (2026-02-20)

**ISSUE-015**: RESOLVED — `run-agent gc` command verified implemented with full flag set.
Summary table updated: MEDIUM open 6 → 5, MEDIUM resolved 0 → 1.
```

## Quality Gates

1. `go build ./...` must pass
2. `go test ./...` — all 18 packages must pass
3. No content removed from MESSAGE-BUS.md (append-only rule)

## Commit format

```
docs(issues): mark ISSUE-015 resolved and update spec question implementations

- ISSUE-015: gc command verified implemented, status → RESOLVED
- message-bus-tools-QUESTIONS: note POST messages, SSE id, RUN_CRASH implemented
- env-contract-QUESTIONS: add CLAUDECODE note
```

## Output

Write your output summary to `$RUN_FOLDER/output.md` and create `$TASK_FOLDER/DONE` when complete.
