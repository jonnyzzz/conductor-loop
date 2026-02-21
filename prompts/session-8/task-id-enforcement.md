# Task: Enforce Task ID Format in run-agent CLI

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go
- /Users/jonnyzzz/Work/conductor-loop/internal/storage/storage.go
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout-QUESTIONS.md

## Human Decision (from storage-layout-QUESTIONS.md Q4)

> Should task IDs be enforced to follow `task-<timestamp>-<slug>` by the CLI, or remain caller-defined?
> **Answer**: yes, enforced, and fully controlled by the run-agent binary. We are assertive and fail if that does not work as expected in the specs/code.

## What to Implement

### 1. Task ID Format

Task IDs must follow the pattern: `task-<timestamp>-<slug>`
- `<timestamp>`: 8-digit date + 6-digit time (e.g., `20260220-153045`)
- `<slug>`: lowercase alphanumeric + hyphens, 3-50 chars (e.g., `my-feature`, `bug-fix`)
- Example valid ID: `task-20260220-153045-my-feature`
- Example invalid IDs: `my-task`, `task-foo`, `random-string`

### 2. Auto-generation

When the user does NOT provide `--task` flag, auto-generate a task ID:
- Format: `task-<YYYYMMDD-HHMMSS>-<random-slug>`
- Random slug: 6-character lowercase alphanumeric (e.g., `a3f9bc`)
- Example: `task-20260220-153045-a3f9bc`

### 3. Validation

When the user DOES provide `--task`, validate it:
- Must match regex: `^task-\d{8}-\d{6}-[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`
- If invalid: print clear error message and exit non-zero

### 4. Implementation Plan

1. Add `ValidateTaskID(taskID string) error` to `internal/storage/` package
2. Add `GenerateTaskID(slug string) string` to `internal/storage/` package
3. In `cmd/run-agent/main.go`: if `--task` is empty, call `GenerateTaskID` with empty slug (generates random)
4. In `cmd/run-agent/main.go`: if `--task` is provided, call `ValidateTaskID` and fail if invalid
5. Apply same logic to both `task` and `job` subcommands

### 5. Tests Required

In `internal/storage/` package:
- TestValidateTaskID: table-driven tests for valid and invalid task IDs
- TestGenerateTaskID: verify generated IDs pass validation and have correct format

In `cmd/run-agent/` package (CLI tests):
- TestJobAutoGeneratesTaskID: when --task is empty, task ID is auto-generated and valid
- TestJobRejectsInvalidTaskID: when --task is invalid format, exit non-zero

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./internal/storage/...` must pass
3. `go test ./cmd/run-agent/...` must pass
4. `go vet ./...` must pass

## Output

Write your changes to the files. Then create a file `/Users/jonnyzzz/Work/conductor-loop/runs/session8-task-id/output.md` with a summary of what was changed.

Do NOT create a DONE file — the orchestrator will handle that.

## Commit

Commit with message format: `feat(storage): enforce task-<timestamp>-<slug> ID format`
