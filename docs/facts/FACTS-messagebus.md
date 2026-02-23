# FACTS: Message Bus & Agent Protocol
<!-- Generated 2026-02-23 from specs, dev docs, git history, and legacy swarm docs -->

---

## Message Format & File Layout

[2026-02-05 00:11:03] [tags: messagebus, format, wire-format]
Message bus files use YAML front-matter separated by `---` document delimiters. Each message consists of two YAML documents: a header (structured metadata) and a body (free text). The exact wire format is:
```
---
<yaml header fields>
---
<free-text body>
```
Introduced in commit `74e5d6571f82ca9393f52d5be7c4fcd58115c0eb` (feat(stage1): core infrastructure).

[2026-02-05 00:11:03] [tags: messagebus, format, required-fields]
Required header fields are: `msg_id`, `ts`, `type`, `project_id`. All other fields are optional. The field `project_id` (not `project`) is the canonical name in conductor-loop — differs from the early legacy spec (`project`) used in jonnyzzz-ai-coder runs pre-2026-02-05.

[2026-02-05 00:11:03] [tags: messagebus, format, optional-fields]
Optional header fields: `task_id`, `run_id`, `parents` (list), `issue_id`, `attachment_path` (legacy single), `attachments` (preferred multi-list), `links`, `meta`. The `attachment_path` field is kept for backward compatibility; `attachments[]` is preferred for new entries.

[2026-02-05 00:11:03] [tags: messagebus, format, body]
The body is free text. It must not contain a line that is exactly `---` (no content), as that terminates the entry. Body is stored after the second `---` separator and is not part of the YAML header (`yaml:"-"` tag in Go struct).

[2026-02-20 18:10:33] [tags: messagebus, format, extended-fields]
Commit `4968fefb3adea19907a2d69e39e5e7011b5554db` extended the Go Message struct with: structured `parents` (object form with `kind`/`meta`), `meta` (free-form map), `links[]`, `issue_id`, `attachments[]`. This aligns the implementation with the object model spec.

---

## Message ID (`msg_id`) Format

[2026-02-05 00:11:03] [tags: messagebus, msg_id, format]
`msg_id` format: `MSG-{YYYYMMDD-HHMMSS}-{NANOSECONDS_9DIGITS}-PID{PID_5DIGITS}-{SEQ_4DIGITS}`
Example: `MSG-20260205-143052-000123456-PID12345-0042`
Components:
- Prefix: `MSG-` (fixed)
- Date-time: UTC, ISO 8601 compact, lexically sortable
- Nanoseconds: 9-digit zero-padded sub-second precision
- PID: `PID` + 5-digit zero-padded process ID (modulo 100000)
- Sequence: 4-digit zero-padded atomic counter per process, wraps at 10000

Implementation in `internal/messagebus/msgid.go`. First introduced in `74e5d6571f82ca9393f52d5be7c4fcd58115c0eb`.

[2026-02-05 00:11:03] [tags: messagebus, msg_id, uniqueness]
Uniqueness guarantee: `(timestamp_nanosecond, PID, sequence)` is globally unique on a single host. PID wraparound risk is negligible for MVP scope. Lexical sort of msg_id values gives chronological order (same as timestamp order), enabling efficient range queries.

[2026-02-05 00:11:03] [tags: messagebus, msg_id, legacy-formats]
Early jonnyzzz-ai-coder runs (pre-2026-02-05) used a different format: `MSG-YYYYMMDD-HHMMSS-agent-rand` (no nanoseconds, agent name as component). The current conductor-loop format (`MSG-YYYYMMDD-HHMMSS-NANOSECONDS-PIDXXXXX-SEQNO`) supersedes that. The analysis doc `analysis/review-message-bus-content.md` in jonnyzzz-ai-coder catalogued violations of the earlier format (missing jobId, inconsistent random suffix, trailing dash).

---

## Concurrency & Locking

[2026-02-05 00:11:03] [tags: messagebus, concurrency, write-strategy]
Initial legacy spec (jonnyzzz-ai-coder swarm, pre-2026-02-04) specified "temp file + atomic swap" for writes. This was identified as catastrophically broken for concurrent appends — all 6 review agents (3 Claude, 3 Gemini) flagged it as the #1 critical blocker (see `docs/swarm/docs/legacy/problem-1-message-bus-race.md`). The race: two agents read the same file, both write their temp file, the second rename drops the first agent's message permanently.

