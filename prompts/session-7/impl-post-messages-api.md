# Task: Add POST /api/v1/messages Endpoint

## Objective
Add a POST endpoint to allow submitting messages to the project or task message bus via the REST API.

## Required Reading
1. Read `/Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go` — existing routes
2. Read `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go` — existing handlers
3. Read `/Users/jonnyzzz/Work/conductor-loop/internal/api/server.go` — server options and structure
4. Read `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go` — message bus API
5. Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools-QUESTIONS.md` Q2, Q4
6. Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui-QUESTIONS.md` Q4

## Human Answers
From message-bus-tools Q2:
> "Yes. The user should be able to post a message with type user or issue to the message bus of project/task levels."

From message-bus-tools Q4:
> "yes [standardize on START/STOP/CRASH with structured metadata]"

From monitoring-ui Q4:
> "Message posting should be independent from the messages listening, just implement independent logic to post messages to message bus (there is the same for CLI) and yet another logic to monitor that from the web"

## Implementation

### Route
Add to `internal/api/routes.go`:
```go
mux.Handle("POST /api/v1/messages", s.wrap(s.handlePostMessage))
```

Note: Check how existing routes handle POST vs GET. Some Go mux patterns require adding the HTTP method. Look at how the existing routes are registered.

### Request Body
```json
{
  "project_id": "my-project",
  "task_id": "task-001",      // optional
  "run_id": "run-123",        // optional
  "type": "USER",             // message type: USER, INFO, FACT, DECISION, ERROR, etc.
  "body": "Message content"   // required, non-empty
}
```

### Response (201 Created)
```json
{
  "msg_id": "MSG-...",
  "timestamp": "2026-02-20T17:00:00Z"
}
```

### Error responses
- 400 Bad Request: missing project_id, missing body, empty body
- 500 Internal Server Error: failed to write to message bus

### Handler Implementation
In `internal/api/handlers.go`, add `handlePostMessage`:
1. Parse JSON body into struct with fields: project_id, task_id, run_id, type, body
2. Validate: project_id required, body required and non-empty
3. Determine bus path:
   - If task_id provided: `{rootDir}/{project_id}/{task_id}/TASK-MESSAGE-BUS.md`
   - If only project_id: `{rootDir}/{project_id}/PROJECT-MESSAGE-BUS.md`
4. Create/open the MessageBus
5. AppendMessage with the given fields
6. Return 201 with msg_id and timestamp

### Note on rootDir
The handler needs access to the server's root directory. Check how existing handlers access rootDir — look for `s.rootDir` or similar in the `Server` struct.

## Tests
Read `internal/api/handlers_test.go` or `internal/api/http_test.go` for existing test patterns.

Add to the test file:
1. `TestPostMessage_Success` — post valid message, verify 201 and msg_id returned
2. `TestPostMessage_MissingProjectID` — verify 400 error
3. `TestPostMessage_EmptyBody` — verify 400 error
4. `TestPostMessage_WithTaskID` — verify message goes to task-level bus

## Quality Gates
- `go build ./...` passes
- `go test ./internal/api/` passes
- `go vet ./internal/api/` passes

## Commit Format
```
feat(api): add POST /api/v1/messages endpoint

- Accept {project_id, task_id?, run_id?, type, body}
- Route to project or task message bus based on task_id presence
- Return 201 with {msg_id, timestamp}
- Add tests for success and error cases

Implements: message-bus-tools Q2, monitoring-ui Q4
```

## Write Output
Write output.md with summary of what was implemented, including the request/response format.
