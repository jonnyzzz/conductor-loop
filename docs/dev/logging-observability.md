# Logging and Observability Strategy

This document describes the current logging and observability surfaces used by conductor-loop.

## Logging Stack

The codebase uses standard library `log.Logger` plus `internal/obslog`.

- No `slog`, `zerolog`, `zap`, or `logrus` integration in production paths.
- Structured logs are emitted through `obslog.Log(...)`.

## Structured Log Format

`internal/obslog/obslog.go` writes logfmt-like records with common keys:

- `ts` (UTC RFC3339Nano)
- `level` (`DEBUG|INFO|WARN|ERROR`)
- `subsystem`
- `event`

Additional event fields are emitted as normalized key/value pairs.

## Redaction and Sanitization

`obslog` applies centralized sanitization before output:

- key-based redaction for sensitive keys (`token`, `api_key`, `authorization`, `secret`, `password`, etc.)
- bearer-token masking
- secret pattern masking (token-like strings / JWT-like payloads)
- value truncation for oversized fields

This is the primary guardrail against secret leakage in structured logs.

## Main Logging Surfaces

### Startup and server

- `cmd/run-agent/serve.go` emits startup/shutdown lifecycle logs.
- `internal/api/server.go` logs server listen/shutdown events.

### Runner orchestration

`internal/runner/*` logs:

- run directory allocation and run-slot lifecycle
- token/version validation warnings
- timeout/failure/completion summaries
- run event post failures

### Message bus internals

`internal/messagebus/messagebus.go` logs:

- lock timeout retries and retry recovery
- append failures / exhausted retries
- auto-rotation and rotation recovery failures

### API request plane

`internal/api/middleware.go` and handlers log:

- request completion and request errors
- key control-plane actions (task/runs/message posting)
- audit-safe contextual metadata (`project_id`, `task_id`, `run_id`, `request_id`)

## Non-Log Observability Artifacts

### Message bus event stream

Task and project buses are operational event streams:

- `<root>/<project>/<task>/TASK-MESSAGE-BUS.md`
- `<root>/<project>/PROJECT-MESSAGE-BUS.md`

Runner lifecycle events:

- `RUN_START`
- `RUN_STOP`
- `RUN_CRASH`

### Per-run metadata and outputs

Each run directory includes:

- `run-info.yaml` (status, PID/PGID, timestamps, paths, exit code)
- `agent-stdout.txt`
- `agent-stderr.txt`
- `output.md`

`output.md` is guaranteed by fallback creation from stdout when missing.

### API audit trail

`internal/api` writes sanitized form-submission audit entries under `_audit/form-submissions.jsonl` in runs root.

## Practical Inspection Workflow

1. Inspect process logs (`run-agent serve`, runner invocations) for structured events.
2. Check task/project message bus files for lifecycle and coordination messages.
3. Read `run-info.yaml` for canonical run status and exit metadata.
4. Correlate with `agent-stdout.txt`, `agent-stderr.txt`, and `output.md`.

## Event Naming Guidance

For new events:

- keep names stable and machine-friendly
- use `_failed` suffix for failures
- prefer short, specific event names over free-form prose

---

Last updated: 2026-02-23