[2026-02-05 00:11:03] [tags: messagebus, concurrency, o-append-flock]
Decision (problem-1-decision.md, 2026-02-04): unanimous 3-agent consensus to use `O_APPEND + flock`. All three agents (Claude, Codex, Gemini) independently recommended this solution. Final decision: exclusive flock for writes, lockless reads. The flock prevents message interleaving (which O_APPEND alone cannot prevent for multi-write messages).

[2026-02-05 00:11:03] [tags: messagebus, concurrency, write-flow]
Write flow: (1) open file with `O_WRONLY|O_APPEND|O_CREATE`, (2) acquire exclusive `flock` with timeout (default 10s), (3) write header+body to OS page cache, (4) release lock, (5) close file. Lock timeout returns `ErrLockTimeout`. No retries in the library — callers are responsible for retry.

[2026-02-05 00:11:03] [tags: messagebus, concurrency, lockless-reads]
Reads are completely lockless: `os.ReadFile()` with no flock. Rationale: append-only property ensures existing messages never change; parser handles incomplete trailing messages gracefully. The parser state machine stops at the last complete `---` delimiter — partial YAML at EOF is silently ignored.

[2026-02-20 13:11:50] [tags: messagebus, concurrency, fsync]
Commit `8065855e01b02a191250ecf67c1cbf297aa6934e` removed fsync from the `AppendMessage` hot path. Default is now NO fsync. Commit `26146daf836cf7bcc6a7aef6ee635a21d811244c` added `WithFsync` option (default false). Without fsync, writes go to OS page cache: ~37,000+ msg/sec throughput (measured), but messages may be lost on OS crash. Data is visible across processes immediately via shared page cache. With fsync: ~100-500 msg/sec (disk-limited).

[2026-02-05 00:11:03] [tags: messagebus, concurrency, platform]
Platform-specific locking:
- Unix/Linux/macOS: `syscall.Flock(fd, LOCK_EX|LOCK_NB)` — advisory lock, lockless reads work correctly
- Windows: `syscall.LockFileEx(handle, LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY, ...)` — mandatory lock, readers MAY be blocked while writer holds lock. Recommended: use WSL2 on Windows for full Unix semantics.

---

## File Paths & Scope

[2026-02-05 00:11:03] [tags: messagebus, paths, scope]
Two scopes of message bus:
- Project bus: `~/run-agent/<project>/PROJECT-MESSAGE-BUS.md` (cross-task facts, stable knowledge)
- Task bus: `~/run-agent/<project>/<task>/TASK-MESSAGE-BUS.md` (task-scoped updates, run lifecycle events)

Task-scoped messages and run lifecycle events MUST go to the task bus. Cross-task facts SHOULD go to the project bus. The UI aggregates both at read time — no mirroring required.

[2026-02-22 00:39:57] [tags: messagebus, paths, auto-discovery]
Commit `c0d96d1629753b41c276426ee6e4185fbda6d8b8` added bus auto-discovery and legacy compat. The CLI `run-agent bus` commands can locate the bus file automatically via `JRUN_*` env vars. The `MESSAGE_BUS` env var is also supported as a direct path.

---

## Message Types

[2026-02-05 00:11:03] [tags: messagebus, types, canonical]
Canonical message types in the current spec (`subsystem-message-bus-tools.md`):
- Agent-posted: FACT, QUESTION, ANSWER, USER, INFO, WARNING, ERROR, OBSERVATION, ISSUE
- Runner-only: START, STOP, CRASH, RUN_START, RUN_STOP (legacy aliases until unified)
- Agents MUST NOT emit START/STOP — those are runner-only lifecycle events.

[2026-02-05 00:11:03] [tags: messagebus, types, legacy-vs-current]
Legacy swarm spec (jonnyzzz-ai-coder, pre-2026-02-04) had types: FACT, QUESTION, ANSWER, USER, START, STOP, ERROR, INFO, WARNING, OBSERVATION, ISSUE. The conductor-loop spec added: CRASH, RUN_START, RUN_STOP (runner lifecycle); added DECISION as a recommended agent type; and explicitly forbade agents from emitting START/STOP.

