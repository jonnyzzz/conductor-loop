# Task: Add Message Bus Rotation to GC Command (ISSUE-016)

## Context

You are implementing a solution for ISSUE-016 in the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

The project is a Go-based multi-agent orchestration framework. The message bus stores messages in append-only files (`TASK-MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`). Over time these files grow and can become large.

**Current state:**
- `run-agent gc` command exists in `cmd/run-agent/gc.go`
- It cleans up old run directories
- It does NOT touch message bus files

**What to implement:**
Add message bus rotation support to the `run-agent gc` command.

## Requirements

### 1. New `--rotate-bus` flag to `run-agent gc`

When `--rotate-bus` is specified, the gc command should also scan for message bus files and rotate any that exceed a size threshold.

### 2. New `--bus-max-size` flag (default: 10MB)

Controls the size threshold above which a message bus file gets rotated. Format: `10MB`, `5MB`, `100KB`, etc.

### 3. Rotation Logic

When a message bus file exceeds the threshold:
1. Rename it to `<filename>.<YYYYMMDD-HHMMSS>.archived` in the same directory
2. The original filename now becomes empty/non-existent, so new messages go to a fresh file
3. Report: "Rotated TASK-MESSAGE-BUS.md (15.3MB → archived)"

### 4. Scope

Rotation should scan:
- Each `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`
- Each `<root>/<project>/PROJECT-MESSAGE-BUS.md`
- Respects `--project` filter if set
- Respects `--dry-run` flag (reports what would be rotated, does NOT rotate)

### 5. Tests

Add tests to `cmd/run-agent/gc_test.go`:
- Test that a large bus file gets rotated when `--rotate-bus` is set
- Test that a small bus file does NOT get rotated
- Test that `--dry-run` reports rotation without doing it
- Test that `--bus-max-size` threshold is respected
- At least 4 new tests

## Implementation Steps

1. Read and understand `cmd/run-agent/gc.go` and `cmd/run-agent/gc_test.go`
2. Add the two new CLI flags
3. Add bus file scanning logic (walk the run directory tree looking for `*MESSAGE-BUS.md` files)
4. Implement rotation function (rename with timestamp suffix)
5. Parse `--bus-max-size` (parse "10MB" → bytes)
6. Wire everything together
7. Add tests
8. Run: `go build ./... && go test ./cmd/run-agent/...`

## Quality Gates

- `go build ./...` passes
- `go test ./cmd/run-agent/...` passes
- `go test -race ./cmd/run-agent/...` passes
- All 13 original test packages still pass: `go test ./internal/... ./cmd/...`

## When Done

Create a `DONE` file in your task directory (`$JRUN_TASK_FOLDER/DONE`) to signal completion.

Write your output summary to `$JRUN_RUN_FOLDER/output.md`.
