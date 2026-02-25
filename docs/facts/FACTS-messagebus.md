# FACTS: Message Bus & Agent Protocol

## Validation Round 2 (codex)

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, message-schema]
`internal/messagebus/messagebus.go` currently defines `Message` with `msg_id`, `ts`, `type`, `project_id`, `task_id`, `run_id`, `issue_id`, `parents`, `links`, `meta`, and body (`yaml:"-"`). There is no `attachment_path` or `attachments` field in the current Go struct.

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, parents-model]
`parents` supports backward-compatible parsing for both string arrays and object arrays. String entries are converted to `Parent{MsgID: ...}`; object entries decode to `Parent{msg_id, kind, meta}` (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, legacy-compat]
The parser accepts legacy single-line records (`[YYYY-MM-DD HH:MM:SS] TYPE: body`) and synthesizes IDs like `LEGACY-LINE-000000123`; this behavior is implemented in `parseLegacyMessageLine` (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, msg-id]
`msg_id` format is generated as `MSG-YYYYMMDD-HHMMSS-NANOSECONDS-PID#####-####` by `GenerateMessageID()` (`internal/messagebus/msgid.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, locking]
Writes open the bus with `os.O_WRONLY|os.O_APPEND|os.O_CREATE` and acquire exclusive lock via `LockExclusive`; read paths are lockless (`ReadMessages` uses `os.ReadFile`) (`internal/messagebus/messagebus.go`, `internal/messagebus/lock.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, retries]
The message bus now retries appends on lock timeout: default `maxRetries=3`, exponential backoff from `100ms`, with contention counters and observability logs (`internal/messagebus/messagebus.go`). Earlier “no retries in library” facts are outdated.

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, fsync]
`fsync` is optional and defaults to disabled. `WithFsync(true)` enables `file.Sync()` after append (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, rotation]
`WithAutoRotate(maxBytes)` is implemented. On threshold hit, current code unlocks/closes, renames to `<bus>.<UTC timestamp>.archived` (best effort), then reopens and relocks before write (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, read-apis]
`ReadLastN(n)` is implemented with a `64KB` initial seek window and window growth before full-read fallback; `ReadMessagesSinceLimited(sinceID, limit)` is also implemented (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, type-validation]
Message type validation in core bus code is permissive: only non-empty `type` and non-empty `project_id` are enforced. There is no hard allowlist of message types in `AppendMessage` (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, run-events]
Runner lifecycle event constants are `RUN_START`, `RUN_STOP`, `RUN_CRASH` (`internal/messagebus/messagebus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, runner-events]
Runner code emits `RUN_START` at launch and emits `RUN_STOP` for success or `RUN_CRASH` for failure/non-zero exit (`internal/runner/job.go`, `internal/runner/wrap.go`); tests explicitly verify this behavior (`internal/runner/job_test.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, bus-cli]
Current CLI subcommands are `run-agent bus post`, `run-agent bus read`, and `run-agent bus discover` (`cmd/run-agent/bus.go`). There is no `bus watch` subcommand in the current implementation.

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, bus-cli-resolution]
`bus post` path resolution order is: `--bus`, `JRUN_MESSAGE_BUS`, `--project/--task` hierarchy, then upward auto-discovery. Message context inference uses bus path, `JRUN_RUN_FOLDER`/`JRUN_TASK_FOLDER`, then `JRUN_*` env vars (`cmd/run-agent/bus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, bus-read]
`bus read` supports `--tail` (default 20) and `--follow`; follow mode polls every 500ms and resets cursor on `ErrSinceIDNotFound` (`cmd/run-agent/bus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, bus-discovery]
Auto-discovery searches upward for files in this order: `TASK-MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`, `MESSAGE-BUS.md` (`cmd/run-agent/bus.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, api-v1]
`/api/v1/messages` supports GET and POST, and `/api/v1/messages/stream` provides SSE (`internal/api/routes.go`, `internal/api/handlers.go`, `internal/api/sse.go`). POST defaults message type to `USER` when omitted.

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, sse]
Message SSE now sets `id: <msg_id>` and emits JSON containing `msg_id`, `timestamp`, `type`, `project_id`, `task_id`, `run_id`, `issue_id`, `parents` (as msg_id list), `meta`, and `body` (`internal/api/sse.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, project-api]
Project-centric endpoints are implemented: `/api/projects/{project}/messages`, `/api/projects/{project}/messages/stream`, `/api/projects/{project}/tasks/{task}/messages`, `/api/projects/{project}/tasks/{task}/messages/stream`, with `since` and `limit` support in list handlers (`internal/api/handlers_projects_messages.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, env-contract]
Runner currently injects `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, optional `JRUN_PARENT_ID`, `JRUN_TASK_FOLDER`, `JRUN_RUN_FOLDER`, and `JRUN_MESSAGE_BUS` into prompts/environment (`internal/runner/orchestrator.go`, `internal/runner/job.go`, `internal/runner/wrap.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, output-md]
`output.md` is guaranteed by fallback logic: `agent.CreateOutputMD(runDir, "")` copies `agent-stdout.txt` to `output.md` when missing (`internal/agent/executor.go`, called from `internal/runner/job.go` and `internal/runner/wrap.go`).

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, spec-history]
`docs/specifications/subsystem-message-bus-tools.md` currently has three revisions in git history: `4e4875d` (2026-02-04), `2fece288` (2026-02-20), `51df36d2` (2026-02-22). The first revision used legacy `project`/`task` field names and planned CLI/API sections.

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, spec-drift]
Specs/dev docs still contain drift relative to code (for example, attachment fields, links schema, and older SSE notes), while implementation in `internal/messagebus`, `cmd/run-agent/bus.go`, and `internal/api/*` is the current source of truth.

[2026-02-23 19:17:10] [tags: messagebus, agent-protocol, external-prehistory]
`/Users/jonnyzzz/Work/jonnyzzz-ai-coder/message-bus-mcp/` is not present in the current filesystem (directory lookup returned `missing`), so no `SPEC.md` from that path could be validated.
