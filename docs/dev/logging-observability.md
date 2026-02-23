# Logging and Observability Strategy

This document defines the production logging strategy for conductor-loop and where operators can inspect runtime events.

## Goals

- Capture high-signal lifecycle and control-plane events.
- Keep logs searchable with consistent structured fields.
- Preserve sensitive data safety with redaction.
- Avoid noisy per-line debug output from hot paths.

## Structured Log Format

Runtime logs use structured `key=value` records (logfmt style) with a common envelope:

- `ts` (RFC3339Nano UTC)
- `level` (`DEBUG|INFO|WARN|ERROR`)
- `subsystem` (for example `api`, `runner`, `messagebus`, `storage`, `startup`)
- `event` (stable event identifier)

Additional fields are added by event, with a strong preference for:

- `project_id`
- `task_id`
- `run_id`
- `request_id`
- `correlation_id`

## Redaction Policy

All structured logs pass through centralized sanitization in `internal/obslog`:

- Key-based redaction for sensitive keys (`token`, `api_key`, `authorization`, `secret`, `password`, etc.).
- Pattern-based redaction for bearer tokens, common API token formats, and JWT-like payloads.
- Truncation of oversized field values.

Rules:

- Never log message bus body payloads.
- Never log raw prompts, raw API keys, or secret-bearing headers.
- Prefer IDs, statuses, counts, and durations over content.

## Coverage by Subsystem

### Startup / Shutdown (`cmd/conductor`, `cmd/run-agent serve`)

- Config discovery/load success and failure.
- Auth safety fallback when API key is missing.
- Server start, stop, signal handling, shutdown failures.

### API Request Handling (`internal/api`)

- Per-request completion log with request/correlation ID and resolved project/task/run IDs when available.
- Structured request error logs.
- UI/API control actions (create task, stop run, resume/delete task, delete project, project GC, message posts).
- Form submission audit trail in `_audit/form-submissions.jsonl` with payload sanitization.

### Runner Orchestration (`internal/runner`)

- Task run start/end, dependency wait failures.
- Ralph loop attempt failures/completions and child-wait failures.
- Run directory allocation, semaphore slot acquire/release, timeout/failure/completion summaries.
- Run start/stop/crash bus-event posting outcomes.

### Message Bus (`internal/messagebus`)

- Lock-timeout retries and retry recovery.
- Exhausted retries and append failures.
- Auto-rotation events and rotation reopen/lock failures.

### Storage / Locking (`internal/storage`)

- Run-info read/write/marshal/unmarshal failures.
- Run-info lock acquisition failures.
- Slow lock waits for run-info updates.

## Operator Inspection Points

### Process Logs

Inspect stdout/stderr for server/runner processes:

- `run-agent serve ...`
- `conductor ...`
- `run-agent task ...`
- `run-agent job ...`

Recommended grep:

```bash
rg "subsystem=(api|runner|messagebus|storage|startup)" /path/to/process.log
rg "event=(run_execution_failed|run_timeout|project_deleted|append_exhausted_retries)" /path/to/process.log
```

### Task and Project Message Buses

- Task bus: `<root>/<project_id>/<task_id>/TASK-MESSAGE-BUS.md`
- Project bus: `<root>/<project_id>/PROJECT-MESSAGE-BUS.md`

Use for domain-level run/task facts and orchestration events.

### API Audit Trail

- `_audit/form-submissions.jsonl` under the configured runs root.

Use for user-triggered API action reconstruction with sanitized payloads.

### Run Artifacts

Per-run directory:

- `run-info.yaml`
- `agent-stdout.txt`
- `agent-stderr.txt`
- `output.md`

Use to correlate structured control-plane events with agent output.

## Event Naming Guidance

- Use verbs in past tense for completed events (`task_run_completed`, `project_deleted`).
- Use `_failed` suffix for errors (`run_event_post_failed`).
- Keep event names stable to preserve query dashboards and alerts.
