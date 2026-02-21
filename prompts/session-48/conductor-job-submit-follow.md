# Task: Add `--follow` flag to `conductor job submit`

## Goal

Add a `--follow` flag to the `conductor job submit` command that after submitting the job, immediately starts streaming the task's log output via the conductor server's SSE endpoint. This eliminates the need to separately run `conductor task logs --follow`.

## Context

- CWD: /Users/jonnyzzz/Work/conductor-loop
- Project: conductor-loop
- Working binaries: ./bin/conductor, ./bin/run-agent

### Current behavior

`conductor job submit --project P --prompt-file prompt.md` submits the job to the conductor server and optionally waits (`--wait` flag). With `--wait`, it polls for completion and prints a summary when done. But the operator gets no output feedback during execution.

### Desired behavior with `--follow`

```bash
conductor job submit --project myproject --prompt-file prompt.md --follow
```

This should:
1. Submit the job to the conductor server (same as today)
2. Print the task ID and run ID
3. Immediately start streaming the task's agent output to stdout via the conductor server's SSE endpoint
4. Block until the task completes
5. Return exit code 0 if successful, non-zero if the server indicates failure

### When `--follow` and `--wait` are both set

`--follow` implies `--wait`. If both are set, use `--follow` behavior (streaming output is strictly better than silent polling).

## Implementation Plan

### 1. Locate the conductor job submit command

File: `cmd/conductor/job.go`

Look at the `newJobSubmitCmd()` function. It already has a `--wait` flag. Add `--follow bool` flag alongside it.

### 2. After job submission, when `--follow` is true:

Extract project and task ID from the submission response, then call `taskLogs()` with `follow=true`. This function already exists in `cmd/conductor/task_logs.go`.

```go
// After successful job submission:
if follow {
    // taskLogs streams the output until the task completes
    return taskLogs(cmd.OutOrStdout(), server, projectID, taskID, "" /* runID */, true /* follow */, 0 /* tail */)
}
```

### 3. Key integration points:

- `taskLogs(out io.Writer, server, project, taskID, runID string, follow bool, tail int) error` — in `cmd/conductor/task_logs.go`
- The submission response contains `task_id` and `project_id` — check the current response struct in `job.go`
- The SSE stream at `/api/projects/{p}/tasks/{t}/runs/{r}/stream?name=stdout` is already implemented

### 4. Wait for run to start

There's a timing issue: the job is just submitted and the run may not have started yet. The `resolveLatestRunID()` function in `task_logs.go` fetches the task detail and waits for a run. We may need to add a brief retry loop if no runs exist yet.

Look at how `taskLogs()` calls `resolveLatestRunID()` - it may already handle "no runs" gracefully, or we may need to add a retry.

If `resolveLatestRunID` fails with "no runs", add a retry loop (max 30s, 1s intervals) before calling `taskLogs`.

### 5. Add tests

Add tests in `cmd/conductor/task_logs_test.go` or a new test file. Test that:
- `--follow` flag is registered
- With `--follow`, after job submit, `taskLogs()` is called with follow=true

Look at existing tests in `cmd/conductor/task_logs_test.go` and `cmd/conductor/job.go` for patterns.

## Quality Requirements

1. `go build ./...` must pass
2. `go test -race ./internal/... ./cmd/...` must pass (ALL 15 packages)
3. `ACCEPTANCE=1 go test ./test/acceptance/...` must pass (ALL 4 scenarios)
4. No data races introduced
5. Edge cases to handle:
   - Job submission fails → return error immediately (no follow)
   - Task submitted but no run starts within 30s → timeout with error
   - Server connection drops during follow → `--follow` reconnects (already handled by taskLogs)
   - `--follow` and `--wait` both set → use follow behavior (stream output)

## Documentation

Update `docs/user/cli-reference.md` to document the `--follow` flag in the `conductor job submit` section.

## Important Notes

- Read `cmd/conductor/job.go` first to understand the current submit flow
- Read `cmd/conductor/task_logs.go` to understand the `taskLogs()` function
- The conductor server must be running for `conductor job submit` to work (this is already a requirement)
- The `--follow` flag should NOT conflict with `--json` output: if both are set, emit JSON for the submission result and then stream logs to stdout (or just disable `--json` when `--follow` is set - pick the simpler approach)
- Write output.md to /Users/jonnyzzz/Work/conductor-loop/runs for this task (path provided by TASK_FOLDER env)

## Commit Format

```
feat(cli): add --follow flag to conductor job submit for real-time log streaming
```

Follow the commit format in AGENTS.md.
