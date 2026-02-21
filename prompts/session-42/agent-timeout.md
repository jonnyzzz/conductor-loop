# Task: Add Agent Run Timeout Support

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

This is a Go-based multi-agent orchestration framework. The task is to add timeout support
for agent job execution so that stuck agents can be automatically killed after a configured duration.

## Current State

- `go build ./...` passes
- All tests pass (`go test -race ./internal/... ./cmd/...`)
- Binaries: `bin/conductor` and `bin/run-agent`

## Task: Implement Agent Run Timeout

### Background

Currently when an agent hangs (e.g., waiting for user input, network issue), there's no way to
automatically kill it. A `--timeout` flag is needed so users can set a maximum run duration.

### Files to Modify

1. **`internal/runner/job.go`** - Core job runner
   - `JobOptions` struct: add `Timeout time.Duration` field
   - `runJob` function: when `opts.Timeout > 0`, create `ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)` and defer cancel
   - Pass the timeout context instead of `ctxOrBackground()` when calling `executeCLI`
   - After `executeCLI` returns, check if context was cancelled (err wraps context.DeadlineExceeded) and post an appropriate ERROR message to the message bus

2. **`internal/runner/task.go`** (or wherever `TaskOptions` is defined)
   - `TaskOptions` struct: add `Timeout time.Duration` field
   - Pass `opts.Timeout` to `JobOptions` when calling `runJob`

3. **`cmd/run-agent/main.go`** - CLI command
   - `newJobCmd()`: add `cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "maximum agent run duration (e.g. 30m, 2h); 0 means no limit")`
   - `newTaskCmd()`: add the same `--timeout` flag

4. **`docs/user/cli-reference.md`** - Documentation
   - Add `--timeout` flag description to `run-agent job` section
   - Add `--timeout` flag description to `run-agent task` section

### Implementation Notes

- `executeCLI` already takes `ctx context.Context` as first parameter
- `ctxOrBackground()` is currently used when no timeout; with timeout, pass the timeout ctx instead
- When context expires, the process is killed via the underlying `proc.Kill()` or `proc.Wait()` returns
- After timeout, log to message bus: `WARN: agent job timed out after X seconds`
- The run-info.yaml should have `status=timeout` (or `status=failed` with error_summary="timed out after Xs")
- ExitCode for timeout: use -1 (same as other process kill scenarios)

### Tests to Add

In `internal/runner/` or `cmd/run-agent/` test files:

1. Test that when `Timeout` is set to a very short duration (e.g., 100ms) and the agent is a
   slow process (sleep 10s), the job completes within timeout + reasonable overhead (e.g., 2s)
2. Test that after timeout, the message bus contains a timeout error message
3. Test that when `Timeout` is 0, there is no timeout (existing behavior)

Use the stub agent pattern from `test/acceptance/acceptance_test.go` - look at how it builds
and uses the `codex` stub binary with `ORCH_STUB_SLEEP_MS` env var.

### Quality Gates

After implementation:
- `go build ./...` must pass
- `go test -race ./internal/... ./cmd/...` must pass
- All existing tests must continue to pass
- Add at least 2 new tests for timeout behavior

## Implementation Steps

1. Read the relevant files first: `internal/runner/job.go`, `internal/runner/task.go` (find where TaskOptions is), `cmd/run-agent/main.go`
2. Understand the context flow in `runJob` and `executeCLI`
3. Implement the changes
4. Write tests
5. Update documentation
6. Run quality gates
7. Commit with message: `feat(runner): add --timeout flag for agent run duration limit`

## Working Directory

/Users/jonnyzzz/Work/conductor-loop

## Done Signal

Create the file `DONE` in the task directory when complete.
