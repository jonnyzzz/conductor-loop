# Message Bus Tooling Subsystem

## Overview
This subsystem defines how messages are written, read, and streamed through file-backed message buses.
Authoritative implementation lives in:
- `internal/messagebus/*`
- `cmd/run-agent/bus.go`
- `internal/api/handlers.go`
- `internal/api/handlers_projects_messages.go`
- `internal/api/sse.go`

## Bus Files and Scope
Canonical locations:
- Project bus: `<root>/<project>/PROJECT-MESSAGE-BUS.md`
- Task bus: `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`

Auto-discovery search order per directory:
1. `TASK-MESSAGE-BUS.md`
2. `PROJECT-MESSAGE-BUS.md`
3. `MESSAGE-BUS.md`

## Message Schema Reference
The message schema is defined in:
- `docs/specifications/subsystem-message-bus-object-model.md`

Important current constraints:
- No canonical `attachments` or `attachment_path` fields in core `Message` struct.
- `links` entries use `url/label/kind`.

## Message Types
Core message bus does not enforce a fixed type allowlist.
Only non-empty `type` is required.

Runner lifecycle events use these constants:
- `RUN_START`
- `RUN_STOP`
- `RUN_CRASH`

## Message Bus Library Semantics

### Write Path
`AppendMessage` behavior:
- Opens file with `O_WRONLY|O_APPEND|O_CREATE`.
- Acquires exclusive lock (`LockExclusive`).
- Generates `msg_id`, normalizes UTC timestamp, applies ISSUE alias logic.
- Appends entry atomically with separator newline between entries.

Defaults:
- Lock timeout: `10s`
- Poll interval: `200ms`
- Max retries on lock timeout: `3`
- Retry backoff base: `100ms` (exponential)
- `fsync`: disabled by default

Options:
- `WithLockTimeout`
- `WithPollInterval`
- `WithMaxRetries`
- `WithRetryBackoff`
- `WithFsync(true|false)`
- `WithAutoRotate(maxBytes)`

Rotation:
- When enabled and threshold reached, bus is renamed to `<path>.<UTC timestamp>.archived` (best effort), then a fresh file is opened.

### Read Path
Available operations:
- `ReadMessages(sinceID)`
- `ReadMessagesSinceLimited(sinceID, limit)`
- `ReadLastN(n)`
- `PollForNew(lastID)`

Behavior:
- Reads are lockless (`os.ReadFile`/stream parsing).
- Unknown `sinceID` returns `ErrSinceIDNotFound`.
- Legacy single-line format is parsed for backward compatibility.

## CLI Contract (`run-agent bus`)
Subcommands:
- `run-agent bus post`
- `run-agent bus read`
- `run-agent bus discover`

There is no `run-agent bus watch` command; streaming is `read --follow`.

### `bus post`
Path resolution order:
1. `--bus`
2. `MESSAGE_BUS` env var
3. `--project` / `--task` path resolution
4. Upward auto-discovery

Context resolution for message fields (`project_id/task_id/run_id`):
1. Explicit flags
2. Inference from resolved bus path, `RUN_FOLDER`, `TASK_FOLDER`
3. `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`

Defaults:
- `--type` default: `INFO`
- Body from `--body` or piped stdin

### `bus read`
Path resolution order:
1. `--project` / `--task` path resolution
2. `--bus`
3. `MESSAGE_BUS` env var
4. Upward auto-discovery

Rules:
- `--bus` and `--project` together are invalid.
- Default tail is `--tail 20`.
- `--tail <= 0` reads full bus.
- `--follow` polls every `500ms`.
- On `ErrSinceIDNotFound` during follow, cursor resets to full replay.

### `bus discover`
- Searches upward from `--from` (or CWD).
- Returns first bus file found using discovery order above.

## REST API Contract

### `/api/v1/messages`
- `GET /api/v1/messages?project_id=<id>&task_id=<id?>&after=<msg_id?>`
- `POST /api/v1/messages`

POST request:
```json
{
  "project_id": "conductor-loop",
  "task_id": "task-...",
  "run_id": "20260223-...",
  "type": "USER",
  "body": "message text"
}
```

POST response (`201`):
```json
{
  "msg_id": "MSG-...",
  "timestamp": "2026-02-23T19:17:10Z"
}
```

Defaults:
- If `type` is omitted, API defaults to `USER`.

### `/api/v1/messages/stream`
- `GET /api/v1/messages/stream?project_id=<id>&task_id=<id?>`
- Uses `Last-Event-ID` as cursor (`msg_id`).

SSE events:
- `event: message`
- `event: heartbeat`

`message` payload:
```json
{
  "msg_id": "MSG-...",
  "timestamp": "2026-02-23T19:17:10.184887Z",
  "type": "FACT",
  "project_id": "conductor-loop",
  "task_id": "task-...",
  "run_id": "20260223-...",
  "issue_id": "MSG-...",
  "parents": ["MSG-..."],
  "meta": {"source": "runner"},
  "body": "..."
}
```

The SSE `id` field is set to `msg_id`.

### Project-centric message endpoints
- `GET|POST /api/projects/{project_id}/messages`
- `GET /api/projects/{project_id}/messages/stream`
- `GET|POST /api/projects/{project_id}/tasks/{task_id}/messages`
- `GET /api/projects/{project_id}/tasks/{task_id}/messages/stream`

Project/task list query params:
- `since` (msg_id cursor)
- `limit` (capped at `5000`)

POST payload for project/task endpoints:
```json
{
  "type": "USER",
  "body": "message text"
}
```

If `type` is omitted, default is `USER`.

## Ordering and Reliability
- Write order is append order under exclusive lock.
- No strict global ordering guarantees across different bus files.
- Clients should track `msg_id` cursors.
- SSE heartbeats are emitted periodically (default `30s` via SSE config).

## Outdated Assumptions Removed
- `bus watch` subcommand.
- Canonical `attachments` fields in bus schema.
- SSE payload limited to `{msg_id, content, timestamp}`.
- Missing SSE `id` field.
