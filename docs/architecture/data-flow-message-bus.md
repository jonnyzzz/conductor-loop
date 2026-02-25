# Message Bus Data Flow

This page describes how message bus data moves through write, read, and follow paths in Conductor Loop.
It is grounded in:

- `internal/messagebus/messagebus.go`
- `internal/messagebus/msgid.go`
- `cmd/run-agent/bus.go`
- `internal/api/sse.go`
- `internal/api/handlers_projects_messages.go`

## Scope hierarchy: run -> task -> project

Message scope is hierarchical in message fields:

- `project_id` is required for writes (`AppendMessage` validation).
- `task_id` is optional and narrows to a task scope.
- `run_id` is optional and narrows to a specific run inside a task.

Physical bus files are project/task scoped:

```text
<root>/
  <project_id>/
    PROJECT-MESSAGE-BUS.md
    <task_id>/
      TASK-MESSAGE-BUS.md
      runs/
        <run_id>/
          run-info.yaml
          ...
```

There is no dedicated run-level bus file. Run scope is carried in each message via `run_id`.

## Write path (`AppendMessage`)

`MessageBus.AppendMessage` (`internal/messagebus/messagebus.go`) flow:

1. Validate bus/message basics (`mb != nil`, `msg != nil`, non-empty `type`, non-empty `project_id`).
2. Generate `msg_id` (`GenerateMessageID`), normalize timestamp to UTC, set `issue_id=msg_id` for `ISSUE`.
3. Serialize message to front-matter format:
   - `---`
   - YAML header
   - `---`
   - body
4. Validate target path (`validateBusPath`): empty path rejected, symlink rejected, non-regular file rejected.
5. Retry loop (`maxRetries`, default 3):
   - attempt 1: immediate
   - attempt N>1: exponential sleep (`retryBackoff * 2^(attempt-2)`, default backoff 100ms)
   - call `tryAppend`
   - retry only on `ErrLockTimeout`; fail fast on any other error
6. `tryAppend`:
   - `os.OpenFile(path, O_WRONLY|O_APPEND|O_CREATE, 0644)`
   - `LockExclusive(file, lockTimeout)` (default 10s)
   - optional auto-rotate if file size `>= autoRotateBytes`:
     - unlock + close current fd
     - rename to `<path>.<YYYYMMDD-HHMMSS>.archived`
     - reopen new bus file with `O_APPEND`
     - re-lock
   - append separator newline when file is non-empty, then write payload
   - optional `fsync` (`WithFsync(true)`)
   - unlock + close

### Write sequence diagram

```text
Caller
  |
  | AppendMessage(msg)
  v
MessageBus.AppendMessage
  | validate + assign msg_id/timestamp
  | serializeMessage
  | validateBusPath
  | for attempt=1..maxRetries
  |   if attempt>1: sleep(backoff*2^(attempt-2))
  |   tryAppend(data)
  |     open(O_WRONLY|O_APPEND|O_CREATE)
  |     LockExclusive(timeout)
  |     [if autoRotate && size>=threshold]
  |       unlock+close -> rename to *.archived -> reopen+lock
  |     appendEntry(newline_if_needed + data)
  |     [if fsync] file.Sync()
  |     unlock+close
  |   on ErrLockTimeout -> retry
  |   on other error -> return error
  v
return msg_id
```

## `msg_id` generation format

`GenerateMessageID` (`internal/messagebus/msgid.go`) returns:

```text
MSG-<YYYYMMDD-HHMMSS>-<nanoseconds_9d>-PID<pid_mod_100000_5d>-<seq_mod_10000_4d>
```

Example shape:

```text
MSG-20260224-081530-123456789-PID01234-0042
```

Components:

- UTC wall clock (`YYYYMMDD-HHMMSS`)
- current second nanoseconds (9 digits)
- process id modulo 100000 (5 digits)
- atomic per-process sequence modulo 10000 (4 digits)

### Concurrency Model Diagram 

```text 
    WRITER (Agent/Runner)              READER (CLI/API/UI) 
   +---------------------+            +---------------------+ 
   |  Generate Msg ID    |            |                     | 
   +----------+----------+            |                     | 
              |                       |                     | 
   +----------v----------+            |                     | 
   | Open O_APPEND       |            |                     | 
   +----------+----------+            |                     | 
              |                       |                     | 
   +----------v----------+            +----------v----------+ 
   | flock (EXCLUSIVE)   |            |  open (READ_ONLY)   | 
   | [Blocks other wrtr] <------------+  [Lockless on Unix] | 
   +----------+----------+            +----------+----------+ 
              |                       |                     | 
   +----------v----------+            +----------v----------+ 
   |  Write YAML Entry   |            | Read From Offset    | 
   +----------+----------+            +----------+----------+ 
              |                       |                     | 
   +----------v----------+            +----------v----------+ 
   |  Optional fsync()   |            |  Parse YAML Docs    | 
   +----------+----------+            +---------------------+ 
              | 
   +----------v----------+ 
   |  unlock & close     | 
   +---------------------+ 
``` 

