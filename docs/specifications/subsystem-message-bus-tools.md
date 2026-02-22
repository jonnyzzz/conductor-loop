# Message Bus Tooling Subsystem

## Overview
Provides message bus tooling (CLI + REST) for writing to and reading from project/task message buses. The message bus is file-based and implemented in the run-agent binary. Agents must communicate only through the message bus tooling; direct file writes are disallowed.

## Goals
- Provide a consistent append-only message format.
- Support polling and streaming for root agents and the monitoring UI.
- Support message relationships, issue/dependency tracking, run lifecycle events, and attachments.
- Keep project and task message buses strictly separated.

## Non-Goals
- Enforcing full issue-tracking workflows.
- Providing strict global FIFO ordering across concurrent writers.
- Supporting multi-host authentication in MVP.

## Responsibilities
- Define message format and required header fields.
- Define message types and lifecycle event payloads.
- Define CLI/REST behavior for posting, reading, polling, and streaming.
- Define threading, dependencies, and ordering rules.

## Storage & Routing
- Project bus path: `~/run-agent/<project>/PROJECT-MESSAGE-BUS.md`.
- Task bus path: `~/run-agent/<project>/<task>/TASK-MESSAGE-BUS.md`.
- Task-scoped messages and run lifecycle events MUST go to the task bus.
- Cross-task facts and stable knowledge SHOULD go to the project bus.
- UI aggregates both buses at read time; no mirroring is required.

## Message Format
Append-only YAML front matter with a free-text body.

Example:
```
---
msg_id: MSG-20260205-123456-123456789-PID01234-0001
ts: 2026-02-05T12:34:56Z
type: ISSUE
project_id: conductor-loop
task_id: task-20260205-docs
run_id: run_20260205-123456-12345
issue_id: ISSUE-1
parents:
  - msg_id: MSG-20260205-120000-000000001-PID01234-0001
    kind: depends_on
attachments:
  - path: attachments/logs/run-stdout.txt
    kind: log
    label: agent stdout
links:
  - kind: output
    path: /path/to/run-agent/conductor-loop/task-20260205-docs/runs/run_20260205-123456-12345/output.md
meta:
  agent_type: codex
---
Need update unit tests before release.
```

Notes:
- Each entry starts with `---` and the header ends with `---`.
- The body is free text. Avoid a line that contains only `---` because it terminates the entry.

### Required Header Fields
- `msg_id`
- `ts`
- `type`
- `project_id`

### Optional Header Fields
- `task_id`
- `run_id`
- `parents`
- `issue_id`
- `attachment_path` (legacy single attachment)
- `attachments` (preferred multi-attachment list)
- `links`
- `meta`

### attachments[] Schema
| Field | Description |
| --- | --- |
| `attachments[].path` | Path relative to the task folder. |
| `attachments[].kind` | Free-form label (log, artifact, screenshot, etc.). |
| `attachments[].label` | Human-friendly label. |
| `attachments[].mime` | Optional MIME type. |
| `attachments[].size_bytes` | Optional size in bytes. |
| `attachments[].sha256` | Optional checksum. |

### links[] Schema
| Field | Description |
| --- | --- |
| `links[].kind` | Free-form label (output, run-info, fact, decision, etc.). |
| `links[].path` | Absolute path to the referenced artifact. |
| `links[].run_id` | Optional run ID the link belongs to. |
| `links[].task_id` | Optional task ID the link belongs to. |

### meta
`meta` is a free-form map for structured payloads (run lifecycle metadata, issue status, etc.).

## Message Types
Canonical types:
- FACT, QUESTION, ANSWER, USER
- INFO, WARNING, ERROR, OBSERVATION
- ISSUE
- START, STOP, CRASH
- RUN_START, RUN_STOP (legacy aliases until unified)

## Issue & Dependency Semantics
- `ISSUE` messages represent issue records. If `issue_id` is omitted, `msg_id` is the issue identifier.
- Use `parents[]` with relationship kinds to express dependencies and links.
- Recommended dependency kinds: `depends_on`, `blocks`, `blocked_by`, `supersedes`, `duplicates`, `relates_to`, `child_of`, `answers`.
- See subsystem-message-bus-object-model.md for the relationship schema.

## Run Lifecycle Events
- Runner emits lifecycle events to the task bus.
- `run_id` is required.
- `meta` SHOULD include: `agent_type`, `pid`, `pgid`, `run_dir`, `cwd`, `prompt_path`, `output_path`, `stdout_path`, `stderr_path`, `parent_run_id`, `previous_run_id`, `command_line`, `exit_code`, `status`, `duration_ms`.
- Body remains a short human-readable summary.

## Tooling Interfaces

### Go Package (Internal)
- `internal/messagebus` provides `AppendMessage`, `ReadMessages`, `PollForNew`.

### REST API (Current)
- `GET /api/v1/messages` with query `project_id` (required), `task_id` (optional), `after` (optional msg_id).
- Response is a list of messages with full headers and body.
- If `after` is unknown, the API returns 404.
- `GET /api/v1/messages/stream` (SSE) with `project_id` and optional `task_id`.
- Clients can send `Last-Event-ID` header with the last seen msg_id.
- SSE events: `message` with payload `{msg_id, content, timestamp}`, `heartbeat` every 30s.
- The server currently does not set SSE `id` fields; clients should track msg_id from the payload.

### REST API (Planned)
- `POST /api/v1/messages` (or `/api/v1/bus`) with fields: `project_id`, `task_id`, `type`, `message`, `parents`, `issue_id`, `attachments`, `links`, `meta`.

### CLI (Planned)
- `run-agent bus post` (append a message entry).
- `run-agent bus read` (read since msg_id).
- `run-agent bus watch` (blocking stream / poll).
- Agents rely on JRUN_* env vars; error messages must not instruct agents to set env vars.

## Ordering & Filtering
- Ordering is based on header `ts`; ties are resolved by `msg_id`.
- No strict FIFO across concurrent writers.
- Clients track last seen msg_id; no cursor files are stored by tooling.

## Concurrency / Atomicity
- Writes use `O_APPEND` with an exclusive file lock and `fsync` after append.
- Lock timeout defaults to 10 seconds.
- Readers tolerate truncated or malformed trailing entries by skipping invalid YAML headers.

## Size Limits & Attachments
- Soft limit: 64KB per message body.
- Larger payloads should be stored as attachments and linked with `attachments` or `attachment_path`.
- Attachment paths are relative to the task folder.

## Compaction / Archival
- No compaction/cleanup in MVP; files can grow (append-only).
