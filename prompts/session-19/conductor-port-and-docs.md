# Task: Fix Port Configuration and Update Docs

## Context

This is the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
You are a Go developer doing housekeeping and documentation improvements.

## Issue

There is a port inconsistency in the project:
- `bin/conductor` (conductor server) uses port **8080** by default (from config)
- `bin/run-agent serve` uses port **14355** by default
- The web UI in `web/src/app.js` hardcodes `http://localhost:8080` for file:// mode
  This is correct for conductor (8080) but confusing for run-agent serve users (14355)

## Goals

### 1. Fix conductor server to have explicit default port in main.go

In `cmd/conductor/main.go`, the `runServer` function uses `apiConfig.Host` and `apiConfig.Port`
from the loaded config. But if no config is loaded, `apiConfig` is zero-value (empty host, port=0).

Check `internal/api/server.go` to see how it handles the port. If port=0, the server likely
binds to a random port or fails.

Add defaults: if `apiConfig.Port == 0`, set it to `8080`. If `apiConfig.Host == ""`, set it to `"0.0.0.0"`.
This makes the conductor server consistent.

### 2. Add --port and --host flags to conductor binary

In `cmd/conductor/main.go`, add `--host` (default `0.0.0.0`) and `--port` (default `8080`) CLI flags,
similar to how `run-agent serve` has them. The CLI flags should override config file values.

### 3. Update the web UI comment to document both servers

In `web/src/app.js`, update the comment near the `API_BASE` constant to clarify:
- When opened via file://, connects to conductor server on port 8080
- When served by run-agent serve, use relative paths automatically
- run-agent serve defaults to 14355

### 4. Update docs

In `docs/user/cli-reference.md` (or similar), ensure the conductor server port is documented.
Check what docs exist and make sure they reflect the actual defaults.

Also check Instructions.md at project root and ensure it documents the correct port.

## Quality gates (MUST pass before writing DONE file)

1. `go build ./...` must pass
2. `go test ./cmd/conductor/...` must pass
3. `go test ./internal/api/...` must pass

## Completion

When done, write a DONE file to the TASK_FOLDER directory.
Commit all changes with message: `fix(conductor): add --host/--port flags and fix default port config`
