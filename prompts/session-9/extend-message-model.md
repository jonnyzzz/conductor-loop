# Task: Extend Message Bus Object Model (Q3)

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-object-model-QUESTIONS.md

## Human Decisions (from message-bus-object-model-QUESTIONS.md)

1. Q1: Support object-form parents with `kind`/`meta` and preserve them on read/write → **Yes**
2. Q2: `issue_id` is just an alias for `msg_id`, use it to make parsing easier where necessary → **Keep as alias**
3. Q3: Dependency kinds (depends_on, blocks, blocked_by, child_of) are **advisory only** — not enforced/validated

## What to Implement

### 1. Extend the `Message` struct in `internal/messagebus/messagebus.go`

Add these new fields:
- `IssueID string` — alias for `MsgID`, used for ISSUE-type messages (`yaml:"issue_id,omitempty"`)
- `Parents []Parent` — structured parents (replaces `ParentMsgIDs []string`)
- `Attachments []string` — list of attachment paths (replaces single `Attachment string`)
- `Links []Link` — advisory links to external resources
- `Meta map[string]string` — arbitrary metadata key-value pairs

**Parent struct**:
```go
// Parent represents a structured parent reference.
type Parent struct {
    MsgID string            `yaml:"msg_id"`
    Kind  string            `yaml:"kind,omitempty"` // e.g. "depends_on", "blocks", "child_of"
    Meta  map[string]string `yaml:"meta,omitempty"`
}
```

**Link struct**:
```go
// Link represents an advisory link.
type Link struct {
    URL   string `yaml:"url"`
    Label string `yaml:"label,omitempty"`
    Kind  string `yaml:"kind,omitempty"`
}
```

### 2. Keep backward compatibility

The `parents` YAML field currently accepts `[]string`. We need to support both formats:
- Old format (string list): `parents: ["msg-001", "msg-002"]`
- New format (object list): `parents: [{msg_id: "msg-001", kind: "depends_on"}]`

Use a custom YAML unmarshaler for the `parents` field.

### 3. Update `IssueID` logic

When `Type == "ISSUE"` and `IssueID == ""`, set `IssueID = MsgID` automatically in `AppendMessage`.
When marshaling to YAML, include `issue_id` only if set (omitempty).

### 4. Keep `Attachment` field for backward compatibility

Keep the existing `Attachment string` field but also support the new `Attachments []string`.
When reading old messages with single `attachment_path`, append to `Attachments`.

### 5. Update `AppendMessage` to write new fields

The message is written as YAML headers + body. Update the marshal/unmarshal to handle the new fields.

### 6. Add tests in `internal/messagebus/messagebus_test.go`

- Test that `Parents` with `kind` is preserved on write/read
- Test that old string-list `parents` is read as `Parents` objects
- Test that `IssueID` is set automatically for ISSUE messages
- Test that `Meta` map is written/read correctly
- Test that `Links` are preserved

## Implementation Steps

1. Read `internal/messagebus/messagebus.go` in full
2. Read `internal/messagebus/messagebus_test.go` to understand test patterns
3. Add `Parent`, `Link` structs before `Message` struct
4. Update `Message` struct with new fields
5. Implement custom YAML unmarshaling for backward-compatible `parents` field
6. Update `AppendMessage` to set `IssueID = MsgID` when `Type == "ISSUE"`
7. Update message serialization (marshal) to include new fields correctly
8. Add tests for new functionality
9. Run: `go test ./internal/messagebus/ -v`
10. Run: `go test ./... ` (all tests pass)
11. Run: `go test -race ./internal/messagebus/`
12. Commit with: `feat(messagebus): extend message model with structured parents, meta, links`

## Quality Gates

- `go build ./...` must pass
- `go test ./...` must pass
- `go test -race ./internal/messagebus/` must pass
- New fields must round-trip correctly (write then read back)
- Old messages without new fields must still parse correctly

## Notes

- Do NOT break the existing `AppendMessage` API signature
- Keep `ParentMsgIDs []string` as a deprecated read-path for backward compat, OR remove it if you can confirm nothing else uses it (check with grep)
- The `Attachment string` field: check usages with grep before removing
- If `parents` YAML field accepts both `[]string` and `[]Parent`, use `yaml.Node` for custom unmarshaling

## Absolute Paths

- MessageBus implementation: /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go
- MessageBus tests: /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus_test.go
- Questions file: /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-object-model-QUESTIONS.md
- AGENTS.md: /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
