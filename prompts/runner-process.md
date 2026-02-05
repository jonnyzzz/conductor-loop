# Task: Implement Process Management

**Task ID**: runner-process
**Phase**: Runner Orchestration
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol, storage, config

## Objective
Implement process spawning with setsid() and stdio redirection.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration.md
Read: docs/decisions/problem-7-DECISION.md (setsid not daemonization)

## Required Implementation

### 1. Package Structure
Location: `internal/runner/`
Files:
- process.go - Process spawning with setsid
- stdio.go - Stdio redirection to files
- pgid.go - Process group ID management

### 2. Process Spawning
```go
type ProcessManager struct {
    runDir string
}

func (pm *ProcessManager) SpawnAgent(ctx context.Context, agentType string, opts SpawnOptions) (*Process, error) {
    // Use exec.Cmd with SysProcAttr.Setsid = true
    // Redirect stdin/stdout/stderr to files
    // Track PID and PGID
    // Return Process handle
}
```

### 3. Setsid Implementation
- Use syscall.SysProcAttr{Setsid: true} on Unix
- Use CREATE_NEW_PROCESS_GROUP on Windows
- Do NOT daemonize (no double-fork)
- Terminal detachment only

### 4. Stdio Redirection
- Create agent-stdout.txt, agent-stderr.txt in run dir
- Open with O_APPEND for concurrent writes
- Use io.MultiWriter for tee-style logging if needed

### 5. Tests Required
Location: `test/unit/process_test.go`
- TestSpawnProcess
- TestProcessSetsid
- TestStdioRedirection
- TestProcessGroupManagement

## Success Criteria
- All tests pass
- Processes properly detached from terminal
- Stdio correctly captured to files

## References
- docs/decisions/problem-7-DECISION.md

## Output
Log to MESSAGE-BUS.md:
- FACT: Process management implemented
- FACT: Setsid working on Unix and Windows
