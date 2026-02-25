# Message Bus Protocol

This document reflects the current implementation in `internal/messagebus/`, `cmd/run-agent/bus.go`, and `internal/api/*`.

## Overview

The message bus is an append-only, file-backed event log used for task/project coordination.

- Storage files:
  - task scope: `TASK-MESSAGE-BUS.md`
  - project scope: `PROJECT-MESSAGE-BUS.md`
- Core package: `internal/messagebus`
- CLI: `run-agent bus post|read|discover`

## Message Schema

`internal/messagebus/messagebus.go`:

```go
type Parent struct {
    MsgID string            `yaml:"msg_id"`
    Kind  string            `yaml:"kind,omitempty"`
    Meta  map[string]string `yaml:"meta,omitempty"`
}

type Link struct {
    URL   string `yaml:"url"`
    Label string `yaml:"label,omitempty"`
    Kind  string `yaml:"kind,omitempty"`
}

type Message struct {
    MsgID     string            `yaml:"msg_id"`
    Timestamp time.Time         `yaml:"ts"`
    Type      string            `yaml:"type"`
    ProjectID string            `yaml:"project_id"`
    TaskID    string            `yaml:"task_id"`
    RunID     string            `yaml:"run_id"`
    IssueID   string            `yaml:"issue_id,omitempty"`
    Parents   []Parent          `yaml:"parents,omitempty"`
    Links     []Link            `yaml:"links,omitempty"`
    Meta      map[string]string `yaml:"meta,omitempty"`
    Body      string            `yaml:"-"`
}
```

Important corrections:

- There is no `attachment_path` / `attachments` field in the current `Message` struct.
- `parents` supports both formats on read:
  - legacy string list (`["MSG-..."]`)
  - structured object list (`[{msg_id, kind, meta}]`)

## Message ID Format

`GenerateMessageID()` (`internal/messagebus/msgid.go`) generates:

`MSG-YYYYMMDD-HHMMSS-NANOSECONDS-PID#####-####`

Example:

`MSG-20260223-204919-240276000-PID67952-0001`

## On-Disk Format

Each message is stored as:

1. YAML metadata block between `---` separators.
2. Body text after the second `---`.

Parser compatibility:

- Legacy one-line records are accepted:
  - `[YYYY-MM-DD HH:MM:SS] TYPE: body`
- Legacy records are synthesized as IDs like `LEGACY-LINE-000000123`.

## Validation Rules

`AppendMessage` enforces:

- `type` must be non-empty
- `project_id` must be non-empty

There is no hardcoded allowlist of message types in the core library.

## Write Path (Locking, Retries, fsync, Rotation)

Write flow (`AppendMessage` -> `tryAppend`):

1. Open bus file with `os.O_WRONLY|os.O_APPEND|os.O_CREATE`.
2. Acquire exclusive lock (`LockExclusive`) with timeout.
3. Optionally rotate (if auto-rotate threshold reached).
4. Append serialized message.
5. Optionally `file.Sync()` when fsync is enabled.
6. Unlock and close.

### Lock strategy

- Locking is exclusive for writers.
- Reads are lockless (`ReadMessages` uses `os.ReadFile`).
- Lock timeout default: `10s`.

### Retry strategy

Append retries on lock timeout (`ErrLockTimeout`):

- default `maxRetries = 3`
- default `retryBackoff = 100ms`
- exponential backoff across attempts
- contention counters are exposed by `ContentionStats()`

### fsync behavior

- `WithFsync(false)` is the default.
- `WithFsync(true)` enables `file.Sync()` after append.

### Rotation behavior

`WithAutoRotate(maxBytes)` is implemented.

When the current file size is `>= maxBytes` at append time:

- writer unlocks/closes current file,
- renames it to `<bus>.<UTC YYYYMMDD-HHMMSS>.archived` (best effort),
- opens/locks a fresh bus file,
- continues writing the new message.

## Read APIs

Implemented read methods (`internal/messagebus/messagebus.go`):

- `ReadMessages(sinceID)`
- `ReadMessagesSinceLimited(sinceID, limit)`
- `ReadLastN(n)`
- `PollForNew(lastID)`

`ReadLastN` details:

- initial seek window: `64KB`
- grows window before full-read fallback

`ErrSinceIDNotFound`:

- returned when a requested `sinceID` is missing
- follow/read loops can reset cursor and continue

## Run Lifecycle Event Types

Defined in `internal/messagebus/messagebus.go`:

- `RUN_START`
- `RUN_STOP`
- `RUN_CRASH`

Runner emits these in `internal/runner/job.go` and `internal/runner/wrap.go`.

## Bus CLI

Current subcommands (`cmd/run-agent/bus.go`):

- `run-agent bus post`
- `run-agent bus read`
- `run-agent bus discover`

There is no `bus watch` subcommand.

### `bus post` path resolution order

1. `--bus`
2. `JRUN_MESSAGE_BUS` env var
3. `--project` / `--task` hierarchy resolution
4. upward auto-discovery
5. error

### `bus post` message context inference

For `project_id`, `task_id`, `run_id`:

1. explicit flags (`--project`, `--task`, `--run`)
2. inferred from resolved bus path, `JRUN_RUN_FOLDER`, `JRUN_TASK_FOLDER`
3. `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`
4. error if `project_id` still missing

### `bus read` behavior

- `--tail` default is `20`
- `--follow` polls every `500ms`
- on `ErrSinceIDNotFound`, follow mode resets cursor (`lastID = ""`) and continues

### Auto-discovery order

Per directory, `bus discover` searches in this order:

1. `TASK-MESSAGE-BUS.md`
2. `PROJECT-MESSAGE-BUS.md`
3. `MESSAGE-BUS.md`

## HTTP/SSE Message APIs

### v1 endpoints

- `GET /api/v1/messages`
- `POST /api/v1/messages`
- `GET /api/v1/messages/stream`

`POST /api/v1/messages` defaults `type` to `USER` when omitted.

### Project/task endpoints

- `GET|POST /api/projects/{project}/messages`
- `GET /api/projects/{project}/messages/stream`
- `GET|POST /api/projects/{project}/tasks/{task}/messages`
- `GET /api/projects/{project}/tasks/{task}/messages/stream`

List endpoints support `since` and `limit`.

### SSE payload shape

Message streaming sets:

- SSE `id: <msg_id>`
- `event: message`
- JSON payload includes:
  - `msg_id`
  - `timestamp`
  - `type`
  - `project_id`
  - `task_id`
  - `run_id`
  - `issue_id`
  - `parents` (msg_id list)
  - `meta`
  - `body`

## Operational Notes

- Message-bus files should live on local filesystems.
- Reads are lockless by design; writes are serialized by exclusive lock.
- Auto-rotation and GC rotation (`run-agent gc --rotate-bus`) are complementary.

---

Last updated: 2026-02-23
