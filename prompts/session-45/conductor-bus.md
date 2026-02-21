# Task: Add `conductor bus` command

## Context

You are implementing a new subcommand group `conductor bus` for the Conductor Loop project.

**Working directory**: /Users/jonnyzzz/Work/conductor-loop
**Primary language**: Go
**Binary**: cmd/conductor/ (built to bin/conductor)

## Goal

Add a `conductor bus` subcommand group that reads the project/task message bus via the conductor server API. This is the server-based equivalent of `run-agent bus read` (which requires local file access).

### Why this is needed

Currently:
- `run-agent bus read --project P [--task T] --root runs` — reads bus locally (requires file access)
- `conductor task logs` — streams agent stdout output via server

But there is NO way to read the TASK-MESSAGE-BUS.md or PROJECT-MESSAGE-BUS.md via the conductor CLI (server-based workflow). The message bus contains rich orchestration events (RUN_START, RUN_STOP, PROGRESS, DECISION, FACT, ERROR) that are essential for understanding what's happening in a task.

### API Endpoints to Use

These endpoints ALREADY EXIST in the server:

```
# Project-level message bus:
GET  /api/projects/{project}/messages           → JSON {messages: [...]}
GET  /api/projects/{project}/messages/stream    → SSE stream

# Task-level message bus:
GET  /api/projects/{project}/tasks/{task}/messages           → JSON {messages: [...]}
GET  /api/projects/{project}/tasks/{task}/messages/stream    → SSE stream
```

The JSON response has shape:
```json
{
  "messages": [
    {
      "msg_id": "MSG-20260221-070000-12345-PID123-0001",
      "ts": "2026-02-21T07:00:00Z",
      "type": "RUN_START",
      "project_id": "conductor-loop",
      "task_id": "task-20260221-070000-myfeature",
      "run_id": "20260221-070000-12345",
      "body": "run started"
    }
  ]
}
```

The SSE stream sends events as:
```
data: {"msg_id":"...","ts":"...","type":"RUN_START",...}

```
(standard SSE format - data lines followed by blank line)

### Command Signature

```
conductor bus read [flags]

Flags:
  --project string    project ID (required)
  --task    string    task ID (optional; reads task-level bus if set, project-level otherwise)
  --server  string    conductor server URL (default: http://localhost:8080)
  --tail    int       show last N messages (default: 0 = all)
  --follow            stream new messages via SSE (keep watching)
  --json              output as raw JSON array (default: formatted text)
```

### Implementation Plan

#### 1. Create `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/bus.go`

New file with:

**`newBusCmd()`** — parent bus command group:
```go
func newBusCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "bus",
        Short: "Read and post to the message bus",
        RunE: func(cmd *cobra.Command, args []string) error {
            return cmd.Help()
        },
    }
    cmd.AddCommand(newBusReadCmd())
    return cmd
}
```

**`newBusReadCmd()`** — the main implementation, following the pattern from `task_logs.go`:
- Flags: --project (required), --task, --server, --tail, --follow, --json
- Function: `conductorBusRead(stdout io.Writer, server, project, taskID string, tail int, follow bool, jsonOutput bool) error`

**Logic (non-follow mode)**:
1. Build URL:
   - If `--task` is set: `GET /api/projects/{project}/tasks/{task}/messages`
   - Otherwise: `GET /api/projects/{project}/messages`
2. Add `?tail={N}` query parameter if tail > 0 (wait — the API doesn't support `?tail`; you'll need to client-side take the last N messages from the returned list)
3. Parse the JSON response
4. If `--json`: print the raw messages JSON array
5. Otherwise: print each message in formatted text:
   ```
   [2026-02-21 07:00:00] RUN_START  run started
   [2026-02-21 07:01:00] PROGRESS   Starting sub-agent for task X
   [2026-02-21 07:02:00] FACT       Build passed
   ```
   Format: `[<timestamp>] <type padded to 12 chars>  <body first line>`

