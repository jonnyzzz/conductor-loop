# Task: Implement Ralph Loop

**Task ID**: runner-ralph
**Phase**: Runner Orchestration
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: runner-process, messagebus

## Objective
Implement the Ralph Loop (Root Agent Loop) with wait-without-restart pattern.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration.md
Read: docs/decisions/problem-2-FINAL-DECISION.md (DONE + children handling)

## Required Implementation

### 1. Package Structure
Location: `internal/runner/`
Files:
- ralph.go - Ralph Loop implementation
- wait.go - Child process waiting logic

### 2. Ralph Loop
```go
type RalphLoop struct {
    runDir        string
    messagebus    *messagebus.MessageBus
    maxRestarts   int
    waitTimeout   time.Duration // 300s default
}

func (rl *RalphLoop) Run(ctx context.Context) error {
    // 1. Check for DONE file
    // 2. If DONE exists:
    //    - Wait for children (up to waitTimeout)
    //    - Check children with kill(-pgid, 0)
    //    - Return when all children exit or timeout
    // 3. If no DONE:
    //    - Check if process should restart
    //    - Respect maxRestarts limit
    //    - Restart if needed
}
```

### 3. DONE File Detection
- Check for DONE file in run directory
- File presence signals "don't restart"
- Must still wait for children

### 4. Child Waiting
- Use kill(-pgid, 0) to detect children
- Poll every 1 second
- Timeout after 300 seconds (configurable)
- Return early if all children exit

### 5. Restart Logic
- Count restarts, enforce maxRestarts limit
- Log restart events to message bus
- Exponential backoff optional

### 6. Tests Required
Location: `test/unit/ralph_test.go`
- TestRalphLoopDONEWithChildren
- TestRalphLoopDONEWithoutChildren
- TestRalphLoopRestartLogic
- TestChildWaitTimeout

## Success Criteria
- All tests pass
- DONE + children scenario working
- 300s timeout enforced

## References
- docs/decisions/problem-2-FINAL-DECISION.md

## Output
Log to MESSAGE-BUS.md:
- FACT: Ralph Loop implemented
- FACT: Wait-without-restart working
