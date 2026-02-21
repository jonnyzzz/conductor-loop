# Task: Add `--follow` flag to `run-agent job`

## Goal

Add a `--follow` flag to the `run-agent job` command that streams the agent's output in real-time while waiting for job completion. This eliminates the need for the operator to run a separate `run-agent output --follow` command.

## Context

- CWD: /Users/jonnyzzz/Work/conductor-loop
- Project: conductor-loop
- Working binary: ./bin/run-agent

### Current behavior

`run-agent job --agent claude --project P --root ./runs --prompt-file prompt.md` runs the job synchronously (blocking until done). The operator gets no output feedback during execution unless they separately run `run-agent output --follow`.

### Desired behavior with `--follow`

```bash
./bin/run-agent job --agent claude --project P --root ./runs --prompt-file prompt.md --follow
```

This should:
1. Dispatch the job (print `task: <id>` to stderr as usual when auto-generated)
2. Immediately start streaming agent output (agent-stdout.txt) to stdout in real-time
3. Block until the job completes
4. Return exit code 0 if the agent succeeded, non-zero if it failed

## Implementation Plan

### 1. Add `--follow` flag to `newJobCmd()` in `cmd/run-agent/main.go`

```go
var follow bool
cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream output in real-time while job runs")
```

### 2. When `--follow` is set:

Use `runner.AllocateRunDir()` to pre-create the run directory BEFORE starting the job. This gives us the run directory path upfront so we can follow it immediately.

```go
// In the RunE func, when follow == true:
rootDir := opts.RootDir
if rootDir == "" {
    if v := os.Getenv("RUNS_DIR"); v != "" {
        rootDir = v
    } else {
        rootDir = "./runs"
    }
}
runsDir := filepath.Join(rootDir, projectID, taskID, "runs")
if err := os.MkdirAll(runsDir, 0755); err != nil {
    return fmt.Errorf("create runs dir: %w", err)
}
runID, runDir, err := runner.AllocateRunDir(runsDir)
if err != nil {
    return fmt.Errorf("allocate run dir: %w", err)
}
_ = runID
opts.PreallocatedRunDir = runDir
```

### 3. Start the job in a goroutine and follow output:

```go
jobDone := make(chan error, 1)
go func() {
    jobDone <- runner.RunJob(projectID, taskID, opts)
}()

// Follow output from the pre-allocated run directory
if err := followOutput(runDir, ""); err != nil {
    // followOutput returned (run finished), wait for job goroutine
}

return <-jobDone
```

### Key integration points:
- `runner.AllocateRunDir(runsDir string) (runID, runDir string, err error)` — public function in `internal/runner/orchestrator.go`
- `runner.JobOptions.PreallocatedRunDir string` — already exists in `internal/runner/job.go`
- `followOutput(runDir, file string) error` — already exists in `cmd/run-agent/output.go`

### 4. Add tests in `cmd/run-agent/commands_test.go` or a new `job_follow_test.go`

Test that:
- `--follow` flag is registered and recognized
- With `--follow`, the command pre-allocates a run dir and sets `PreallocatedRunDir`
- Integration: `--follow` with a fast-completing job streams output correctly

Look at existing tests in `cmd/run-agent/output_follow_test.go` and `cmd/run-agent/commands_test.go` for patterns.

## Quality Requirements

1. `go build ./...` must pass
2. `go test -race ./internal/... ./cmd/...` must pass (ALL 15 packages)
3. `ACCEPTANCE=1 go test ./test/acceptance/...` must pass (ALL 4 scenarios)
4. No data races introduced
5. The implementation must handle edge cases:
   - Job completes very quickly before follow starts (already handled by `followOutput` which checks status first)
   - `followOutput` times out due to no data (not a failure)
   - Job fails (non-zero exit code) - should propagate via `RunJob` return value

## Important Notes

- Read `cmd/run-agent/output.go` carefully - `followOutput(runDir, file string)` already handles the "run already complete" case gracefully
- Read `internal/runner/orchestrator.go` lines 128-132 for `AllocateRunDir`
- Read `internal/runner/job.go` for `JobOptions.PreallocatedRunDir` usage
- The `--follow` flag should be short: `-f` is already used by `output --follow`, so use it here too
- After `followOutput` returns, ALWAYS drain from `jobDone` channel to avoid goroutine leaks
- Write output.md to /Users/jonnyzzz/Work/conductor-loop/runs for this task (path provided by TASK_FOLDER env)

## Commit Format

```
feat(cli): add --follow flag to run-agent job for real-time output streaming
```

Follow the commit format in AGENTS.md.
