# Task: Implement `run-agent gc` Command (ISSUE-015)

## Context

You are an implementation agent for the Conductor Loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

Read these files first (absolute paths):
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — conventions, commit format, code style
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md — tool paths, build/test commands
- /Users/jonnyzzz/Work/conductor-loop/ISSUES.md — ISSUE-015 description

## Task

Implement a `run-agent gc` subcommand that cleans up old run directories.

### ISSUE-015 Background

Each agent run creates a directory under `<root>/<project>/<task>/` (e.g., `runs/conductor-loop/task-xxx/run_YYYYMMDD-HHMMSS-PID/`).
No cleanup mechanism exists. A task with 100 restarts = 100 run directories. Disk usage grows indefinitely.

### Requirements

1. **New subcommand**: `run-agent gc` with the following flags:
   - `--root string` — root directory (default: `./runs` if RUNS_DIR env not set)
   - `--older-than duration` — delete runs older than this duration (default: `168h` = 7 days)
   - `--dry-run` — print what would be deleted without deleting (default: false)
   - `--project string` — limit gc to a specific project (optional, default: all projects)
   - `--keep-failed` — keep runs with non-zero exit codes (default: false)

2. **Run discovery**: Scan `<root>/<project>/<task>/run_*` directories. Each run dir has a `run-info.yaml` file (see internal/storage/runinfo.go for schema).

3. **Age check**: Use the run directory name (which encodes timestamp as `run_YYYYMMDD-HHMMSS-PID`) OR the `start_time` field in `run-info.yaml` for age calculation.

4. **Safety**: Only delete runs where `status` is `completed` or `failed` in `run-info.yaml`. Never delete runs with `status: running` (active runs). If `run-info.yaml` is missing, skip the run (may be active).

5. **Output**: Print what's being deleted (or would be deleted in --dry-run mode). Final summary: `Deleted N runs, freed X MB`.

6. **Tests**: Add table-driven unit tests in `cmd/run-agent/gc_test.go` (or wherever appropriate).

### Implementation Guide

1. Look at existing code structure:
   - `cmd/run-agent/main.go` — how subcommands are registered
   - `internal/storage/runinfo.go` — RunInfo struct and ReadRunInfo function
   - `internal/storage/storage.go` — how run directories are structured
   - `internal/storage/layout.go` — directory layout functions

2. Create `cmd/run-agent/gc.go` with the `gc` command

3. Register it in `cmd/run-agent/main.go` like other subcommands

4. The directory layout is: `<root>/<project_id>/<task_id>/<run_id>/`
   - run_id format: `run_YYYYMMDD-HHMMSS[0-9]+-<pid>` (see internal/storage/storage.go:NewRunDir())

### Quality Requirements

1. Run `go build ./...` — must pass
2. Run `go test ./...` — all tests must pass
3. Run `go test -race ./cmd/run-agent/` — no races
4. Follow commit format from AGENTS.md: `feat(runner): add run-agent gc command for run cleanup`

### Important Notes

- NEVER skip tests. Fix the code if tests fail.
- Use absolute paths for all file references
- Follow AGENTS.md code style (Go 1.24.0+, table-driven tests, lowercase error messages)
- Keep the implementation simple — no premature abstractions
- The gc command is purely local (no REST API changes needed)

After completing the implementation, write a summary of all files changed to stdout.