**Logic (follow mode, `--follow`)**:
1. First do a non-follow read to get existing messages (show last N if --tail)
2. Then connect to the SSE stream endpoint:
   - If `--task` is set: `GET /api/projects/{project}/tasks/{task}/messages/stream`
   - Otherwise: `GET /api/projects/{project}/messages/stream`
3. Parse SSE events:
   - `data: <json>` lines: decode the message JSON and print it in the same format as above
   - Blank lines: SSE event separator, ignore
   - `event: done` line: stop streaming
   - `event: heartbeat` line: ignore
4. If connection drops and `--follow`, reconnect with exponential backoff (2s, 4s, 8s, cap 30s)

#### 2. Register in `main.go`

In `newRootCmd()` in `cmd/conductor/main.go`, add:
```go
cmd.AddCommand(newBusCmd())
```

Look at how other commands are added (status, task, project, watch, job) and follow the same pattern.

#### 3. Create `cmd/conductor/bus_test.go`

Write tests covering:
- Project-level bus read (non-follow mode) — use httptest.NewServer
- Task-level bus read (non-follow mode)
- --tail filtering (client-side, from end of message list)
- Empty bus (should print "no messages" or equivalent)
- JSON output mode
- HTTP error from server (404, 500)
- SSE follow mode: stream a few messages then `event: done` → should terminate
- --help shows correct usage

Aim for 8+ tests.

## Implementation Notes

### Formatted message output

```go
func formatBusMessage(msg struct {
    MsgID     string    `json:"msg_id"`
    Timestamp time.Time `json:"ts"`
    Type      string    `json:"type"`
    Body      string    `json:"body"`
}) string {
    ts := msg.Timestamp.UTC().Format("2006-01-02 15:04:05")
    msgType := fmt.Sprintf("%-12s", msg.Type)
    body := msg.Body
    if idx := strings.IndexByte(body, '\n'); idx >= 0 {
        body = body[:idx] + "..."
    }
    return fmt.Sprintf("[%s] %s  %s", ts, msgType, body)
}
```

### SSE parsing for follow mode

The SSE endpoint emits:
```
data: {"msg_id":"...","type":"RUN_START","body":"run started",...}

event: heartbeat
data: {}

event: done
data: {}
```

Use a `bufio.Scanner` on the response body, line by line:
```go
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "data: ") {
        payload := line[len("data: "):]
        // decode and print
    }
    if strings.HasPrefix(line, "event: done") {
        return nil // stream ended
    }
}
```

### Reconnect logic for follow mode

```go
backoff := 2 * time.Second
const maxBackoff = 30 * time.Second
for {
    err := streamBus(...)
    if err == nil {
        return nil // clean done
    }
    // print reconnect notice
    time.Sleep(backoff)
    backoff = min(backoff*2, maxBackoff)
}
```

### JSON structs for API response

```go
type busMessagesResponse struct {
    Messages []busMessage `json:"messages"`
}

type busMessage struct {
    MsgID     string    `json:"msg_id"`
    Timestamp time.Time `json:"ts"`
    Type      string    `json:"type"`
    ProjectID string    `json:"project_id"`
    TaskID    string    `json:"task_id"`
    RunID     string    `json:"run_id"`
    Body      string    `json:"body"`
}
```

## Files to Study

Before implementing, read these files:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task_logs.go` — SSE streaming pattern
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task_logs_test.go` — test patterns
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go` — how to register a new command

## Quality Requirements

1. `go build ./cmd/conductor/...` must pass
2. `go test ./cmd/conductor/...` must pass with all new tests
3. `go test -race ./cmd/conductor/...` must pass (no data races)
4. `./bin/conductor bus --help` must show `read` as a subcommand
5. `./bin/conductor bus read --help` must show correct usage

## Docs Update

Add a brief section to `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md` documenting `conductor bus read` under the conductor commands section.

## Commit

Once done, commit with:
```
feat(cli): add conductor bus read command for viewing message bus via server
```

Create a DONE file when complete:
```bash
touch /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/${JRUN_TASK_ID}/DONE
```
