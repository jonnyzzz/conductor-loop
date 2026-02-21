# Task: Full SSE Payload for Message Stream (Q5)

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/internal/api/sse.go
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools-QUESTIONS.md

## Human Decision (Q5 from subsystem-message-bus-tools-QUESTIONS.md)

> The SSE message stream currently sends `message` events with `{msg_id, content, timestamp}` and does not set an SSE `id`. Should the stream emit full message payloads and include `id` for resumable clients?
> **Answer**: yes

## Current State

In `internal/api/sse.go`, the `streamMessages` function sends:
```go
payload := messagePayload{
    MsgID:     msg.MsgID,
    Content:   msg.Body,
    Timestamp: ts.Format(time.RFC3339Nano),
}
```

Where `messagePayload` is:
```go
type messagePayload struct {
    MsgID     string `json:"msg_id"`
    Content   string `json:"content"`
    Timestamp string `json:"timestamp"`
}
```

And the SSE event does NOT set the `id` field.

## What to Implement

### 1. Expand `messagePayload` to full message payload

```go
type messagePayload struct {
    MsgID      string            `json:"msg_id"`
    Timestamp  string            `json:"timestamp"`
    Type       string            `json:"type,omitempty"`
    ProjectID  string            `json:"project_id,omitempty"`
    TaskID     string            `json:"task_id,omitempty"`
    RunID      string            `json:"run_id,omitempty"`
    Parents    []string          `json:"parents,omitempty"`   // string IDs for simplicity
    Attachment string            `json:"attachment,omitempty"`
    Meta       map[string]string `json:"meta,omitempty"`
    Content    string            `json:"content"`             // Body text
}
```

Note: If the message model has been extended with `Parent` struct (from Q3 work), use the ID list for simplicity in JSON.

### 2. Set SSE `id` to `msg_id` for resumable clients

In `streamMessages`, update the SSEEvent:
```go
ev := SSEEvent{
    ID:    msg.MsgID,  // <-- ADD THIS
    Event: "message",
    Data:  string(data),
}
```

This allows clients to send `Last-Event-ID: <msg_id>` to resume from a specific message.

### 3. Update `streamMessages` to populate full payload

```go
// Get parent msg IDs (string list for JSON simplicity)
parentIDs := make([]string, 0, len(msg.ParentMsgIDs))
parentIDs = append(parentIDs, msg.ParentMsgIDs...)

payload := messagePayload{
    MsgID:      msg.MsgID,
    Timestamp:  ts.Format(time.RFC3339Nano),
    Type:       msg.Type,
    ProjectID:  msg.ProjectID,
    TaskID:     msg.TaskID,
    RunID:      msg.RunID,
    Parents:    parentIDs,
    Attachment: msg.Attachment,
    Content:    msg.Body,
}
```

Note: Check the current Message struct fields. If `ParentMsgIDs` has been renamed/refactored (from Q3 work being done in parallel), use the appropriate field. Be defensive: use `msg.ParentMsgIDs` if it exists, or build parent IDs from the Parents slice if struct was updated.

### 4. Update `handleMessages` GET endpoint too

In `internal/api/handlers.go`, the `handleMessages` function returns messages. Update it to return full message objects (not just trimmed data). Check what `handleMessages` returns currently.

### 5. Update SSE tests

In `internal/api/sse_test.go`, update tests that check the `message` event payload to include the new fields.

## Implementation Steps

1. Read `internal/api/sse.go` in full (you'll need to read lines carefully)
2. Read `internal/api/handlers.go` to understand `handleMessages`
3. Read `internal/api/sse_test.go` to understand existing tests
4. Read `internal/messagebus/messagebus.go` to see current `Message` struct fields
5. Update `messagePayload` struct with new fields
6. Update `streamMessages` to populate all fields and set SSE `id`
7. Update tests if they check the exact payload shape
8. Run: `go test ./internal/api/ -v`
9. Run: `go test ./...` (all tests pass)
10. Run: `go test -race ./internal/api/`
11. Commit with: `feat(api): send full message payload in SSE stream with id field`

## Quality Gates

- `go build ./...` must pass
- `go test ./...` must pass
- `go test -race ./internal/api/` must pass (no races)
- The SSE message event must include `id:` field
- The payload must include all message fields

## Notes

- The `messagePayload` struct is defined in `internal/api/sse.go` around line 366-370
- The SSE writer already supports the `ID` field (see `sseWriter.Send()`)
- The `streamMessages` function is around line 225 in `sse.go`
- Be careful: Q3 work (extend-message-model.md) may change the `Message` struct concurrently. If the struct has changed, adapt accordingly. Use defensive field access.

## Absolute Paths

- SSE handler: /Users/jonnyzzz/Work/conductor-loop/internal/api/sse.go
- Handlers: /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go
- SSE tests: /Users/jonnyzzz/Work/conductor-loop/internal/api/sse_test.go
- MessageBus: /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go
- Questions file: /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools-QUESTIONS.md
- AGENTS.md: /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
