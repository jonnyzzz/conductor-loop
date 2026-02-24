# API Request Lifecycle

This page describes how HTTP requests move through the API server, based on:

- `internal/api/routes.go`
- `internal/api/middleware.go`
- `internal/api/auth.go`
- `internal/api/path_security.go`
- `internal/api/handlers.go`
- `internal/api/handlers_projects.go`
- `internal/api/handlers_projects_messages.go`
- `internal/api/sse.go`
- `internal/api/handlers_metrics.go`

## 1. Middleware Entry Chain

`Server.routes()` builds a `http.ServeMux`, then wraps it in this order:

1. `withAuth(...)`
2. `withCORS(...)`
3. `withLogging(...)`

Because wrappers are applied inside-out, runtime request flow is:

1. `withLogging`
2. `withCORS`
3. `withAuth`
4. route handler (`mux`)

Key behavior in each layer:

- `withLogging`:
  - Reads `X-Request-ID` or generates one (`req-<unixnano>-<rand>`).
  - Stores request ID in context.
  - Sets `X-Request-ID` response header.
  - Records completion log (method, path, status, bytes, duration, IDs).
  - Calls `s.metrics.RecordRequest(method, status)`.
- `withCORS`:
  - Validates `Origin` against configured allowlist.
  - Adds CORS headers when allowed.
  - Short-circuits all `OPTIONS` requests with `204 No Content`.
- `withAuth`:
  - Applies `RequireAPIKey(key)` middleware.
  - If key is empty, auth is effectively disabled (pass-through).

## 2. Routing and REST Handler Dispatch

`routes.go` registers top-level routes on `ServeMux`, including:

- `/metrics` -> `handleMetrics` (not wrapped with `s.wrap`)
- `/api/v1/*` health/status/version/tasks/runs/messages endpoints
- `/api/v1/runs/stream/all` (SSE)
- `/api/v1/messages/stream` (SSE)
- `/api/projects/*` project-centric API via `handleProjectsRouter`

Most API handlers are bound as `s.wrap(handlerFunc)`, where `handlerFunc` returns `*apiError`.
`s.wrap` standardizes error serialization via `writeError(...)` into JSON:

```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "..."
  }
}
```

`decodeJSON(...)` adds request-shape guardrails for JSON endpoints:

- 1 MiB max body (`io.LimitReader`)
- unknown fields rejected (`DisallowUnknownFields`)
- trailing bytes rejected

## 3. Path Confinement and Identifier Validation Protections

Security checks are layered:

- `validateIdentifier(value, name)`:
  - trims and URL-unescapes value
  - rejects empty values
  - rejects `/`, `\`, and `..`
  - used for `project_id`, `task_id`, and related path/query IDs
- `joinPathWithinRoot(root, segments...)`:
  - computes a clean joined path
  - enforces that target remains inside configured root
- `requirePathWithinRoot(root, target, field)`:
  - returns `403 FORBIDDEN` if path escapes root

Project/task lookup also keeps traversal bounded:

- `findProjectDir(...)` checks only root-bound candidates.
- `findProjectTaskDir(...)` checks root-bound candidates and depth-pruned walk.

Sensitive file-serving and message bus code paths re-check confinement before filesystem reads/writes:

- run/task file endpoints (`serveTaskFile`, `serveRunFile`, `serveRunFileStream`)
- message bus endpoints (`handleMessages`, `handlePostMessage`, `handleProjectMessages*`, `handleTaskMessages*`)

## 4. Optional API Key Model and Exempt Endpoints

Auth model in `auth.go`:

- If resolved API key is empty, middleware is a no-op.
- Accepted credentials:
  - `Authorization: Bearer <key>`
  - `X-API-Key: <key>`
- Rejection:
  - `401 Unauthorized`
  - `WWW-Authenticate: Bearer realm="conductor"`
  - JSON body: `{"error":"unauthorized","message":"valid API key required"}`

Auth-exempt paths:

- `/api/v1/health`
- `/api/v1/version`
- `/metrics`
- `/ui/` (prefix match)

`OPTIONS` requests are also exempt from API key enforcement.

## 5. SSE Streaming Architecture

All SSE flows use `newSSEWriter(...)`, which sets:

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`
- `X-Accel-Buffering: no`

