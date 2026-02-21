# Task: Make --bus Flag Optional Using $MESSAGE_BUS Env Var

## Context

The conductor-loop project is at /Users/jonnyzzz/Work/conductor-loop.

The `run-agent bus post` and `run-agent bus read` commands currently require `--bus <path>` to specify the message bus file. This is:

```bash
# Current (annoying)
run-agent bus post --bus /path/to/TASK-MESSAGE-BUS.md --type INFO --body "hello"

# Desired (natural for sub-agents)
run-agent bus post --type INFO --body "hello"  # uses $MESSAGE_BUS automatically
```

**The key insight:** When `run-agent job` launches an agent subprocess (e.g. claude), it already injects the `MESSAGE_BUS` environment variable pointing to the task's message bus file (see `internal/runner/job.go`). So sub-agents already have `$MESSAGE_BUS` set. Making `--bus` optional when this env var is present would significantly reduce friction.

The relevant file is: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/bus.go`

## What to Implement

In `cmd/run-agent/bus.go`:

1. **In `newBusPostCmd()` RunE function**, change the `--bus` check to:
   ```go
   if busPath == "" {
       busPath = os.Getenv("MESSAGE_BUS")
   }
   if busPath == "" {
       return fmt.Errorf("--bus is required (or set MESSAGE_BUS env var)")
   }
   ```

2. **In `newBusReadCmd()` RunE function**, apply the same change:
   ```go
   if busPath == "" {
       busPath = os.Getenv("MESSAGE_BUS")
   }
   if busPath == "" {
       return fmt.Errorf("--bus is required (or set MESSAGE_BUS env var)")
   }
   ```

3. **Update the flag descriptions** to mention the env var fallback:
   - `bus post --bus`: `"path to message bus file (uses MESSAGE_BUS env var if not set)"`
   - `bus read --bus`: `"path to message bus file (uses MESSAGE_BUS env var if not set)"`

4. **Add tests** in `cmd/run-agent/commands_test.go` to verify:
   - `bus post` uses `MESSAGE_BUS` env var when `--bus` is not provided
   - `bus read` uses `MESSAGE_BUS` env var when `--bus` is not provided
   - `bus post` still fails with error when neither `--bus` nor `MESSAGE_BUS` is set

## Important: Check Existing Tests

First read the existing tests in `cmd/run-agent/commands_test.go` to understand the test pattern and ensure you don't break them.

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Build must pass
go build ./...

# All tests must pass
go test ./cmd/run-agent/ -v

# Race detector must pass
go test -race ./cmd/run-agent/
```

## Files to Change

- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/bus.go` — add env var fallback
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/commands_test.go` — add tests for env var fallback

## Also Update Instructions.md

The file `/Users/jonnyzzz/Work/conductor-loop/Instructions.md` contains this stale comment:
```
## Message Bus Commands (planned; not implemented yet)

The current `run-agent` binary exposes only `task` and `job` commands. The `bus` subcommands below are planned but not implemented yet. Until they exist, use the REST API served by `run-agent serve`:
```

This is WRONG — bus subcommands ARE implemented (since session #7). Update the section header and description to reflect the actual state. Change:
- "planned; not implemented yet" → remove this note
- Remove "Until they exist, use the REST API" sentence
- Add the correct usage examples showing `--bus` and `$MESSAGE_BUS`

## Commit Format (from AGENTS.md)

Two commits:
1. `feat(cli): use MESSAGE_BUS env var as fallback for bus post/read --bus flag`
2. `docs: update Instructions.md bus subcommands section`

Or combine into one commit if appropriate.

## Signal Completion

When done, create the DONE file:
```bash
touch "$TASK_FOLDER/DONE"
```
