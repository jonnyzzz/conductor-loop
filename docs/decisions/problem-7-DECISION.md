# Problem #7 Decision: Process Detachment Clarification

## Implementation Status (R3 - 2026-02-24)

- Clarification is implemented in code: Unix process setup uses `Setsid=true` (`internal/runner/pgid_unix.go`) while the runner still waits and records exits (`internal/runner/process.go`, `internal/runner/job.go`, `internal/runner/wrap.go`).
- Process-group termination is implemented on Unix via `kill(-pgid, SIGTERM/SIGKILL)` (`internal/runner/stop_unix.go`), with Windows-specific behavior in `internal/runner/stop_windows.go`.
- Related Ralph loop behavior is implemented in `internal/runner/ralph.go` + `internal/runner/wait.go` (DONE-aware child wait, no post-DONE restart).

## Assessment: **CLARIFICATION NEEDED, NOT A BUG**

### Current Specification
From subsystem-runner-orchestration.md:30:
> "Detaches from controlling terminal but still waits on agent PID"

This appears contradictory but is actually standard Unix behavior.

### Clarification: What "Detach" Means

**"Detach"** = `setsid()` - Create new process group/session

**NOT** = Daemonization (double-fork + redirect stdio)

### Unix Process Model

```go
cmd := exec.Command("agent-binary")
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setsid: true,  // ← This is "detach from controlling terminal"
}

// Parent can still:
cmd.Start()        // Fork + exec
err := cmd.Wait()  // Wait for child exit
```

**Effect of `Setsid: true`**:
- Child becomes session leader (new PGID = PID)
- Detached from parent's controlling terminal
- CTRL-C in parent terminal won't signal child
- Parent can still `waitpid()` on child

**This is NOT daemonization**:
- Child's parent is still the runner (not init/PID 1)
- Child's stdio can still be redirected to files
- Child exits → parent receives exit code

### Why This Approach?

1. **Isolation**: Child processes don't receive terminal signals (SIGINT, SIGHUP)
2. **Traceability**: Runner can still wait and collect exit code
3. **Cleanup**: Runner can signal entire process group (`kill(-pgid, SIGTERM)`)

### Specification Update

**File**: `subsystem-runner-orchestration.md`
**Line**: 30

**Current**:
```markdown
- Detaches from controlling terminal but still waits on agent PID
```

**Proposed**:
```markdown
- Creates new process group (`setsid()`) to isolate from terminal signals
- Runner still waits on agent PID and collects exit code
- This is NOT daemonization - parent remains `run-agent`, not init/PID 1
```

### Go Implementation

```go
func spawnAgent(args ...string) (*exec.Cmd, error) {
    cmd := exec.Command("agent-binary", args...)

    // Detach from controlling terminal
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid: true,  // New session, detached from terminal
    }

    // Redirect stdio to files (not /dev/null)
    cmd.Stdout = stdoutFile
    cmd.Stderr = stderrFile

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    // Parent can still wait
    go func() {
        err := cmd.Wait()
        log.Printf("Agent exited: %v", err)
    }()

    return cmd, nil
}
```

## Conclusion

**Not a bug, just unclear wording**. The specification is describing standard Unix process group management.

**Action**: Update subsystem-runner-orchestration.md:30 to clarify that "detach" means `setsid()` not daemonization.

**Status**: RESOLVED with specification clarification.
