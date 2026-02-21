# Task: Add `run-agent serve` and `run-agent bus` Commands

## Objective
Implement two new subcommands for the `run-agent` CLI:
1. `run-agent serve` — start the HTTP server (lightweight version of `conductor`)
2. `run-agent bus` — read/post messages to the message bus from CLI

## Required Reading
1. Read `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go` — existing CLI structure
2. Read `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go` — how the server is started
3. Read `/Users/jonnyzzz/Work/conductor-loop/internal/api/server.go` — `api.NewServer` and `api.Options`
4. Read `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go` — `AppendMessage`, `ReadMessages`
5. Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui-QUESTIONS.md` Q1 answer
6. Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools-QUESTIONS.md` Q1 answer

## Change 1: `run-agent serve` Command

Add a `serve` cobra command to `cmd/run-agent/main.go`.

### Command signature
```
run-agent serve [flags]

Flags:
  --host string    HTTP server host (default "127.0.0.1")
  --port int       HTTP server port (default 14355)
  --root string    run-agent root directory (optional)
  --config string  config file path (optional)
```

### Behavior
- Start the HTTP server using `api.NewServer` (same as conductor)
- Default: listen on `127.0.0.1:14355` (per monitoring-ui Q1 answer)
- No config file required — server works without agents configured (monitoring-only mode)
- Handle graceful shutdown on SIGINT/SIGTERM (same as conductor/main.go)
- Log startup message to stderr: `run-agent serve: listening on http://127.0.0.1:14355`

### Implementation Notes
- Reuse `runServer` logic from conductor — but this time it's inside `cmd/run-agent/main.go`
- The server should NOT fail if no config is provided — just start with no agents
- Import `internal/api` and `internal/config` packages
- Handle signal shutdown the same way as conductor's `runServer` function
- Add `cmd.AddCommand(newServeCmd())` to `newRootCmd()`

## Change 2: `run-agent bus` Command

Add a `bus` cobra command with two subcommands: `post` and `read`.

### Command: `run-agent bus post`
```
run-agent bus post [flags]

Flags:
  --bus string      path to message bus file (required)
  --type string     message type (default "INFO")
  --project string  project ID
  --task string     task ID
  --run string      run ID
  --body string     message body (reads from stdin if not provided and stdin is a pipe)
```

Behavior:
- Create/open the message bus at `--bus` path
- Post a message with the given fields
- Print the message ID to stdout: `msg_id: <id>`
- If `--body` is empty and stdin has data, read body from stdin

### Command: `run-agent bus read`
```
run-agent bus read [flags]

Flags:
  --bus string    path to message bus file (required)
  --tail int      print last N messages (default 20)
  --follow        watch for new messages (Ctrl-C to exit)
```

Behavior:
- Open the message bus at `--bus` path
- With `--tail N`: print the last N messages
- With `--follow`: continuously watch for new messages (poll every 500ms) and print them
- Print each message in a human-readable format:
  ```
  [2026-02-20 15:00:00] (INFO) This is the message body
  ```

### Implementation Notes
- Use `messagebus.NewMessageBus` to open the bus
- Use `bus.ReadMessages()` or equivalent to read messages (check the messagebus package API)
- Add `cmd.AddCommand(newBusCmd())` to `newRootCmd()`
- `newBusCmd()` should return a cobra command with `AddCommand(newBusPostCmd(), newBusReadCmd())`

## Tests
Add tests in `cmd/run-agent/main_test.go` (or a new file). Read the existing test to understand patterns.

Test cases:
1. `TestServeCmd_StartsServer` — start serve in a goroutine, verify health endpoint responds, then stop
2. `TestBusPostCmd_PostsMessage` — write a temp bus file path, run bus post, verify message written
3. `TestBusReadCmd_ReadMessages` — create bus with messages, run bus read --tail 5, verify output

## Quality Gates
- `go build ./...` passes
- `go test ./cmd/run-agent/` passes
- `go vet ./cmd/run-agent/` passes
- Rebuild the binary: `go build -o bin/run-agent ./cmd/run-agent`
- Verify: `./bin/run-agent --help` shows `serve` and `bus` commands
- Verify: `./bin/run-agent bus --help` shows `post` and `read` subcommands

## Commit Format
```
feat(run-agent): add serve and bus subcommands

- serve: start HTTP server on 127.0.0.1:14355 (configurable)
- bus post: post a message to a message bus file
- bus read: read/tail/follow messages from a message bus file

Implements: monitoring-ui Q1, message-bus-tools Q1
```

## Write Output
Write output.md with summary of what was implemented.
