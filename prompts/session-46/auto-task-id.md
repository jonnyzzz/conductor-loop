# Task: Make --task optional in conductor job submit

## Context

You are working on the conductor-loop project (Go-based multi-agent orchestration).
CWD: /Users/jonnyzzz/Work/conductor-loop

## Problem

`conductor job submit` requires `--task <id>` to be explicitly provided:

```go
// cmd/conductor/job.go line 97
_ = cmd.MarkFlagRequired("task")
```

However, `run-agent job` auto-generates a valid task ID when `--task` is omitted.
The task ID format is: `task-<YYYYMMDD>-<HHMMSS>-<slug>` (6-char random hex slug).

Users should be able to run:
```bash
conductor job submit --project my-project --agent claude --prompt "Hello" --wait
# Instead of requiring:
conductor job submit --project my-project --task task-20260221-120000-abc123 --agent claude --prompt "Hello"
```

## Task

1. Make `--task` optional in `conductor job submit` in `cmd/conductor/job.go`
2. When `--task` is omitted, auto-generate a valid task ID in the format `task-YYYYMMDD-HHMMSS-<6-char-random-hex>`
3. When the auto-generated or explicitly provided task ID is used, echo it to stdout so users know the task ID
4. Update the help text/description to indicate `--task` is now optional
5. Add/update tests in `cmd/conductor/commands_test.go` or a new test file

## Implementation Details

### Task ID Format
```
task-20260221-120000-a1b2c3
        ^          ^       ^
   YYYYMMDD    HHMMSS  6-char random hex
```

Use `time.Now().UTC()` for timestamp and `crypto/rand` or `math/rand` for 6-char hex suffix.
Note: The format uses 6-digit time (HHMMSS) not 4-digit.

### Changes Required

1. **cmd/conductor/job.go** (`newJobSubmitCmd` function):
   - Remove `_ = cmd.MarkFlagRequired("task")` (line ~97)
   - Add task ID generation logic in `RunE`:
     ```go
     if taskID == "" {
         taskID = generateTaskID()
     }
     ```
   - Add `generateTaskID()` helper function
   - Update usage/short description to note task is optional

2. **Tests** (add to existing `cmd/conductor/commands_test.go` or new `job_test.go`):
   - Test that `--task` omission auto-generates valid task ID format
   - Test that explicit `--task` is used as-is
   - Test `generateTaskID()` format validation

### Important Notes

- The generated task ID must match: `^task-\d{8}-\d{6}-[0-9a-f]{6}$`
- Print the task ID when submitting: `Task created: <task-id>, run_id: <run-id>` (already in jobSubmit)
- If `--json` is set, the response JSON already includes task_id so no extra printing needed
- Use `crypto/rand` for the random suffix (safer than math/rand)
- No external dependencies; only stdlib

## Quality Gates

1. `go build ./cmd/conductor/` PASS
2. `go test -race ./cmd/conductor/` PASS (all existing tests + new ones)
3. Manually verify: `./bin/conductor job submit --project test --agent claude --prompt "hello"` generates a valid task ID (you cannot actually run this since you don't have a server, but verify the code path)
4. Update docs/user/cli-reference.md: mark `--task` as optional in the `conductor job submit` section

## Files to Modify

- `cmd/conductor/job.go` — primary change
- `cmd/conductor/commands_test.go` OR new `cmd/conductor/job_test.go` — tests
- `docs/user/cli-reference.md` — update flag table to show `--task` as optional

## Commit

After all changes are made and tests pass, commit:
```
feat(cli): make --task optional in conductor job submit (auto-generate task ID)
```

Format: `feat(cli): <description>` — single commit, all changes together.
