# Task: Fix TASK_FOLDER Environment Variable Missing from Agent Processes

## Context

The conductor-loop project is at /Users/jonnyzzz/Work/conductor-loop.

**Dog-food bug discovered:** When `run-agent job` launches a Claude/Codex/Gemini agent, the prompt preamble includes `TASK_FOLDER=<path>` as TEXT, but `TASK_FOLDER` is NOT set as an actual shell environment variable for the agent process.

This means when agents try to run `touch "$TASK_FOLDER/DONE"`, the `$TASK_FOLDER` shell variable is unset, so the DONE file is created in the current working directory (cwd) instead of the correct task directory. During session #15, this created a stray DONE file at `/Users/jonnyzzz/Work/conductor-loop/DONE`.

**Note:** For `run-agent job` specifically, DONE files are not required for job completion (the job completes when the agent process exits). But having TASK_FOLDER set as an env var is still valuable for agents that use it in shell operations.

## Root Cause

In `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go`, lines 127-133:
```go
envOverrides := map[string]string{
    "JRUN_PROJECT_ID": projectID,
    "JRUN_TASK_ID":    taskID,
    "JRUN_ID":         runID,
    "JRUN_PARENT_ID":  parentRunID,
    "RUNS_DIR":        runsDir,
    "MESSAGE_BUS":     busPath,
}
```

`TASK_FOLDER` and `RUN_FOLDER` are NOT in this map, even though they're mentioned in the prompt preamble.

Looking at the prompt preamble in `/Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go`:
```go
fmt.Fprintf(&b, "TASK_FOLDER=%s\n", params.TaskDir)
fmt.Fprintf(&b, "RUN_FOLDER=%s\n", params.RunDir)
```
These are written as text in the prompt but not set as env vars.

## What to Implement

In `internal/runner/job.go`, add `TASK_FOLDER` and `RUN_FOLDER` to the `envOverrides` map:

```go
envOverrides := map[string]string{
    "JRUN_PROJECT_ID": projectID,
    "JRUN_TASK_ID":    taskID,
    "JRUN_ID":         runID,
    "JRUN_PARENT_ID":  parentRunID,
    "RUNS_DIR":        runsDir,
    "MESSAGE_BUS":     busPath,
    "TASK_FOLDER":     taskDir,   // ADD THIS
    "RUN_FOLDER":      runDir,    // ADD THIS
}
```

Note: `taskDir` and `runDir` are already available at this point in the code (they're computed earlier in `runJob()`). But these are relative paths at this point — you should use the absolute paths. Check whether `runDirAbs` is available at this point in the code, or use `taskDir` directly (which should already be absolute since it comes from `resolveTaskDir`).

Check the code flow carefully in `runJob()` to understand which variables are available at lines 127-133 and use the correct ones (absolute paths).

## Also Add Tests

Add a test in `internal/runner/env_contract_test.go` to verify:
- `TASK_FOLDER` env var is set to the task directory in the spawned agent process
- `RUN_FOLDER` env var is set to the run directory

The existing tests in `env_contract_test.go` show the pattern for testing env var injection.

## Clean Up Stray Files

Also delete the stray DONE file at `/Users/jonnyzzz/Work/conductor-loop/DONE`:
```bash
rm /Users/jonnyzzz/Work/conductor-loop/DONE
```

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Build must pass
go build ./...

# All tests must pass
go test ./...

# Race detector
go test -race ./internal/runner/
```

## Files to Change

- `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` — add TASK_FOLDER and RUN_FOLDER to envOverrides
- `/Users/jonnyzzz/Work/conductor-loop/internal/runner/env_contract_test.go` — add tests

## Commit Format

```
fix(runner): inject TASK_FOLDER and RUN_FOLDER as environment variables

Previously these were only in the prompt preamble text but not
set as env vars for agent subprocesses. This caused agents using
`touch "$TASK_FOLDER/DONE"` to create DONE files in the cwd instead
of the correct task directory.
```

## Signal Completion

When done, create the DONE file at the TASK_FOLDER env var location:
```bash
# $TASK_FOLDER should now be set correctly after this fix!
touch "$TASK_FOLDER/DONE"
```