[2026-02-20 11:56:06] [tags: messagebus, types, run-lifecycle]
Commit context (work-in-progress state, 2026-02-20): Runner emits `RUN_START`, `RUN_STOP`, `RUN_CRASH` to task bus. `RUN_CRASH` emitted for non-zero exit codes; `RUN_STOP` for successful completion. Verified by `TestRunJobCLIEmitsRunStop` and `TestRunJobCLIEmitsRunCrash` in `internal/runner/job_test.go`. These event types defined as constants in `internal/messagebus/messagebus.go`.

[2026-02-20 11:56:06] [tags: messagebus, types, user-request]
`USER_REQUEST` type was added for threaded task creation flow. Parent source message types for threaded tasks must be `QUESTION` or `FACT`. Child task metadata persisted in `TASK-THREAD-LINK.yaml` with `parent_project_id`, `parent_task_id`, `parent_run_id`, `parent_message_id` fields.

---

## Threading & Relationships (parents[])

[2026-02-05 00:11:03] [tags: messagebus, threading, parents-schema]
`parents[]` header field accepts two forms:
1. String form (shorthand): `parents: [<msg_id>, <msg_id>]` — each entry treated as `{msg_id: ..., kind: reply}`.
2. Object form: `parents: [{msg_id: <id>, kind: <relationship>, meta: <map>}]`

Both forms MUST be accepted by tooling. String entries are backward-compatible with legacy agents.

[2026-02-05 00:11:03] [tags: messagebus, threading, relationship-kinds]
Recommended (non-normative) relationship kinds: `reply`, `answers`, `relates_to`, `depends_on`, `blocks`, `blocked_by`, `supersedes`, `duplicates`, `child_of`. These are advisory only — tooling MUST accept any `kind` string without validation in MVP. Writers SHOULD use snake_case for consistency.

[2026-02-05 00:11:03] [tags: messagebus, threading, cross-scope]
Task messages MAY reference project-level messages via `parents[]`. UI aggregates both buses and resolves cross-scope references with scope labels (e.g., "[project]", "[task]").

[2026-02-05 00:11:03] [tags: messagebus, threading, issue-id]
`issue_id` is an alias for `msg_id` in ISSUE messages. If omitted, `msg_id` serves as the issue identifier. Decision: `issue_id` kept as a convenience header for easier parsing, but semantically equivalent to `msg_id`. Dependency kinds are advisory only — no runtime enforcement planned.

---

## Size Limits & Rotation

[2026-02-05 00:11:03] [tags: messagebus, size, limits]
Soft limit: 64KB per message body. Larger payloads should be stored as attachments and referenced via `attachments[]` or `attachment_path`. Attachment paths are relative to the task folder. No hard enforcement in MVP.

[2026-02-21 00:24:03] [tags: messagebus, size, auto-rotate]
Commit `afa967391255cf72e9e6935efc8b279d45f6778f` added `WithAutoRotate(maxBytes int64)` option. When a write would exceed the threshold, the bus file is renamed to `<path>.YYYYMMDD-HHMMSS.archived` and a fresh file starts. SSE streaming handles rotation via `ErrSinceIDNotFound` reset (client resets `lastID` to empty string and re-streams from beginning of new file).

[2026-02-21 00:11:03] [tags: messagebus, size, read-last-n]
Commit `180a1a2713c8594385dee9969991e0167a68ca84` added `ReadLastN(n int)` method. Uses a 64KB seek window (doubles up to 3× before falling back to full read) — avoids loading the entire file for small tail queries.

[2026-02-05 00:11:03] [tags: messagebus, size, gc-command]
`run-agent gc --rotate-bus --bus-max-size 10MB --root runs` rotates all bus files exceeding the threshold in a single pass. No compaction/cleanup in MVP otherwise — files are append-only and grow indefinitely.

---

## Go API & Implementation

[2026-02-05 00:11:03] [tags: messagebus, implementation, package]
Package: `internal/messagebus/`. Key files:
- `messagebus.go` — Core implementation: `MessageBus`, `AppendMessage`, `ReadMessages`, `PollForNew`
- `msgid.go` — `GenerateMessageID()` function
- `lock.go` — Platform-agnostic `LockExclusive`/`Unlock` interface
- `lock_unix.go` — `syscall.Flock` implementation (build tag: `!windows`)
- `lock_windows.go` — `syscall.LockFileEx` implementation (build tag: `windows`)

