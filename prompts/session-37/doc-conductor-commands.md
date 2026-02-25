# Task: Update cli-reference.md for Conductor Commands (Sessions #33-#36)

## Context

You are a sub-agent working on the conductor-loop project. The conductor CLI has gained several
new commands in sessions #33-36 that are NOT yet documented in `docs/user/cli-reference.md`.

## What to Add

The following conductor commands are implemented but undocumented in `docs/user/cli-reference.md`:

### 1. `conductor task list` (added Session #34)
- File: cmd/conductor/task.go (newTaskListCmd)
- Usage: `conductor task list --project <id> [--server URL] [--json]`
- Calls GET /api/projects/{p}/tasks
- Shows TASK ID / STATUS / RUNS / LAST ACTIVITY table
- `--project` is required

### 2. `conductor task delete <task-id>` (added Session #35)
- File: cmd/conductor/task.go (newTaskDeleteCmd)
- Usage: `conductor task delete <task-id> --project <id> [--server URL] [--json]`
- Calls DELETE /api/projects/{p}/tasks/{t}
- Returns 204 (deleted), 409 (running runs exist), 404 (not found)
- `--project` is required

### 3. `conductor project list` (added Session #34)
- File: cmd/conductor/project.go (newProjectListCmd)
- Usage: `conductor project list [--server URL] [--json]`
- Calls GET /api/projects
- Shows PROJECT ID / TASKS / LAST ACTIVITY table

### 4. `conductor project stats` (added Session #35)
- File: cmd/conductor/project.go (newProjectStatsCmd)
- Usage: `conductor project stats --project <id> [--server URL] [--json]`
- Calls GET /api/projects/{p}/stats
- Shows: tasks count, runs breakdown (running/completed/failed/crashed), message bus files+size
- `--project` is required

### 5. `conductor job submit --prompt-file` (added Session #36)
- File: cmd/conductor/job.go
- The existing `conductor job submit` now supports `--prompt-file <path>` as alternative to `--prompt`
- `--prompt` and `--prompt-file` are mutually exclusive
- Errors: both set, neither set, file not found, empty file

## Instructions

1. Read `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md` to understand the current structure
2. Read the relevant source files to understand flags and behavior:
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go`
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/project.go`
   - `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/job.go`
3. Add documentation sections for all 5 items above following the existing style
4. For `conductor job submit`, add `--prompt-file` to the flags table and add an example
5. Run `go build -o bin/conductor ./cmd/conductor` to verify the commands exist as described
6. Verify the cli-reference.md is correct and consistent with the actual flags

## Quality

- Follow the existing documentation style exactly (headers, flag tables, example blocks)
- Do NOT change existing documentation content (only add new sections)
- All flags must match what the binary actually accepts

## Completion

Create the DONE file at: $JRUN_TASK_FOLDER/DONE
