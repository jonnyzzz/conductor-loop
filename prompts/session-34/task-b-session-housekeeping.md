# Task B: Session #33 Documentation + Housekeeping

## Context

You are an expert Go developer and technical writer working on the conductor-loop project.
Project root: /Users/jonnyzzz/Work/conductor-loop

Session #33 (the last session before this one) added two new commands to the `conductor` CLI:

1. **`conductor status`** — calls GET /api/v1/status, shows version/uptime/active-runs/configured-agents
2. **`conductor task status <task-id>`** — calls GET /api/v1/tasks/{id}, shows task status and runs table
3. **`conductor task stop <task-id>`** — calls DELETE /api/v1/tasks/{id}, stops all running runs

These were committed as: `98f6667 feat(cli): add conductor status and task stop commands`

## Goal

Two main tasks:

### Task 1: Verify quality gate (go test -race)

Session #33 skipped `go test -race`. Verify it passes now:

```bash
# Run from /Users/jonnyzzz/Work/conductor-loop:
go test -race ./internal/... ./cmd/...
```

If it fails, investigate and fix the race condition. If it passes, document the result.

### Task 2: Update the project MEMORY.md for session #33

The file `/Users/jonnyzzz/.claude/projects/-Users-jonnyzzz-Work-conductor-loop/memory/MEMORY.md`
has "Project State (as of Session #32)" — it needs to be updated to reflect session #33.

Read the current MEMORY.md and update the "Project State" section to say:
- "(as of Session #33, 2026-02-21)"
- Add the new conductor status and task stop/status commands to the "Key Commands" section

New commands to add to the Key Commands section:
```bash
# conductor server client commands (require conductor server running)
./bin/conductor status [--server URL] [--json]
./bin/conductor task status <task-id> [--project P] [--server URL] [--json]
./bin/conductor task stop <task-id> [--project P] [--server URL] [--json]
```

Also update the "Architecture Notes" or a new "Session #33 Additions" section:
```
## Session #33 Additions (2026-02-21)

- **feat(cli): conductor status + task commands** (commit 98f6667)
  - conductor status: GET /api/v1/status → version/uptime/active-runs/configured-agents
  - conductor task status <id>: GET /api/v1/tasks/{id} → task detail table
  - conductor task stop <id>: DELETE /api/v1/tasks/{id} → stops all running runs
  - 11 new tests in commands_test.go (status, task stop, formatUptime)
```

### Task 3: Check and update cli-reference.md

First read:
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md` lines 90-180

Verify that the conductor commands added in session #33 are accurately documented.
Fix any inaccuracies based on the actual implementation in:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/status.go`
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go`

If the docs are already accurate (they may be since the same commit updated them), just confirm and skip.

### What NOT to do

- Do NOT modify test files
- Do NOT add golangci-lint configuration
- Do NOT refactor existing code
- Only add/update documentation and MEMORY.md

### Quality Gates

```bash
# In /Users/jonnyzzz/Work/conductor-loop:
go build ./...
go test ./internal/... ./cmd/...
go test -race ./internal/... ./cmd/...
```

All must pass. If `go test -race` finds a race condition, fix it before creating the DONE file.

### Commit Format

If you change any tracked files (not MEMORY.md which is outside the repo):
```
chore: session #33 housekeeping - verify race-free and update docs
```

If nothing needs changing in the repo (docs already correct), make a minimal change like
updating a comment or adding a trailing note to MESSAGE-BUS.md to confirm the race check passed.

## DONE File

When all quality gates pass, create the file:
`$JRUN_TASK_FOLDER/DONE`

(The JRUN_TASK_FOLDER environment variable is set to your task directory automatically.)
