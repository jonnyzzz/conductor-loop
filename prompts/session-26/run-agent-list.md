# Sub-Agent Task: Add `run-agent list` Command (Session #26)

## Role
You are an implementation agent. Your CWD is /Users/jonnyzzz/Work/conductor-loop.

## Task
Add a `run-agent list` command that lists projects, tasks, and runs from the filesystem.

## Context
Currently there is no CLI command to list what tasks exist for a project. The operator
must either start the HTTP server or manually browse the filesystem. This command fills
that gap for quick status checks without a running server.

The filesystem layout is:
```
<root>/<project>/<task>/runs/<run_id>/
<root>/<project>/<task>/runs/<run_id>/run-info.yaml  ← status, exit code, timestamps
<root>/<project>/<task>/runs/<run_id>/pid.txt         ← present while running
<root>/<project>/<task>/TASK-MESSAGE-BUS.md
<root>/<project>/<task>/DONE                          ← present when task completed
```

The run-info.yaml has this structure:
```yaml
run_id: run-20260221-...
project_id: my-project
task_id: task-20260221-...
status: "running"|"completed"|"failed"|"crashed"
started_at: "2026-02-21T..."
finished_at: "2026-02-21T..."
exit_code: 0
agent_type: claude
```

Read existing code in:
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/ — existing commands (gc.go, output.go, stop.go, validate.go)
- /Users/jonnyzzz/Work/conductor-loop/internal/storage/ — ReadRunInfo, ListRuns, etc.
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go — how commands are registered

## Required Implementation

### File: cmd/run-agent/list.go

Implement a `list` command with these behaviors:

```
run-agent list [--root <dir>]
    Lists all projects in the root directory.
    Output: project names, one per line.

run-agent list --project <project-id> [--root <dir>]
    Lists all tasks for a project.
    Output: table with columns: TASK_ID, RUNS, LATEST_STATUS, DONE
    DONE column shows "DONE" if DONE file exists, "-" otherwise.
    LATEST_STATUS is the status from the most recent run.

run-agent list --project <project-id> --task <task-id> [--root <dir>]
    Lists all runs for a task.
    Output: table with columns: RUN_ID, STATUS, EXIT_CODE, STARTED, DURATION
    DURATION is finished_at - started_at (or "running" if not finished).
```

Flags:
- `--root string` — root directory (default: ./runs or JRUN_RUNS_DIR env var)
- `--project string` — project ID (optional, shows tasks if set)
- `--task string` — task ID (requires --project, shows runs if set)
- `--json` — output as JSON instead of table

### Implementation Notes:
1. Read the JRUN_RUNS_DIR env var as default root (same as gc.go does)
2. Use `storage.ReadRunInfo()` to read run metadata
3. List runs by scanning the filesystem (use filepath.Glob or os.ReadDir)
4. Sort tasks and runs by name (alphabetical = chronological for task-YYYYMMDD-HHMMSS-* format)
5. Show the most recent run's status for each task
6. A task is "DONE" if a DONE file exists at <root>/<project>/<task>/DONE
7. Handle missing run-info.yaml gracefully (show "unknown" status)

### File: cmd/run-agent/list_test.go

Write tests covering:
1. List projects (no flags) — shows all project dirs
2. List tasks for a project (--project) — shows task rows
3. List runs for a task (--project + --task) — shows run rows
4. Empty project (no tasks) — shows empty table
5. Missing project dir — error
6. JSON output flag works
7. DONE file detection works

Use table-driven tests. Create temp directories with real run-info.yaml files.

### Register the command in main.go
Add the list command to the root command in cmd/run-agent/main.go (follow the pattern of existing commands).

## Code Style
- Follow the existing code style in cmd/run-agent/ (cobra commands, similar to gc.go)
- Use `tabwriter` for table formatting (see how other Go CLI tools do it, or use simple Sprintf)
- Keep file under 300 lines
- Use `errors.Wrap()` from github.com/pkg/errors for error context

## Steps
1. Read the existing source files listed above to understand patterns
2. Implement cmd/run-agent/list.go
3. Implement cmd/run-agent/list_test.go
4. Register in main.go
5. Run: go build ./... (must pass)
6. Run: go test ./cmd/run-agent/ (must pass, including new tests)
7. Run: go test ./internal/... ./cmd/... (all must pass)
8. Test manually: ./bin/run-agent list --root /Users/jonnyzzz/Work/conductor-loop/runs
9. Test manually: ./bin/run-agent list --project conductor-loop --root /Users/jonnyzzz/Work/conductor-loop/runs
10. Commit: git add cmd/run-agent/list.go cmd/run-agent/list_test.go cmd/run-agent/main.go
11. Commit message: `feat(cli): add run-agent list command for project/task/run enumeration`
12. Create DONE file: touch /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/${JRUN_TASK_ID}/DONE

## Quality Requirements
- go build ./... must pass
- go test ./cmd/run-agent/ must pass
- All new tests must pass
- Command must work correctly against the real runs/ directory
