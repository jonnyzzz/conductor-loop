# Ralph Loop (Root Agent Loop) Specification

This document specifies the Ralph Loop, the core restart management system for root agent processes in conductor-loop.

## Table of Contents

1. [Overview](#overview)
2. [Design Goals](#design-goals)
3. [Loop Algorithm](#loop-algorithm)
4. [Configuration](#configuration)
5. [DONE File Detection](#done-file-detection)
6. [Exit Conditions](#exit-conditions)
7. [Restart Logic](#restart-logic)
8. [Process Group Management](#process-group-management)
9. [Timeout Handling](#timeout-handling)
10. [Wait-Without-Restart Pattern](#wait-without-restart-pattern)
11. [Error Handling](#error-handling)
12. [Implementation Details](#implementation-details)

---

## Overview

The Ralph Loop is the restart manager for root agent processes. It automatically restarts failed agents up to a configurable limit, enabling resilient long-running tasks.

**Name Origin:** "Ralph" = **R**oot **A**gent **L**oop **P**rocess **H**andler

**Key Features:**
- Automatic restart on failure
- Configurable restart limits
- DONE file detection for completion signaling
- Process group cleanup
- Timeout enforcement
- Wait-without-restart signal support

**Package:** `internal/runner/`
**File:** `ralph.go`

---

## Design Goals

1. **Resilience:** Automatically recover from transient failures
2. **Simplicity:** Easy to understand and debug
3. **Control:** User can signal completion via DONE file
4. **Safety:** Prevent infinite loops with restart limits
5. **Cleanup:** Kill all child processes on exit

---

## Loop Algorithm

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────┐
│                      Ralph Loop                             │
│                                                             │
│  1. Initialize                                              │
│     restartCount = 0                                        │
│     maxRestarts = 100 (default)                             │
│                                                             │
│  2. Restart Loop (infinite)                                 │
│     ┌────────────────────────────────────────┐             │
│     │ loop:                                  │             │
│     │                                        │             │
│     │   a. Check context canceled → STOP     │             │
│     │                                        │             │
│     │   b. Check for DONE file (pre-run)     │             │
│     │      - If exists → wait children, STOP │             │
│     │                                        │             │
│     │   c. Check max restarts                │             │
│     │      - If restartCount >= max → FAIL   │             │
│     │                                        │             │
│     │   d. Run agent process (attempt N)     │             │
│     │      - Any exit code: logged, loop     │             │
│     │        continues regardless            │             │
│     │                                        │             │
│     │   e. Increment restartCount            │             │
│     │                                        │             │
│     │   f. Check for DONE file (post-run)    │             │
│     │      - If exists → wait children, STOP │             │
│     │                                        │             │
│     │   g. Delay before restart              │             │
│     │      - sleep(restartDelay)             │             │
│     │                                        │             │
│     └────────────────────────────────────────┘             │
│                                                             │
│  3. Exit                                                    │
│     - Return status (success or failure)                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Critical:** The loop stops **only** on: DONE file present, max restarts exceeded, or
context cancellation. A zero exit code does NOT stop the loop — the agent must create
the `DONE` file to signal task completion. See the [Exit Conditions](#exit-conditions)
section for the complete list.

### Pseudocode

```python
def ralph_loop():
    restart_count = 0
    max_restarts = 100
    restart_delay = 1  # second

    while True:
        # 1. Check context cancellation
        if context_canceled():
            return CANCELED

        # 2. Check DONE file BEFORE running agent
        if done_file_exists():
            wait_for_children()
            return SUCCESS

        # 3. Check max restarts
        if restart_count >= max_restarts:
            log("Max restarts exceeded, stopping")
            return FAILURE

        # 4. Run agent — any exit code is logged but loop always continues
        err = run_agent_process(attempt=restart_count)
        if err:
            log(f"WARNING: agent failed on restart #{restart_count}: {err}")
        # NOTE: exit code 0 (err=None) does NOT stop the loop here!
        restart_count += 1

        # 5. Check DONE file AFTER agent exits
        if done_file_exists():
            wait_for_children()
            return SUCCESS

        # 6. Restart delay (always — even after successful exit code)
        sleep(restart_delay)
```

---

## Configuration

### RalphLoop Structure

**File:** `internal/runner/ralph.go`

```go
type RalphLoop struct {
    runDir       string             // Run directory (contains DONE file)
    messagebus   *messagebus.MessageBus // Message bus for logging
    maxRestarts  int                // Maximum restart attempts
    waitTimeout  time.Duration      // Wait timeout
    pollInterval time.Duration      // DONE file check interval
    restartDelay time.Duration      // Delay between restarts
    projectID    string             // Project identifier
    taskID       string             // Task identifier
    runRoot      RootRunner         // Agent execution callback
}
```

### Default Configuration

**Constants:**

Reference: `internal/runner/ralph.go:17-22`

```go
const (
    defaultRalphWaitTimeout  = 300 * time.Second  // 5 minutes
    defaultRalphPollInterval = time.Second        // 1 second
    defaultRalphMaxRestarts  = 100                // 100 attempts
    defaultRalphRestartDelay = time.Second        // 1 second
)
```

### Configuration Options

```go
type RalphOption func(*RalphLoop)

// Override maximum restarts
func WithMaxRestarts(max int) RalphOption

// Override child wait timeout
func WithWaitTimeout(timeout time.Duration) RalphOption

// Override polling interval for DONE file
func WithPollInterval(interval time.Duration) RalphOption

// Override restart delay
func WithRestartDelay(delay time.Duration) RalphOption

// Set project/task identifiers
func WithProjectTask(projectID, taskID string) RalphOption

// Set root runner callback
func WithRootRunner(run RootRunner) RalphOption
```

### Example Configuration

```go
rl, err := NewRalphLoop(
    runDir,
    messagebus,
    WithMaxRestarts(50),
    WithWaitTimeout(10*time.Minute),
    WithRestartDelay(5*time.Second),
    WithProjectTask("my-project", "task-001"),
    WithRootRunner(func(ctx context.Context, attempt int) error {
        // Run agent process
        return runAgent(ctx, attempt)
    }),
)
```

---

## DONE File Detection

### Purpose

The DONE file allows agents to signal task completion from within their execution environment, bypassing the need for specific exit codes.

### Location

**Path:** `{task_dir}/DONE`

Example: `/path/to/run-agent/my-project/task-001/DONE`

### Creation

**From Agent:**
```bash
#!/bin/bash
# Inside agent script

# Do work...
echo "Implementing feature..."

# Signal completion
echo "Task completed successfully" > $TASK_FOLDER/DONE
exit 0
```

**From CLI:**
```bash
touch /path/to/task-folder/DONE
```

### Detection

**Polling:**
- Ralph loop checks for DONE file after each agent execution
- Uses `os.Stat()` to check file existence
- No read required (presence is enough)

**Implementation:**
```go
func (rl *RalphLoop) doneExists() (bool, error) {
    path := filepath.Join(rl.runDir, "DONE")  // rl.runDir is the task directory
    info, err := os.Stat(path)
    if err == nil {
        if info.IsDir() {
            return false, errors.New("DONE is a directory")
        }
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, errors.Wrap(err, "stat DONE")
}
```

### Behavior

**If DONE file exists:**
1. Ralph loop waits for any active child runs to complete (up to `waitTimeout`)
2. No restart attempted
3. Loop returns nil (success)
4. Message logged: "task completed (DONE marker present, no active children)"
5. Completion fact propagated to `PROJECT-MESSAGE-BUS.md` (best-effort)

**If DONE file does not exist:**
1. Ralph loop continues normal exit code handling
2. May restart if exit code is non-zero

---

## Exit Conditions

### Stop Conditions (No Restart)

Ralph loop stops and does NOT restart in these cases:

> **Important:** Exit code 0 by itself is NOT a stop condition. The loop always
> continues to the next attempt unless DONE is present, restarts are exhausted,
> or the context is canceled.

#### 1. DONE File Detected

Checked **before** the first run and **after** every run. Path: `<task_dir>/DONE`.

```go
done, err := rl.doneExists()  // filepath.Join(rl.runDir, "DONE")
if done {
    return rl.handleDone(ctx)  // wait for children, return nil
}
```

**Behavior:**
- Waits for active child runs (up to `waitTimeout` = 300s)
- No restart
- Return nil (success)
- Completion fact propagated to `PROJECT-MESSAGE-BUS.md` (best-effort)

#### 2. Max Restarts Exceeded

```go
if restarts >= rl.maxRestarts {
    // posts ERROR to task message bus
    return errors.New("max restarts exceeded")
}
```

**Behavior:**
- Posts an ERROR message to the task message bus
- No restart
- Return error

#### 3. Context Canceled

```go
if err := ctx.Err(); err != nil {
    return err
}
```

**Behavior:**
- No restart
- Return context.Canceled or context.DeadlineExceeded

---

## Restart Logic

### Restart Conditions

Ralph loop restarts the agent in these cases:

1. **Any Exit Code (zero or non-zero):** The loop always restarts unless a stop condition is met
2. **Within Restart Limit:** restartCount < maxRestarts
3. **No DONE File:** DONE file does not exist (checked pre- and post-run)

> The loop does NOT stop on exit code 0. The agent must write `DONE` to stop the loop.

### Restart Flow

```
1. Log restart decision
   "Agent failed with exit code 1, restarting (attempt 2/100)"

2. Delay before restart
   sleep(restartDelay)  // Default: 1 second

3. Increment restart counter
   restartCount++

4. Continue loop
   goto step 1 (run agent again)
```

### Restart Delay

**Purpose:** Prevent tight restart loops that consume CPU

**Default:** 1 second

**Configurable:** Via `WithRestartDelay(duration)`

**Example:**
```go
// Wait 5 seconds between restarts
WithRestartDelay(5 * time.Second)
```

### Restart Counter

**Purpose:** Track number of restart attempts

**Logging:**
```
Attempt 1: Agent failed (exit 1), restarting...
Attempt 2: Agent failed (exit 1), restarting...
Attempt 3: Agent failed (exit 1), restarting...
...
Attempt 100: Max restarts exceeded, stopping
```

---

## Process Group Management

### Purpose

Process groups allow killing the entire process tree (parent + all children) together.

### Unix/Linux/macOS

**Setting PGID:**
```go
cmd := exec.CommandContext(ctx, agentCLI, args...)
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,  // Create new process group
    Pgid:    0,     // Use PID as PGID
}
```

**Killing Process Group:**
```go
// Kill entire process group
// Negative PID kills process group
syscall.Kill(-pgid, syscall.SIGTERM)

// Wait for graceful shutdown
time.Sleep(5 * time.Second)

// Force kill if still alive
syscall.Kill(-pgid, syscall.SIGKILL)
```

### Windows

**Limitation:** No native process group support

**Workaround:**
- Use job objects (complex)
- Track children manually (unreliable)
- Recommend WSL2 for Windows users

### Cleanup Flow

```
1. Agent process exits (or times out)

2. Ralph loop checks for children
   - Use PGID to find all processes in group

3. Send SIGTERM to process group
   - Graceful shutdown signal

4. Wait 5 seconds

5. Send SIGKILL to process group (if needed)
   - Force kill any remaining processes

6. Verify all processes terminated
   - Check /proc or ps output
```

---

## Timeout Handling

### Wait Timeout

**Purpose:** Prevent indefinite waiting for agent completion

**Default:** 5 minutes (300 seconds)

**Configuration:**
```go
WithWaitTimeout(10 * time.Minute)
```

### Timeout Flow

```
1. Start agent process

2. Create timeout timer
   timer := time.NewTimer(waitTimeout)

3. Wait for completion or timeout
   select {
   case <-processExitChan:
       // Process completed normally
   case <-timer.C:
       // Timeout exceeded
       killProcessGroup(pgid)
       return errors.New("agent timeout")
   }
```

### Timeout Behavior

**If timeout occurs:**
1. Send SIGTERM to process group
2. Wait 5 seconds for graceful shutdown
3. Send SIGKILL to process group (if needed)
4. Treat as restart condition (non-zero exit)
5. May restart if within limit

**Logging:**
```
Agent timeout after 5m0s, killing process group
Process group killed, treating as failure
Restarting agent (attempt 2/100)
```

---

## Wait-Without-Restart Pattern

### Purpose

Allow agents to signal that they are waiting for external events (user input, webhook, etc.) and should NOT be restarted on exit.

### Signal Mechanism

**Option 1: Exit Code Convention**
```bash
# Inside agent
exit 42  # Special exit code for wait-without-restart
```

**Option 2: Message Bus Signal**
```go
messagebus.AppendMessage(&Message{
    Type: "wait_without_restart",
    Body: "Waiting for user input",
})
```

**Option 3: File Marker**
```bash
touch $TASK_FOLDER/WAIT_WITHOUT_RESTART
```

### Ralph Loop Behavior

```go
if waitWithoutRestartSignaled() {
    log("Wait-without-restart signal detected")
    return nil  // Stop loop without restart
}
```

**Result:**
- Ralph loop stops
- Run status: `stopped` (not `failed`)
- No restart attempted

---

## Error Handling

### Error Types

1. **Transient Errors:** Temporary issues (network, rate limit)
   - **Action:** Restart agent
   - **Example:** Network timeout, API rate limit

2. **Fatal Errors:** Permanent issues (configuration, not found)
   - **Action:** Stop loop, return error
   - **Example:** Executable not found, invalid API key

3. **Context Errors:** Cancellation, timeout
   - **Action:** Stop loop, return context error
   - **Example:** User cancellation, deadline exceeded

### Error Classification

```go
func isFatalError(err error) bool {
    if err == nil {
        return false
    }

    // Executable not found
    if errors.Is(err, exec.ErrNotFound) {
        return true
    }

    // Permission denied
    if os.IsPermission(err) {
        return true
    }

    // Configuration errors
    if errors.Is(err, ErrInvalidConfig) {
        return true
    }

    // Transient errors
    return false
}
```

### Error Logging

```go
if err := rl.runRoot(ctx, restartCount); err != nil {
    if isFatalError(err) {
        log("Fatal error: %v", err)
        return err
    }
    log("Transient error: %v", err)
    // Continue to restart logic
}
```

---

## Implementation Details

### RalphLoop.Run Method

**File:** `internal/runner/ralph.go`

**Signature:**
```go
func (rl *RalphLoop) Run(ctx context.Context) error
```

**Implementation Outline:**

```go
// Actual implementation (internal/runner/ralph.go)
func (rl *RalphLoop) Run(ctx context.Context) error {
    restarts := 0
    for {
        // 1. Check context cancellation
        if err := ctx.Err(); err != nil {
            return err
        }

        // 2. Check DONE file BEFORE running
        done, err := rl.doneExists()
        if err != nil {
            return err
        }
        if done {
            return rl.handleDone(ctx)  // wait for children
        }

        // 3. Check max restarts
        if restarts >= rl.maxRestarts {
            // posts ERROR to message bus
            return errors.New("max restarts exceeded")
        }

        // 4. Run agent — any exit code, loop always continues
        if err := rl.runRoot(ctx, restarts); err != nil {
            // logs WARNING but does NOT stop the loop
        }
        restarts++

        // 5. Check DONE file AFTER running
        done, err = rl.doneExists()
        if err != nil {
            return err
        }
        if done {
            return rl.handleDone(ctx)  // wait for children
        }

        // 6. Sleep before restart (even if exit code was 0)
        if err := sleepWithContext(ctx, rl.restartDelay); err != nil {
            return err
        }
        // loop back to step 1
    }
}
```

### RootRunner Callback

**Type:**
```go
type RootRunner func(ctx context.Context, attempt int) error
```

**Purpose:** Execute one agent run attempt

**Parameters:**
- `ctx`: Context for cancellation
- `attempt`: Restart attempt number (0-indexed)

**Returns:**
- `nil`: Success (exit code 0)
- `error`: Failure or fatal error

**Example Implementation:**
```go
rootRunner := func(ctx context.Context, attempt int) error {
    log("Starting agent (attempt %d)", attempt)

    // Spawn agent process
    cmd := exec.CommandContext(ctx, "agent-cli", args...)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

    // Start process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("start agent: %w", err)
    }

    // Wait for completion
    if err := cmd.Wait(); err != nil {
        return fmt.Errorf("wait agent: %w", err)
    }

    return nil
}
```

---

## Best Practices

### 1. Set Reasonable Restart Limits

**Good:**
```go
WithMaxRestarts(10)  // For most tasks
WithMaxRestarts(50)  // For flaky environments
```

**Bad:**
```go
WithMaxRestarts(1000)  // Too many, wastes resources
```

### 2. Use DONE Files for Explicit Completion

**Good:**
```bash
# Inside agent
if task_complete; then
    echo "Complete" > $TASK_FOLDER/DONE
    exit 0
fi
```

### 3. Add Restart Delays for Transient Errors

**Good:**
```go
WithRestartDelay(5 * time.Second)  // Give system time to recover
```

**Bad:**
```go
WithRestartDelay(0)  // Tight loop, wastes CPU
```

### 4. Log Restart Attempts

**Good:**
```
[2026-02-05 10:00:00] Attempt 1: Agent failed (exit 1), restarting...
[2026-02-05 10:00:05] Attempt 2: Agent failed (exit 1), restarting...
[2026-02-05 10:00:10] Attempt 3: Agent succeeded
```

### 5. Clean Up Process Groups

**Good:**
```go
// Always clean up, even on error
defer func() {
    if pgid > 0 {
        killProcessGroup(pgid)
    }
}()
```

---

## Testing

### Unit Tests

**File:** `internal/runner/ralph_test.go`

**Test Cases:**
- Run with immediate success (exit 0)
- Run with eventual success (after N restarts)
- Run with max restarts exceeded
- Run with DONE file detection
- Run with context cancellation
- Run with fatal error
- Run with restart delay

### Integration Tests

- Test with real agent execution
- Test process group cleanup
- Test timeout handling

---

## References

- [Architecture Overview](architecture.md)
- [Runner Orchestration](subsystems.md#6-runner-orchestration)
- [Process Lifecycle](architecture.md#process-lifecycle)

---

**Last Updated:** 2026-02-23 (facts-validated)
**Version:** 1.1.0