[2026-02-05 00:11:03] [tags: messagebus, implementation, go-struct]
Go `Message` struct fields (YAML tags): `msg_id`, `ts`, `type`, `project_id`, `task_id`, `run_id`, `parents` (omitempty), `attachment_path` (omitempty), `Body` (yaml:"-", not in header). Extended in commit `4968fefb` with: `IssueID`, `Meta`, `Attachments[]`, `Links[]`, object-form Parents.

[2026-02-05 00:11:03] [tags: messagebus, implementation, options]
Constructor options (functional options pattern):
- `WithLockTimeout(d time.Duration)` — lock acquisition timeout (default 10s)
- `WithPollInterval(d time.Duration)` — polling interval for `PollForNew` (default 200ms)
- `WithClock(fn func() time.Time)` — injectable clock for testing
- `WithFsync(bool)` — enable/disable fsync on write (default false, added 2026-02-20)
- `WithAutoRotate(maxBytes int64)` — rotate bus file at threshold (added 2026-02-21)

[2026-02-05 00:11:03] [tags: messagebus, implementation, read-api]
`ReadMessages(sinceID string)`: lockless read. Returns all messages if `sinceID=""`. Returns messages after (not including) `sinceID` if found. Returns `ErrSinceIDNotFound` if `sinceID` not in file (client out of sync — should reset to ""). Filtering is a linear scan — no index file.

[2026-02-05 00:11:03] [tags: messagebus, implementation, poll-api]
`PollForNew(lastID string)`: blocks until new messages appear. Uses busy-wait polling with `pollInterval` sleep (default 200ms) — NOT event-driven (no inotify/fsnotify). Appropriate for SSE streaming endpoints.

---

## REST API & SSE

[2026-02-05 00:11:03] [tags: messagebus, rest-api, endpoints]
REST API endpoints:
- `GET /api/v1/messages?project_id=<id>[&task_id=<id>][&after=<msg_id>]` — returns list of messages. Returns 404 if `after` ID not found.
- `GET /api/v1/messages/stream` (SSE) — streaming endpoint. Query params: `project_id`, `task_id` (optional). Supports `Last-Event-ID` header for resumable clients.
- `POST /api/v1/messages` — post a message (implemented 2026-02-20, registered in `internal/api/routes.go:24`, handler `handlePostMessage`). Fields: `type`, `project_id`, `task_id`, `content`.

[2026-02-20 11:56:06] [tags: messagebus, rest-api, sse-id]
Commit context (2026-02-20): SSE `streamMessages` handler in `internal/api/sse.go` now sets `ev.ID = msg.MsgID` for each SSE event, enabling resumable clients via `Last-Event-ID` header. Full message payload returned: `msg_id`, `timestamp`, `type`, `project_id`, `task_id`, `run_id`, `issue_id`, `parents`, `meta`, `content`. SSE heartbeat event sent every 30s.

---

## CLI Commands

[2026-02-05 00:11:03] [tags: messagebus, cli, commands]
`run-agent bus` subcommands:
- `run-agent bus post --type <TYPE> --body "<text>"` — append a message. Reads `JRUN_*` env vars automatically. Prints generated `msg_id` to stdout. Optionally: `--project`, `--task`, `--root` for explicit addressing.
- `run-agent bus read --project <p> --task <t> --root <r> [--tail N]` — read messages.
- `run-agent bus watch` — blocking stream/poll (planned).

Agents rely on `JRUN_*` env vars; CLI error messages MUST NOT instruct agents to set env vars directly.

---

## Agent Protocol

[2026-02-05 00:11:03] [tags: agent-protocol, behavioral-rules, communication]
Agents MUST NOT communicate directly with other agents or users; all coordination happens via the message bus and `output.md`. Agents MUST use `run-agent bus` tooling — direct file appends to bus files are disallowed. Agents SHOULD log progress to stderr during long operations.

[2026-02-05 00:11:03] [tags: agent-protocol, behavioral-rules, scope]
Agents MUST work on a scoped task and exit when done. Agents MUST delegate if a task is too large or outside their folder context. Agents SHOULD scope work to a single module/folder and delegate other folders to sub-agents. Max delegation depth: 16 (configurable; not currently enforced at runtime — logged as future issue per Q3 resolution in agent-protocol-QUESTIONS.md).

