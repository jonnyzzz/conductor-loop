# Task: Add Claude JSON Streaming Support (--output-format stream-json)

## Background

You are an implementation agent working on the conductor-loop project.

- Project root: /Users/jonnyzzz/Work/conductor-loop
- Build: `go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent`
- Test: `go test ./...`
- Commit format: `<type>(<scope>): <subject>` (from AGENTS.md)

The conductor-loop project orchestrates AI agents. The Claude backend currently uses
`--output-format text` which gives a plain text response on stdout. The project spec
(docs/specifications/subsystem-agent-backend-claude-QUESTIONS.md) requires switching to
`--output-format stream-json` to capture progress messages (tool calls, token usage, etc.).

## Current Claude Invocation

See `internal/agent/claude/claude.go` function `claudeArgs()`:
```
claude -C <cwd> -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < prompt.md
```

## Required Change

Switch to JSON streaming output:
```
claude -C <cwd> -p --input-format text --output-format stream-json --verbose --tools default --permission-mode bypassPermissions < prompt.md
```

When `--output-format stream-json` is used, stdout becomes a stream of newline-delimited JSON objects (ndjson). Example format:

```json
{"type":"system","subtype":"init","session_id":"xxx","tools":[...],"model":"claude-opus-4-5"}
{"type":"assistant","message":{"id":"msg_xxx","type":"message","role":"assistant","content":[{"type":"text","text":"I'll start by..."}],"usage":{"input_tokens":100,"output_tokens":20}},"session_id":"xxx"}
{"type":"assistant","message":{"id":"msg_xxx","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_xxx","name":"Write","input":{"file_path":"/tmp/output.md","content":"Final answer"}}],"usage":{"input_tokens":120,"output_tokens":50}},"session_id":"xxx"}
{"type":"tool_result","tool_use_id":"toolu_xxx","is_error":false,"session_id":"xxx"}
{"type":"result","subtype":"success","is_error":false,"result":"The final answer text","session_id":"xxx","total_cost_usd":0.001,"duration_ms":5000,"usage":{"input_tokens":200,"output_tokens":100}}
```

Key event types:
- `type=system`: initialization, lists tools and model
- `type=assistant`: assistant messages, may contain text or tool_use blocks
- `type=tool_result`: result of a tool call
- `type=result`: FINAL event, `result` field contains the final text response

## What to Implement

### 1. Update `claudeArgs()` in `internal/agent/claude/claude.go`

Change `--output-format text` to `--output-format stream-json` and add `--verbose`.

Remove the `--output-format text` and `--input-format text` flags from the args list, replace with:
```
--output-format stream-json --verbose
```

Keep `--input-format text` for stdin (prompt is still plain text on stdin).

Actually check what flags Claude uses - read the current claudeArgs() implementation first.

### 2. Create `internal/agent/claude/stream_parser.go`

Add a JSON stream parser that extracts the final output text:

```go
package claude

import (
    "bufio"
    "bytes"
    "encoding/json"
    "strings"
)

// streamEvent represents a single JSON event from Claude's stream-json output.
type streamEvent struct {
    Type    string          `json:"type"`
    Subtype string          `json:"subtype,omitempty"`
    Result  string          `json:"result,omitempty"`
    IsError bool            `json:"is_error,omitempty"`
    Message json.RawMessage `json:"message,omitempty"`
}

// assistantMessage is the nested message within a stream event.
type assistantMessage struct {
    Content []messageContent `json:"content"`
}

// messageContent is a content block within an assistant message.
type messageContent struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}

// ParseStreamJSON extracts the final human-readable text from Claude's
// --output-format stream-json output. It returns the text from the "result"
// event if found, otherwise concatenates text blocks from assistant messages.
// Returns ("", false) if no useful text is found.
func ParseStreamJSON(data []byte) (string, bool) {
    // ... implementation
}
```

The parser should:
1. Scan line by line (ndjson format)
2. For each line, try to parse as JSON
3. If `type=result` and `is_error=false`: return `result` field immediately
4. If `type=assistant`: extract text content blocks, accumulate in a buffer (fallback)
5. After all lines, return accumulated text if no result event found

### 3. Add post-processing in `internal/agent/claude/claude.go`

After the process completes in `Execute()`, add output extraction:

```go
// After waitForProcess returns, extract output.md from JSON stream if not already written
runDir := filepath.Dir(runCtx.StdoutPath)
if writeErr := writeOutputMDFromStream(runDir, runCtx.StdoutPath); writeErr != nil {
    // Non-fatal: output.md may be missing but that's handled by the fallback
    // Just log to stderr (we don't have a logger here, so skip silently)
    _ = writeErr
}
```

Function `writeOutputMDFromStream(runDir, stdoutPath string) error`:
1. Check if `filepath.Join(runDir, "output.md")` already exists → return nil (don't overwrite)
2. Read stdoutPath file contents
3. Call `ParseStreamJSON(contents)` → get text, ok
4. If ok: create output.md with extracted text → return nil
5. If not ok: return error (caller silently ignores it)

### 4. Create `internal/agent/claude/stream_parser_test.go`

Test cases for ParseStreamJSON:
- Valid stream with result event: should return the result text
- Valid stream with only assistant messages (no result): should concatenate text
- Empty input: should return ("", false)
- Invalid JSON lines: should skip gracefully and use other lines
- is_error=true result: should return ("", false)
- Multiple assistant messages: should concatenate all text blocks

### 5. Update `internal/agent/claude/claude_test.go`

Add tests for the new `writeOutputMDFromStream` function.

## Important Constraints

1. **Do NOT break existing behavior**: if ParseStreamJSON fails for any reason, the existing fallback still works
2. **Do NOT change the `CreateOutputMD` function** - it's generic and used by other agents
3. **The change must be backward-compatible**: if a user upgrades conductor-loop, existing runs in the `runs/` directory are unaffected
4. **Tests must pass**: run `go test ./internal/agent/claude/...` after implementing

## Quality Gates

After implementing:
1. `go build ./...` must pass
2. `go test ./internal/agent/claude/...` must pass
3. `go test -race ./internal/agent/claude/...` must pass
4. `go vet ./internal/agent/claude/...` must pass

## Files to Read First

1. `/Users/jonnyzzz/Work/conductor-loop/internal/agent/claude/claude.go` - current implementation
2. `/Users/jonnyzzz/Work/conductor-loop/internal/agent/claude/claude_test.go` - existing tests
3. `/Users/jonnyzzz/Work/conductor-loop/internal/agent/executor.go` - CreateOutputMD
4. `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` - how finalizeRun works

## Commit

After all tests pass, commit with:
```
feat(agent): add stream-json output parsing for Claude backend

- Switch Claude invocation to --output-format stream-json --verbose
- Add ParseStreamJSON to extract final text from JSON stream events
- Write output.md from parsed result after execution (if not present)
- Fallback: if parsing fails, existing CreateOutputMD handles agent-stdout.txt

Resolves TODO from docs/specifications/subsystem-agent-backend-claude-QUESTIONS.md
```

## Done Signal

Create a `DONE` file in the task directory (path is in TASK_FOLDER env var):
```bash
touch "$TASK_FOLDER/DONE"
```
