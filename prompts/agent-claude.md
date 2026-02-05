# Task: Implement Claude Agent Backend

**Task ID**: agent-claude
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Claude agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-claude.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/claude/`
Files:
- claude.go - Claude agent implementation

### 2. Claude Agent
```go
type ClaudeAgent struct {
    token string
    model string
}

func (a *ClaudeAgent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
    // Execute claude CLI with prompt
    // Redirect stdio to files
    // Return on completion
}

func (a *ClaudeAgent) Type() string {
    return "claude"
}
```

### 3. CLI Integration
- Use `claude` CLI binary
- Pass prompt via stdin
- Set working directory with `-C` flag
- Capture stdout/stderr to files

### 4. Tests Required
Location: `test/integration/agent_claude_test.go`
- TestClaudeExecution (requires claude CLI)
- TestClaudeStdioCapture

## Success Criteria
- All tests pass
- Claude CLI integration working
- Stdio properly captured

## References
- docs/specifications/subsystem-agent-backend-claude.md

## Output
Log to MESSAGE-BUS.md:
- FACT: Claude agent backend implemented
- FACT: Integration tests passing