[2026-02-05 00:11:03] [tags: agent-protocol, output-md, resolution]
Resolution (2026-02-04, Q1 in agent-protocol-QUESTIONS): `output.md` creation is best-effort from the agent (runner prepends `Write output.md to <full_path>` at start of prompt). Runner INDEPENDENTLY captures stdout to `agent-stdout.txt` and stderr to `agent-stderr.txt`. If `output.md` does not exist after agent exits, runner MUST create it from `agent-stdout.txt`. This "Unified Rule" makes `output.md` always exist for parent agents.

[2026-02-05 00:11:03] [tags: agent-protocol, run-folder, ownership]
Run folder ownership: prompt-guided (no OWNERSHIP.md file). Runner injects `RUN_FOLDER` path in the prompt preamble text (not as an env var in spec, but JRUN_* env vars ARE set in practice). Sub-agents write only inside their own `RUN_FOLDER`. Sub-agents MUST NOT modify sibling run folders or parent run files (except via message bus).

[2026-02-05 00:11:03] [tags: agent-protocol, environment-variables]
JRUN_* env vars injected by runner into agent process:
- `JRUN_PROJECT_ID` — project identifier
- `JRUN_TASK_ID` — task identifier
- `JRUN_ID` — run identifier for this execution
- `JRUN_PARENT_ID` — run ID of parent (if spawned as sub-agent)
Also: `TASK_FOLDER`, `RUN_FOLDER`, `MESSAGE_BUS` (absolute path), `CONDUCTOR_URL`.

[2026-02-05 00:11:03] [tags: agent-protocol, task-state]
`TASK_STATE.md`: free text written by root agent. No strict schema; must stay short (status + next step, no logs). On restarts, new root agent MUST read `TASK_STATE.md` and continue from it. Root agents MUST write updates each cycle and before exit.

[2026-02-05 00:11:03] [tags: agent-protocol, facts-files]
FACT files: Markdown with YAML front matter. Naming: `FACT-<timestamp>-<name>.md` (project-level), `TASK-FACTS-<timestamp>.md` (task-level). Promotion to project-level facts is decided by the root agent; task agents can propose via message bus.

[2026-02-05 00:11:03] [tags: agent-protocol, cwd-guidance]
CWD guidance by agent role:
- Root agent: task folder
- Code-change sub-agents (Implementation/Test/Debug): project source root
- Research/Review sub-agents: task folder (unless parent overrides)
CWD recorded in `run-info.yaml` for audit.

[2026-02-05 00:11:03] [tags: agent-protocol, cancellation]
Cancellation protocol: runner sends SIGTERM to agent process group (pgid), waits 30 seconds, then SIGKILL. Agent should flush message bus and TASK_STATE.md on SIGTERM when possible.

[2026-02-05 00:11:03] [tags: agent-protocol, restart-prefix]
On root-agent restart, runner MUST prefix the prompt with `Continue working on the following` before the original task prompt — but only on restart (not first run), and only after all sub-agents have completed. Resolution of Q2 in agent-protocol-QUESTIONS.md (2026-02-20).

[2026-02-05 00:11:03] [tags: agent-protocol, exit-codes]
Exit codes: 0 = completed, 1 = failed. Other statuses conveyed via stdout/stderr and message bus. Runner emits `RUN_STOP` (exit 0) or `RUN_CRASH` (exit != 0) lifecycle events.

---

## Evolution: Legacy vs Current

[2026-02-04 00:00:00] [tags: messagebus, evolution, concurrency-model]
CHANGED: Legacy spec used "temp file + atomic swap" for writes (identified as broken in 2026-02-04 swarm review). Current implementation uses `O_APPEND + flock` exclusively (implemented 2026-02-05 in `74e5d6571`).

[2026-02-04 00:00:00] [tags: messagebus, evolution, field-names]
CHANGED: Legacy swarm spec used `project` (not `project_id`) and `task` (not `task_id`) as field names. Conductor-loop uses `project_id` and `task_id` consistently to match run-info.yaml and storage layout conventions.

