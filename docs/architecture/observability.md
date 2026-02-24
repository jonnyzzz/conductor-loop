# Observability Architecture

This page describes the current observability design based on:

- `internal/metrics/metrics.go`
- `internal/api/handlers_metrics.go`
- `internal/api/middleware.go`
- `internal/api/request_id.go`
- `internal/api/audit.go`
- `internal/obslog/obslog.go`
- `internal/api/routes.go`
- `README.md`

## 1. Observability Surfaces

Conductor Loop exposes three primary observability channels:

- Prometheus-compatible metrics at `GET /metrics`
- Structured runtime logs via `internal/obslog` (`key=value`, centralized redaction)
- Sanitized JSONL audit records for form submissions at `<root>/_audit/form-submissions.jsonl`

Request correlation ties these together through `X-Request-ID`.

## 2. Prometheus Metrics (`/metrics`)

`internal/api/routes.go` registers `/metrics` directly to `handleMetrics`, and `internal/api/handlers_metrics.go` implements:

- Method: `GET` only (`405` for other methods)
- Content type: `text/plain; version=0.0.4`
- Payload: `s.metrics.Render()`

`internal/metrics/metrics.go` renders Prometheus text format metric families:

- `conductor_uptime_seconds` (gauge)
- `conductor_active_runs_total` (gauge)
- `conductor_completed_runs_total` (counter)
- `conductor_failed_runs_total` (counter)
- `conductor_messagebus_appends_total` (counter)
- `conductor_queued_runs_total` (gauge)
- `conductor_api_requests_total{method,status}` (counter)
- `conductor_agent_runs_total{agent_type}` (counter, emitted when data exists)
- `conductor_agent_fallbacks_total{from_type,to_type}` (counter, emitted when data exists)

`internal/api/middleware.go` records `conductor_api_requests_total` for completed requests via `s.metrics.RecordRequest(method, status)`.

## 3. Structured Logging (`internal/obslog`)

`internal/obslog/obslog.go` emits log lines as normalized `key=value` pairs with a fixed envelope:

- `ts`
- `level`
- `subsystem`
- `event`

Fields added by callers are normalized to lowercase underscore keys, then redacted/sanitized before output.

Redaction and sanitization behavior:

- Key-based redaction: keys containing sensitive parts (for example `token`, `password`, `api_key`, `authorization`, `cookie`, `session`) are replaced with `[REDACTED]`
- Pattern-based redaction: bearer tokens and common token/JWT-like patterns in values are replaced with `[REDACTED]`
- Value truncation: values longer than `2048` runes are truncated with `...[TRUNCATED]`

In `internal/api/middleware.go`, request completion logs use this logger (`event=request_completed`) and include request correlation fields (`request_id`, `correlation_id`) plus method/path/status/bytes/duration and extracted project/task/run identifiers.

## 4. Form Submission Audit Log

`internal/api/audit.go` writes form submission audit records to:

- `<root>/_audit/form-submissions.jsonl`

Path details:

- Directory: `_audit`
- Filename: `form-submissions.jsonl`
- Fully resolved by `filepath.Join(s.rootDir, formSubmissionAuditDir, formSubmissionAuditFilename)`

Record details (JSONL, one object per line) include:

- Timestamp and request correlation (`request_id`, `correlation_id`)
- HTTP metadata (`method`, `path`, `endpoint`, `remote_addr`)
- Entity identifiers (`project_id`, `task_id`, `run_id`, `message_id`)
- Sanitized payload (`payload`)

Audit payload redaction behavior is explicit and independent from runtime log formatting:

- Sensitive-key redaction to `[REDACTED]` using the same key-part model (`token`, `secret`, `password`, `api_key`, etc.)
- String pattern redaction for bearer/JWT/token-like values
- String truncation at `4096` runes with `...[TRUNCATED]`

Write behavior is append-only JSONL with flush semantics:

- Creates parent directory (`0755`) if needed
- Opens file in append/create/write mode (`0640`)
- Appends newline-delimited JSON
- Calls `file.Sync()` after write

If audit write fails, API logs `event=audit_write_failed` through `obslog`.

## 5. Request Correlation (`X-Request-ID`)

`internal/api/request_id.go` and `internal/api/middleware.go` implement correlation:

- Header name: `X-Request-ID`
- If client provides it, server reuses it
- If missing, server generates `req-<unixnano>-<random>` (with monotonic counter fallback)
- ID is stored in request context and returned in response headers

The same ID is propagated as both `request_id` and `correlation_id` in:

- API completion logs (`withLogging`)
- Form submission audit records (`writeFormSubmissionAudit`)

This gives a single join key across HTTP responses, structured logs, and audit records.

## 6. Health and Version Endpoints

`internal/api/routes.go` exposes:

- `/api/v1/health`
- `/api/v1/version`

Both routes are wrapped by the API middleware chain, so they participate in request logging, request metrics, and request ID correlation.

Per `README.md`, `/api/v1/health`, `/api/v1/version`, and `/metrics` remain publicly accessible even when API key auth is enabled, supporting health checks and scraping.

## 7. Data-Flow Diagram

```text
Client / Prometheus
       |
       | HTTP request (optional X-Request-ID)
       v
withLogging (middleware)
  - read/generate request ID
  - set response header X-Request-ID
       |
       v
withCORS -> withAuth -> routes mux
       |                    |
       |                    +--> GET /metrics -> handleMetrics -> metrics.Render()
       |                    |
       |                    +--> mutating API handler
       |                              |
       |                              +--> writeFormSubmissionAudit()
       |                                      - sanitize payload
       |                                      - append JSONL to <root>/_audit/form-submissions.jsonl
       |
       +--> on response completion:
              - obslog.Log(event=request_completed, request_id=...)
              - metrics.RecordRequest(method,status)
```
