# Task: Add Project/Task-Aware Flags to `bus read` and `bus post` Commands

You are an experienced Go developer working on the conductor-loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Goal

Enhance the `run-agent bus read` and `run-agent bus post` commands to accept `--project`, `--task`, and `--root` flags that auto-resolve the message bus file path from the project/task hierarchy.

## Problem

Currently, to read the task message bus for a task, users must specify the full path:
```bash
./bin/run-agent bus read --bus runs/conductor-loop/task-20260221-012534-cbdjw9/TASK-MESSAGE-BUS.md
```

This is cumbersome. After this feature, users can do:
```bash
./bin/run-agent bus read --project conductor-loop --task task-20260221-012534-cbdjw9 --root runs
```

Or for the project-level bus:
```bash
./bin/run-agent bus read --project conductor-loop --root runs
```

## Background

The message bus file path structure is:
- Task-level: `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`
- Project-level: `<root>/<project>/PROJECT-MESSAGE-BUS.md`

The root defaults to `./runs` or `$JRUN_RUNS_DIR` env var (similar to other commands).

## Implementation

Read these files first to understand the current implementation:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/bus.go` — current bus command (all of it)
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go` — to understand how root is handled in other commands
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list.go` — to see how `--root` and `--project` are handled elsewhere

### Changes to `cmd/run-agent/bus.go`

For BOTH `newBusReadCmd()` and `newBusPostCmd()`:

1. Add flags:
   ```
   --project string   project ID (used with --root to resolve bus path)
   --task string      task ID (used with --project and --root to resolve bus path; if omitted, reads project-level bus)
   --root string      root directory (default: JRUN_RUNS_DIR env var, then ./runs)
   ```

2. Add path resolution logic:
   - If `--project` is specified (with or without `--task`):
     - Compute root: use `--root` flag, else `$JRUN_RUNS_DIR` env var, else `./runs`
     - If `--task` is specified: bus path = `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`
     - If `--task` is NOT specified: bus path = `<root>/<project>/PROJECT-MESSAGE-BUS.md`
     - If `--bus` is ALSO specified, return an error: "cannot specify both --bus and --project"
   - If `--project` is NOT specified:
     - Fall back to existing `--bus` / `JRUN_MESSAGE_BUS` env var behavior

3. Keep backward compatibility: `--bus` flag still works as before when `--project` is not specified.

### Resolution order (for bus path)

```
Priority:
1. --project (+ optional --task) → auto-resolve path from project/task hierarchy
2. --bus flag
3. JRUN_MESSAGE_BUS env var
4. Error: bus path required
```

If both `--bus` and `--project` are specified, return a clear error message.

### Tests to add in `cmd/run-agent/bus_test.go` (or create if doesn't exist)

Add at least 6 tests:
1. TestBusReadWithProject — reads project-level bus (PROJECT-MESSAGE-BUS.md)
2. TestBusReadWithProjectAndTask — reads task-level bus (TASK-MESSAGE-BUS.md)
3. TestBusReadBusFlagAndProjectError — error when both --bus and --project specified
4. TestBusPostWithProject — posts to project-level bus
5. TestBusPostWithProjectAndTask — posts to task-level bus
6. TestBusRootDefaultsToRunsDir — root defaults to ./runs when not specified and JRUN_RUNS_DIR not set

Use table-driven tests where appropriate. Use temp directories for test isolation.

## Quality Requirements

Read `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md` for commit format and code style.

After implementation:
1. Run `go build ./...` — must pass
2. Run `go test ./cmd/run-agent/...` — must pass (all tests, including new ones)
3. Run `go test -race ./cmd/run-agent/...` — no data races
4. Verify manually:
   - Create a temp bus file structure, run `./bin/run-agent bus read --project ... --task ... --root ...`
   - Verify it reads the right file

Commit format:
```
feat(cli): add --project/--task/--root flags to bus read and bus post commands
```

Create the DONE file at `/Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/$TASK_ID/DONE` when complete.

## Constraints

- Only modify `cmd/run-agent/bus.go` and add test file `cmd/run-agent/bus_test.go`
- Do NOT modify any internal/ packages
- Do NOT modify any other cmd/run-agent/*.go files
- Keep backward compatibility with the existing `--bus` flag
- Match the coding style of other commands (see list.go, stop.go for reference)
- Document the new flags in the command's Help text