### 5.1 Run Stream (`/api/v1/runs/{run_id}/stream`)

Path:

1. Route -> `handleRunByID` -> `streamRun(...)`
2. Parse `Last-Event-ID` as a run cursor (`parseCursor`)
3. Subscribe via `StreamManager.SubscribeRun(runID, cursor)`
4. Emit:
   - `event: log` with cursor IDs (`id: s=<stdout>;e=<stderr>`)
   - `event: status` when run status/exit changes
   - `event: heartbeat` periodically

If cursor is behind, catch-up is replayed from log files before live streaming resumes.

### 5.2 All-Runs Stream (`/api/v1/runs/stream/all`)

Path:

1. Subscribe to all currently known runs (`listRunIDs` + `SubscribeRun`)
2. Fan-in all run subscriptions into one outbound channel
3. Start `RunDiscovery` polling for newly created runs
4. Auto-subscribe to newly discovered runs
5. Forward all events plus periodic heartbeats

This stream does not consume `Last-Event-ID` for global resume.

### 5.3 Message Stream

Endpoints:

- `/api/v1/messages/stream?project_id=...&task_id=...`
- `/api/projects/{project}/messages/stream`
- `/api/projects/{project}/tasks/{task}/messages/stream`

All route to `streamMessageBusPath(...)`:

1. Resolve and confine bus path (`PROJECT-MESSAGE-BUS.md` or `TASK-MESSAGE-BUS.md`)
2. Read `Last-Event-ID` as last message ID
3. Poll message bus at configured interval
4. Emit `event: message` with `id: <msg_id>` and JSON payload
5. Emit periodic `event: heartbeat`

If `Last-Event-ID` no longer exists (`ErrSinceIDNotFound`), server resets cursor and continues polling.

### 5.4 SSE Tunables and Limits

From `SSEConfig` (with defaults):

- Poll interval: `100ms`
- Discovery interval: `1s`
- Heartbeat interval: `30s`
- Max clients per run: `10` (`ErrMaxClientsReached` -> HTTP `429`)

## 6. Prometheus `/metrics` Scrape Path

`/metrics` is served by `handleMetrics`:

- method: `GET` only (else `405`)
- content type: `text/plain; version=0.0.4`
- payload: `s.metrics.Render()`

Although `/metrics` uses a direct handler (no `s.wrap`), it still traverses global middleware:

- request ID/logging
- CORS
- auth middleware (path is exempt when API key auth is enabled)

## Sequence Diagrams

### REST Request

```text
Client
  |
  | HTTP request
  v
withLogging
  |-- read/generate X-Request-ID
  |-- add request ID to context + response header
  v
withCORS
  |-- [OPTIONS] -> 204 No Content (stop)
  |-- [other]   -> next
  v
withAuth
  |-- [key disabled/exempt/valid] -> next
  |-- [invalid or missing key]    -> 401 JSON (stop)
  v
ServeMux route match
  |-- /metrics -> handleMetrics
  |-- /api/... -> s.wrap(handler)
  v
handler
  |-- success -> response body/status
  |-- *apiError -> writeError JSON payload
  v
withLogging finalize
  |-- structured request log
  |-- metrics.RecordRequest(method,status)
  v
Client response (includes X-Request-ID)
```

### SSE Stream

```text
Client                        API SSE Endpoint                     Backend
  | GET /stream (+ Last-Event-ID)   |                               |
  |-------------------------------->| newSSEWriter headers          |
  |                                 |------------------------------->|
  |                                 | init subscription/poller      |
  |                                 |                               |
  |                                 | [run stream] SubscribeRun(run,cursor)
  |                                 | [all-runs] subscribe all + RunDiscovery
  |                                 | [messages] poll MessageBus.ReadMessages(lastID)
  |                                 |                               |
  |<--------------------------------| event: log/status/message     |
  |<--------------------------------| id: <cursor or msg_id>        |
  |<--------------------------------| data: {...}                   |
  |<--------------------------------| event: heartbeat (periodic)   |
  |                                 |                               |
  | disconnect / network error      |                               |
  |------------------------------X  | cleanup subscriptions/tickers |
```
