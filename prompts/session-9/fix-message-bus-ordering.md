# Task: Fix Flaky TestMessageBusOrdering Test

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/test/integration/messagebus_concurrent_test.go
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go

## Problem

`TestMessageBusOrdering` in `test/integration/messagebus_concurrent_test.go` (line 37) is FLAKY.

The test:
1. Starts 5 goroutines, each writing 20 messages concurrently
2. Records returned message IDs in `order []string` slice (with mutex)
3. Reads all messages from file
4. Asserts `order[i] == messages[i].MsgID` for all i

**Root cause**: The `order` slice records msg IDs in the order goroutines RETURN from `AppendMessage`.
But the file records messages in the order goroutines ACQUIRE the file lock.
These two orderings are different because:
- Goroutine A acquires lock, writes, releases lock (→ message appears in file)
- Goroutine B may get scheduled between A releasing and A returning
- Goroutine B acquires lock, writes, releases
- Now file has: [A's msg, B's msg]
- But goroutines may update `order` as [B's ID, A's ID]

So the ordering check is fundamentally incorrect for concurrent writers.

## Fix Required

Modify `TestMessageBusOrdering` to NOT assert ordering between concurrent writers.

**Option 1 (Preferred)**: Split into two tests:
- `TestMessageBusOrderingSingleWriter`: Single goroutine writes N messages sequentially. Assert they appear in file in write order. This IS deterministic.
- `TestMessageBusConcurrentCompleteness`: Multiple concurrent writers. Assert all messages appear in file (as a set). No ordering assertion.

**Option 2**: Simply remove the ordering assertion from the concurrent test and add a set membership check.

### Implementation Details

For the concurrent test, instead of:
```go
for i, msg := range messages {
    if order[i] != msg.MsgID {
        t.Fatalf("message order mismatch at %d: got %q want %q", i, msg.MsgID, order[i])
    }
}
```

Use set membership:
```go
orderSet := make(map[string]bool, len(order))
for _, id := range order {
    orderSet[id] = true
}
for _, msg := range messages {
    if !orderSet[msg.MsgID] {
        t.Fatalf("message %q in file but not in written set", msg.MsgID)
    }
}
for _, id := range order {
    found := false
    for _, msg := range messages {
        if msg.MsgID == id {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("message %q written but not found in file", id)
    }
}
```

Also add `TestMessageBusOrderingSingleWriter`:
```go
func TestMessageBusOrderingSingleWriter(t *testing.T) {
    path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
    bus, err := messagebus.NewMessageBus(path)
    if err != nil {
        t.Fatalf("new message bus: %v", err)
    }

    const numMsgs = 20
    written := make([]string, numMsgs)
    for i := 0; i < numMsgs; i++ {
        msgID, err := bus.AppendMessage(&messagebus.Message{
            Type:      "FACT",
            ProjectID: "project",
            TaskID:    "task-order",
            Body:      fmt.Sprintf("message %d", i),
        })
        if err != nil {
            t.Fatalf("append message %d: %v", i, err)
        }
        written[i] = msgID
    }

    messages, err := bus.ReadMessages("")
    if err != nil {
        t.Fatalf("read messages: %v", err)
    }
    if len(messages) != numMsgs {
        t.Fatalf("expected %d messages, got %d", numMsgs, len(messages))
    }
    for i, msg := range messages {
        if written[i] != msg.MsgID {
            t.Fatalf("order mismatch at %d: got %q want %q", i, msg.MsgID, written[i])
        }
    }
}
```

## Steps

1. Read `test/integration/messagebus_concurrent_test.go` in full
2. Modify `TestMessageBusOrdering` to check set membership instead of ordering
3. Add `TestMessageBusOrderingSingleWriter` test to the same file
4. Run tests to verify both pass: `go test ./test/integration/ -run TestMessageBusOrdering -v`
5. Run all tests: `go test ./...` (should all pass)
6. Commit with: `test(messagebus): fix flaky TestMessageBusOrdering concurrent ordering`

## Quality Gates

- All tests must pass: `go test ./...`
- The fixed test must pass reliably (not flaky)
- No race conditions: `go test -race ./test/integration/`
- Follow AGENTS.md commit format

## Absolute Paths

- Test file: /Users/jonnyzzz/Work/conductor-loop/test/integration/messagebus_concurrent_test.go
- MessageBus: /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go
- AGENTS.md: /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