## Read paths (lockless reads)

All three read APIs do not acquire bus locks explicitly. They open/read files directly.
On Unix/macOS this works well with advisory `flock` writes. On Windows, lock behavior is mandatory and readers can block while writer holds lock (`internal/messagebus/lock_windows.go`).

### `ReadMessages(sinceID)`

- Full-file read via `os.ReadFile`.
- Parse all messages.
- If `sinceID == ""`: return all.
- Else return only messages after matching `sinceID`.
- If not found: return `ErrSinceIDNotFound`.
- Missing file: returns empty list.

### `ReadLastN(n)`

- `n <= 0`: same as full read.
- For small files (`<=64KB`): full read + trim.
- For large files: seek from end, parse tail chunk, grow chunk exponentially (up to 4 attempts), fallback to full read.
- Missing file: returns empty list.

### `ReadMessagesSinceLimited(sinceID, limit)`

- `limit <= 0`: delegates to `ReadMessages`.
- Streaming parse from file reader (no full-file load).
- Scan until `sinceID`, then keep only latest `limit` messages in a ring buffer.
- If `sinceID` not found: `ErrSinceIDNotFound`.
- Missing file: returns empty list.

## API and SSE follow paths

### REST read/list API

Project/task message list endpoints (`internal/api/handlers_projects_messages.go`):

- `GET /api/projects/{project_id}/messages`
- `GET /api/projects/{project_id}/tasks/{task_id}/messages`

Selection logic:

- `since == "" && limit > 0` -> `ReadLastN(limit)`
- `since != "" && limit > 0`:
  - fast path: `ReadLastN(tailWindow)` and slice if `since` present
  - fallback: `ReadMessagesSinceLimited(since, limit)`
- otherwise -> `ReadMessages(since)`

`ErrSinceIDNotFound` maps to HTTP 404.

### SSE follow API

SSE bus stream endpoints:

- `GET /api/v1/messages/stream?project_id=...&task_id=...` (`internal/api/sse.go`)
- `GET /api/projects/{project_id}/messages/stream`
- `GET /api/projects/{project_id}/tasks/{task_id}/messages/stream`

Follow loop in `streamMessageBusPath`:

1. Read `Last-Event-ID` into `lastID`.
2. Poll timer tick -> `ReadMessages(lastID)`.
3. On success: emit each message as SSE `event: message` with `id: <msg_id>`, set `lastID` to latest sent.
4. On `ErrSinceIDNotFound`: set `lastID = ""` and continue.

`lastID` reset is the key recovery behavior after rotation/truncation: if the old `since` id is no longer in the active file, stream resumes from the beginning of the current bus file.

## CLI read/follow path

`run-agent bus read` (`cmd/run-agent/bus.go`) does:

1. Initial fetch:
   - `--tail > 0` -> `ReadLastN(tail)`
   - else -> `ReadMessages("")`
2. Follow mode (`--follow`):
   - set `lastID` to last printed message (if any)
   - every 500ms call `ReadMessages(lastID)`
   - on `ErrSinceIDNotFound`, reset `lastID=""` and continue

As with SSE, rotation can drop the old id from the active file; reset-to-empty reattaches the follower to the current file.

### Follow recovery sequence (rotation case)

```text
Follower lastID = MSG-old
Writer rotates bus -> active file no longer contains MSG-old
Follower poll: ReadMessages("MSG-old") -> ErrSinceIDNotFound
Follower sets lastID=""
Next poll: ReadMessages("") -> reads from start of current active file
```

## CLI bus discovery and auto-resolution order

`run-agent bus post` path resolution (`resolveBusPostPath`):

1. `--bus`
2. `JRUN_MESSAGE_BUS` env
3. `--project` (+ optional `--task`) -> `resolveBusFilePath(root, project, task)`
4. auto-discover nearest bus file upward from CWD
5. error

`resolveBusFilePath` root fallback:

1. explicit `--root`
2. `JRUN_RUNS_DIR` env
3. `./runs`

`run-agent bus read` path resolution:

1. `--project` (+ optional `--task`) -> auto-resolve
2. `--bus`
3. `JRUN_MESSAGE_BUS` env
4. auto-discover nearest bus file upward from CWD
5. error

If both `--project` and `--bus` are provided to `bus read`, command fails.

Auto-discovery order within each searched directory (`discoverBusFilePath`):

1. `TASK-MESSAGE-BUS.md`
2. `PROJECT-MESSAGE-BUS.md`
3. `MESSAGE-BUS.md` (legacy)

Then move one directory up and repeat until filesystem root.

Message context inference for `bus post` (`resolveBusPostMessageContext`) is separate from path resolution:

1. explicit flags `--project/--task/--run`
2. inferred from resolved bus path and `JRUN_RUN_FOLDER`/`JRUN_TASK_FOLDER`
3. `JRUN_PROJECT_ID` / `JRUN_TASK_ID` / `JRUN_ID`

This is how run -> task -> project identity is carried even when only a bus path is provided.
