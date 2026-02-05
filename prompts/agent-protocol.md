# Task: Implement Agent Protocol Interface

**Task ID**: agent-protocol
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement the common agent protocol interface that all agent backends will implement.

## Specifications
Read: docs/specifications/subsystem-agent-protocol.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/`
Files:
- agent.go - Agent interface and context types
- executor.go - Execution helper functions
- stdio.go - Stdout/stderr capture utilities

### 2. Agent Interface
```go
type Agent interface {
    // Execute runs the agent with the given context
    Execute(ctx context.Context, runCtx *RunContext) error

    // Type returns the agent type (claude, codex, etc.)
    Type() string
}

type RunContext struct {
    RunID       string
    ProjectID   string
    TaskID      string
    Prompt      string
    WorkingDir  string
    StdoutPath  string
    StderrPath  string
    Environment map[string]string
}
```

### 3. Executor Functions
- SpawnProcess(cmd, args, stdin, stdout, stderr) - Process spawning with setsid
- CaptureOutput(stdout, stderr, files) - Stdio redirection
- CreateOutputMD(runDir, fallback) - Runner fallback for output.md

### 4. Tests Required
Location: `test/unit/agent_test.go`
- TestAgentInterface
- TestRunContext
- TestSpawnProcess
- TestCaptureOutput

## Success Criteria
- All tests pass
- Interface documented
- IntelliJ MCP review: no warnings

## References
- docs/specifications/subsystem-agent-protocol.md
- docs/decisions/problem-5-DECISION.md (output.md responsibility)

## Output
Log to MESSAGE-BUS.md:
- FACT: Agent protocol interface implemented
- FACT: N unit tests passing