[2026-02-04 00:00:00] [tags: messagebus, evolution, attachment-model]
CHANGED: Legacy spec had only `attachment_path` (single attachment, relative path). Current spec adds `attachments[]` multi-attachment list with `path`, `kind`, `label`, `mime`, `size_bytes`, `sha256` fields. `attachment_path` kept as legacy alias.

[2026-02-04 00:00:00] [tags: messagebus, evolution, parents-model]
EXTENDED: Legacy object model spec had only: `reply`, `blocks`, `supersedes`, `relates_to`, `answers` as suggested kinds. Current spec adds: `depends_on`, `blocked_by`, `duplicates`, `child_of`. Both specs state these are advisory/non-normative.

[2026-02-20 18:10:33] [tags: messagebus, evolution, model-extension]
EXTENDED (2026-02-20): Commit `4968fefb` added `meta` (free-form map), `links[]`, `issue_id`, and `attachments[]` to the Go struct, bringing implementation in line with the full object model spec.

[2026-02-21 00:24:03] [tags: messagebus, evolution, rotation]
ADDED (2026-02-21): `WithAutoRotate` and `ReadLastN` added to address unbounded file growth. Both were listed as "No compaction/cleanup in MVP" in the original spec — addressed in Session #25.

[2026-02-22 00:39:57] [tags: messagebus, evolution, auto-discovery]
ADDED (2026-02-22): Bus auto-discovery and legacy compat added — CLI can locate bus from env vars without explicit `--root` flag in most contexts.

---

## Performance & Known Limitations

[2026-02-05 00:11:03] [tags: messagebus, performance, throughput]
Without fsync: ~37,000+ msg/sec (measured, 10 concurrent writers, macOS). With fsync: ~100-500 msg/sec (disk-limited). Practical limit: <50 concurrent writers before lock contention degrades throughput. Per-task buses reduce contention.

[2026-02-05 00:11:03] [tags: messagebus, limitations, known]
Known limitations (from review + dev doc):
1. File size grows unbounded (mitigated by `WithAutoRotate` and `gc --rotate-bus`)
2. No complex queries — linear scan only (filter by sinceID)
3. NFS/network filesystems: O_APPEND and flock may not work correctly
4. Windows: mandatory locks may block readers during writes
5. No transactional semantics (cannot atomically append to two buses)
6. Polling-based notification (PollForNew uses 200ms busy-wait, not inotify)
7. No garbage collection of deleted/obsolete messages

[2026-02-05 00:11:03] [tags: messagebus, limitations, file-size]
File size tested up to 100 MB (~100,000 messages). Practical limit ~1 GB before reads become slow (full file parse on each read). Per-task bus design limits individual file growth. Compaction not implemented — rotation (renaming) is the strategy.

---

## jonnyzzz-ai-coder Pre-History

[2026-01-29 00:00:00] [tags: messagebus, pre-history, jonnyzzz-ai-coder]
The `message-bus-mcp` directory referenced in the task prompt does not exist in `/Users/jonnyzzz/Work/jonnyzzz-ai-coder/`. Found instead: `analysis/review-message-bus-content.md`, `analysis/message-bus-usefulness.md`, `prompts/conductor-loop/message-bus.md`. The analysis (2026-01-29) examined the jonnyzzz-ai-coder MESSAGE-BUS.md files and found: 0 Q&A pairs, 0 cross-references, 59% protocol compliance (missing required fields). MESSAGE-BUS was used as an audit log, not a coordination channel.

[2026-01-28 00:00:00] [tags: messagebus, pre-history, early-msg-id-format]
Early jonnyzzz-ai-coder message IDs used format: `MSG-YYYYMMDD-HHMMSS-{agent}-{rand}` (e.g., `MSG-20260128-000000-codex-001`). Some lacked random suffix. Some had trailing dash. Types included: FACT, COMPLETE, PROGRESS, TASK, REVIEW — COMPLETE and TASK types not present in current conductor-loop canonical type list.

[2026-01-26 00:00:00] [tags: messagebus, pre-history, decision-date]
The problem-1-decision.md and problem-1-review.md documents originate from the jonnyzzz-ai-coder swarm orchestration in January-February 2026. The O_APPEND+flock consensus was reached by 2026-02-04 and implemented in conductor-loop on 2026-02-05.
