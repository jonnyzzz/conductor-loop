# Problem: Process Detachment vs Wait Contradiction

## Context
From subsystem-runner-orchestration.md:30:
> "Detaches from controlling terminal but still waits on agent PID"

## Problem
These seem contradictory in Unix process model.

## Clarification Needed
**"Detach" likely means**:
- `setsid()` - Create new process group/session
- NOT daemonization (no double-fork)

Still allows parent to `waitpid()` on child

## Your Task
Confirm interpretation and specify exact syscalls:
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setsid: true,  // ← This is "detach"
}
```

Specify in subsystem-runner-orchestration.md what "detach" means precisely.
