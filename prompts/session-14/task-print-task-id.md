# Task: Print Auto-Generated Task ID in run-agent Commands

## Background

When the user runs `run-agent task` or `run-agent job` without `--task`, the CLI
auto-generates a task ID (e.g., `task-20260220-201530-abc123`). The user doesn't
currently see what ID was generated, making it hard to:
- Reference the task in subsequent commands
- Find the run directory
- Look up status via API

## Goal

Print the task ID to stdout when it's auto-generated (not provided by user), so
the user can reference it later.

## Changes Required

### `cmd/run-agent/main.go`

In `newTaskCmd()` RunE function, after `resolveTaskID()` when `taskID` was originally
empty (auto-generated), print the task ID:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    projectID = strings.TrimSpace(projectID)
    originalTaskID := strings.TrimSpace(taskID)
    if projectID == "" {
        return fmt.Errorf("project is required")
    }
    var err error
    taskID, err = resolveTaskID(originalTaskID)
    if err != nil {
        return err
    }
    // Print auto-generated task ID so user can reference it
    if originalTaskID == "" {
        fmt.Fprintf(cmd.OutOrStderr(), "task: %s\n", taskID)
    }
    ...
```

Same pattern for `newJobCmd()`.

**Important**: Use `cmd.OutOrStderr()` (stderr) not stdout, so the task ID
message doesn't pollute stdout output that might be piped.

## Test Requirements

Add a test in `cmd/run-agent/main_test.go` that:
- Runs the task command with `--task` provided: verifies NO task ID line printed
- Runs the task command without `--task`: verifies a task ID line IS printed to stderr

Look at existing tests in `cmd/run-agent/main_test.go` for patterns.

## Verification

```bash
go build -o /tmp/run-agent-test ./cmd/run-agent

# Test: auto-generated ID is printed
/tmp/run-agent-test task --project test --agent claude --prompt "hello" --root /tmp 2>&1 | head -5
# Should show: task: task-20260220-HHMMSS-xxxxx

# Test: explicit ID is NOT printed
/tmp/run-agent-test job --project test --task task-20260220-201530-test --agent claude --prompt "hello" --root /tmp 2>&1 | head -5
# Should NOT show a "task: ..." line

go test ./cmd/run-agent/
go build ./...
go test ./...
```

## Commit Format

```
feat(cli): print auto-generated task ID to stderr for run-agent task/job

When --task is omitted and a task ID is auto-generated, print the ID to
stderr so users can reference it in subsequent commands or API calls.
```

## Working Directory

All paths are absolute. CWD: `/Users/jonnyzzz/Work/conductor-loop`

## Notes

- Use `cmd.OutOrStderr()` (NOT stdout) for the message
- The task ID should be printed before `RunTask/RunJob` returns (so user sees it early)
- Only print when auto-generated, not when user provides explicit --task
