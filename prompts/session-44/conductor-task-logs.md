# Task: Add `conductor task logs` command

## Context

You are implementing a new subcommand `conductor task logs` for the Conductor Loop project.

**Working directory**: /Users/jonnyzzz/Work/conductor-loop
**Primary language**: Go
**Binary**: cmd/conductor/ (built to bin/conductor)

## Goal

Add a `conductor task logs` subcommand that streams/prints task output via the conductor server's existing SSE API.

### Why this is needed

Currently, operators can:
- `conductor task status` — check task status
- `conductor watch` — wait for task completion
- `run-agent output --follow` — follow task output (directly, no server needed)

But there is NO way to follow task output via the conductor CLI (server-based workflow).

### Command Signature

```
conductor task logs <task-id> [flags]

Flags:
  --project string     project ID (required)
  --server  string     conductor server URL (default: http://localhost:8080)
  --run     string     specific run ID to stream (default: latest active or most recent)
  --follow             keep streaming until run completes (default: false, print what's available then exit)
  --tail    int        output last N lines of existing output before streaming (default: 0 = all)
```

### Implementation Plan

1. **Create `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task_logs.go`**

The new file should:
- Define `newTaskLogsCmd()` function following the pattern in `task.go`
- Register it as a subcommand in `newTaskCmd()` in `task.go`
- Implement `taskLogs(stdout io.Writer, server, project, taskID, runID string, follow bool, tail int) error`

**Logic**:
1. If `--run` is not provided, first GET `/api/projects/{project}/tasks/{taskID}` to find the latest run ID (the last run in the runs list that is "running" or the most recent if all done)
2. Then stream `/api/projects/{project}/tasks/{taskID}/runs/{runID}/stream/agent-stdout.txt` (SSE endpoint)
3. Parse SSE events:
   - `data:` lines are output lines — print them to stdout
   - `event: done` means streaming complete — exit
   - Empty data lines / heartbeats — ignore
4. If `--follow` is false, read until server closes the stream or gets `event: done`
5. If `--follow` is true, keep reconnecting if connection drops (retry with 2s backoff, max 30s)

**SSE parsing**:
The existing server sends SSE in format:
```
event: line
data: <content>

event: done
data: {}

```
or simple data-only:
```
data: <line content>

```

Parse using `bufio.Scanner` reading line by line from the response body.

2. **Update `task.go`**

Add `cmd.AddCommand(newTaskLogsCmd())` in `newTaskCmd()`.

3. **Add tests in `cmd/conductor/task_logs_test.go`**

Write table-driven tests using `httptest.Server` to mock:
- Normal output streaming (print lines until done)
- Empty output (no lines, just done event)
- Auto-detection of latest run when no --run specified
- 404 response handling

### Existing Patterns to Follow

- **SSE parsing**: Look at `internal/api/sse.go` for how the server sends events
- **HTTP client patterns**: Follow `task.go` and `watch.go` for HTTP call patterns
- **Error handling**: Return `fmt.Errorf("task logs: %w", err)` wrapping style
- **Output**: Write to `cmd.OutOrStdout()`, not `os.Stdout` directly (for testing)

### API Endpoints to Use

1. `GET /api/projects/{project}/tasks/{taskID}` — returns task detail with runs list
   - Response: `{"project_id": "...", "task_id": "...", "status": "...", "runs": [{"run_id": "...", "status": "...", ...}]}`
   - Find the most recent run: last element of `runs` array

2. `GET /api/projects/{project}/tasks/{taskID}/runs/{runID}/stream/agent-stdout.txt` — SSE stream of stdout
   - Content-Type: text/event-stream
   - Sends lines of agent output as SSE events
   - Sends `event: done` when file read completes

### Key Files to Read

Before implementing, read these files:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go` — existing task commands
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/watch.go` — SSE-adjacent polling pattern
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` (lines 614-700) — how SSE is sent
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/sse.go` — SSE event format

### Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./cmd/conductor/...` must pass with all new tests green
3. `go test -race ./cmd/conductor/...` must pass (no data races)
4. `./bin/conductor task logs --help` must show correct usage
5. Run acceptance test: `ACCEPTANCE=1 go test ./test/acceptance/...` must still pass

### Commit Format

```
feat(cli): add conductor task logs command for streaming task output
```

Follow the commit format from AGENTS.md: `<type>(<scope>): <subject>`

### Important Notes

- Do NOT modify MESSAGE-BUS.md directly (orchestrator handles that)
- Follow existing code style in the repository (no unnecessary abstractions)
- Keep the implementation simple and focused — no extra features
- The `--follow` flag should keep trying to reconnect if the connection drops
- Do NOT add `--json` output to this command (task output is human-readable text)
- Use `cobra.MarkFlagRequired` for `--project` flag
