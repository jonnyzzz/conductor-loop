# Problem #2 FINAL DECISION: Ralph Loop DONE + Children Running

## Implementation Status (R3 - 2026-02-24)

The historical decision in this file is implemented in current code:

- `internal/runner/ralph.go` checks `DONE` before and after root attempts.
- When `DONE` exists and children are active, `handleDone()` waits through `WaitForChildren()` in `internal/runner/wait.go`.
- Root is not restarted after `DONE`.
- Wait defaults match the decision intent (300s timeout, 1s poll) and are configurable through Ralph options.
- Timeout path emits a warning and then completes the task.

Note: newer issue tracking uses different numbering; this file keeps the original "Problem #2" label.

## Decision: Modified Approach A - Wait Without Restart

When DONE exists and children are running, the Ralph loop should:
1. **Wait for all children to exit** (with timeout)
2. **NOT restart the root agent**
3. **Declare task complete** once all children exit

## Rationale

### 1. Semantic Correctness
**DONE means "root agent has completed its work"**. If root writes DONE, it has declared completion. Restarting it would be logically inconsistent - the root agent would see DONE and immediately exit again, serving no purpose.

### 2. No Loss of Functionality
If root needs to aggregate results from children, it should:
- **Wait for children before writing DONE** (correct pattern)
- Monitor message bus for CHILD_DONE messages
- Aggregate results
- Write output.md
- Write DONE (only when truly finished)

The spec should document this as the **correct agent pattern**, not work around incorrect patterns with restarts.

### 3. Avoids Race Conditions
The "restart after children exit" approach creates multiple race conditions:
- Root might spawn NEW children after restart
- Root might write DONE again (idempotent but wasteful)
- Root might crash during restart, requiring complex recovery logic
- Infinite loop if root is buggy (writes DONE, exits immediately, repeat)

### 4. Simpler Implementation
No need to handle:
- Restart logic after wait completes
- Detection of "final restart" vs normal restart
- Root agent's behavior when restarted after DONE exists
- Edge cases where restart itself fails

## Detailed Algorithm

```go
// Ralph Loop - run-agent task subcommand
func runTask(projectID, taskID string, config Config) error {
    restartCount := 0

    for {
        // ═══════════════════════════════════════════════════════
        // STEP 1: Check for completion BEFORE starting root
        // ═══════════════════════════════════════════════════════

        if fileExists(taskDir + "/DONE") {
            log.Info("DONE marker exists, checking for active children...")

            // Find all running children
            children, err := findActiveChildren(taskID)
            if err != nil {
                log.Error("Failed to enumerate children: %v", err)
                // Proceed with caution - assume no children
                children = []ChildProcess{}
            }

            if len(children) == 0 {
                log.Info("No active children, task complete")
                postMessageBus(taskID, Message{
                    Type: "INFO",
                    Body: "Task completed successfully (DONE marker present, all children exited)",
                })
                return nil
            }

            // ═══════════════════════════════════════════════════════
            // STEP 2: Wait for children to exit (with timeout)
            // ═══════════════════════════════════════════════════════

            log.Info("DONE marker exists but %d children still running: %v",
                len(children), childRunIDs(children))

            postMessageBus(taskID, Message{
                Type: "INFO",
                Body: fmt.Sprintf("Waiting for %d children to complete: %v",
                    len(children), childRunIDs(children)),
            })

            err = waitForChildrenToExit(children, config.ChildWaitTimeout)

            if err != nil {
                // Timeout or other error
                log.Warn("Timeout waiting for children: %v", err)
                postMessageBus(taskID, Message{
                    Type: "WARNING",
                    Body: fmt.Sprintf("Timeout waiting for children after %v: %v",
                        config.ChildWaitTimeout, childRunIDs(children)),
                    Metadata: map[string]interface{}{
                        "orphaned_runs": childRunIDs(children),
                        "timeout_seconds": config.ChildWaitTimeout.Seconds(),
                    },
                })
                // Proceed to completion anyway (orphan children)
            }

            log.Info("All children exited (or timeout), task complete")
            postMessageBus(taskID, Message{
                Type: "INFO",
                Body: "Task completed (all children finished or timed out)",
            })
            return nil
        }

        // ═══════════════════════════════════════════════════════
        // STEP 3: DONE not present, start/restart root agent
        // ═══════════════════════════════════════════════════════

        // Check restart limits
        if restartCount >= config.MaxRestarts {
            log.Error("Max restarts (%d) exceeded", config.MaxRestarts)
            postMessageBus(taskID, Message{
                Type: "ERROR",
                Body: fmt.Sprintf("Task failed: max restarts (%d) exceeded", config.MaxRestarts),
            })
            return fmt.Errorf("max restarts exceeded")
        }

        // Generate run ID and start root agent
        runID := generateRunID()
        log.Info("Starting root agent (restart #%d): %s", restartCount, runID)

        rootAgent := startRootAgent(projectID, taskID, runID, restartCount)

        // Wait for root agent to exit
        exitCode := rootAgent.Wait()
        log.Info("Root agent %s exited with code %d", runID, exitCode)

        // Record exit in run-info
        updateRunInfo(runID, exitCode)

        // Post STOP event to message bus
        postMessageBus(taskID, Message{
            Type: "RUN_STOP",
            Metadata: map[string]interface{}{
                "run_id": runID,
                "exit_code": exitCode,
                "restart_count": restartCount,
            },
        })

        restartCount++

        // Brief pause before checking DONE again (avoid tight loop)
        time.Sleep(1 * time.Second)
    }
}

// waitForChildrenToExit polls until all children exit or timeout
func waitForChildrenToExit(children []ChildProcess, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    pollInterval := 1 * time.Second

    for {
        // Re-check active children (some may have exited)
        stillRunning := []ChildProcess{}
        for _, child := range children {
            if isProcessAlive(child.PGID) {
                stillRunning = append(stillRunning, child)
            }
        }

        if len(stillRunning) == 0 {
            log.Info("All children have exited")
            return nil
        }

        // Check timeout
        if time.Now().After(deadline) {
            return fmt.Errorf("timeout: %d children still running after %v: %v",
                len(stillRunning), timeout, childRunIDs(stillRunning))
        }

        // Log progress
        log.Debug("Waiting for %d children: %v",
            len(stillRunning), childRunIDs(stillRunning))

        time.Sleep(pollInterval)
    }
}

// findActiveChildren discovers all running child processes for a task
func findActiveChildren(taskID string) ([]ChildProcess, error) {
    taskRunsDir := filepath.Join(config.ProjectsRoot, projectID, "tasks", taskID, "runs")

    // List all run directories
    runDirs, err := os.ReadDir(taskRunsDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read runs directory: %w", err)
    }

    var children []ChildProcess

    for _, entry := range runDirs {
        if !entry.IsDir() {
            continue
        }

        runInfoPath := filepath.Join(taskRunsDir, entry.Name(), "run-info.yaml")
        runInfo, err := readRunInfo(runInfoPath)
        if err != nil {
            log.Warn("Failed to read run-info for %s: %v", entry.Name(), err)
            continue
        }

        // Skip completed runs (have end_time)
        if runInfo.EndTime != "" {
            continue
        }

        // Skip root agent runs (parent_run_id is empty)
        if runInfo.ParentRunID == "" {
            continue
        }

        // Check if process is still alive
        if isProcessAlive(runInfo.PGID) {
            children = append(children, ChildProcess{
                RunID: runInfo.RunID,
                PID:   runInfo.PID,
                PGID:  runInfo.PGID,
            })
        } else {
            // Process died but run-info not updated (crash?)
            log.Warn("Process %d (PGID %d) for run %s is dead but run-info has no end_time",
                runInfo.PID, runInfo.PGID, runInfo.RunID)
            // Update run-info to mark as crashed
            updateRunInfoCrashed(runInfoPath, runInfo)
        }
    }

    return children, nil
}

// isProcessAlive checks if a process group is still alive
func isProcessAlive(pgid int) bool {
    // Send signal 0 to process group (doesn't send signal, just checks existence)
    // Negative PID means process group
    err := syscall.Kill(-pgid, 0)
    if err == nil {
        return true // Process group exists
    }
    if errors.Is(err, syscall.ESRCH) {
        return false // No such process group
    }
    // EPERM means process exists but we can't signal it (still alive)
    return errors.Is(err, syscall.EPERM)
}
```

## Configuration Parameters

Add to `config.hcl`:

```hcl
task {
  # Maximum number of root agent restarts before giving up
  max_restarts = 50

  # Maximum time to wait for children when DONE exists (default: 5 minutes)
  child_wait_timeout = "300s"

  # Polling interval when waiting for children (default: 1 second)
  child_poll_interval = "1s"
}
```

## Child Detection Strategy

### Method: Process Group ID (PGID) Check

**Why PGID instead of PID?**
- Handles deep process trees (child spawns child spawns child)
- Works with detached processes (setsid creates new PGID)
- Single check covers entire subtree
- `run-agent job` already sets up process groups

**Implementation:**
1. Enumerate all `run-info.yaml` files in `tasks/{task_id}/runs/`
2. Filter for runs with:
   - Empty `end_time` (not completed)
   - Non-empty `parent_run_id` (not root agent)
   - Matching `task_id`
3. For each run, check if PGID is alive: `kill(-pgid, 0)`
4. If PGID alive → count as active child
5. If PGID dead but `end_time` empty → mark as crashed, update run-info

**Handles edge cases:**
- Orphaned processes (detached with setsid)
- Deep subtrees (grandchildren, great-grandchildren)
- Crashed processes (PGID dead but run-info not updated)
- Zombie processes (kill returns ESRCH)

## Timeout Policy

**Default: 300 seconds (5 minutes)**

**On timeout:**
1. Log WARNING to message bus with orphaned run IDs
2. Proceed to task completion anyway
3. Do NOT kill orphaned children (let them finish naturally)
4. Orphaned children will eventually exit, updating their run-info

**Rationale for NOT killing orphans:**
- Child might be writing critical results
- Abrupt termination could corrupt data
- Child might have its own children (cascade kill complex)
- Better to complete with warning than risk data loss

**User can:**
- Manually stop orphaned runs with `run-agent stop {run_id}`
- Configure shorter timeout in config.hcl
- Inspect orphaned runs in UI

## Edge Cases Handled

### 1. Root Crashes During Wait
**Scenario:** Root exits, Ralph sees DONE, starts waiting, supervisor crashes

**Handling:**
- On supervisor restart, Ralph loop resumes
- Checks DONE (still exists)
- Checks children (still running or exited)
- Resumes wait or completes

**No data loss**, idempotent behavior.

### 2. Root Never Spawned Children
**Scenario:** DONE exists, findActiveChildren() returns empty list

**Handling:**
- Ralph sees DONE + no children
- Task completes immediately
- No wait, no restart

**Optimal behavior** for simple tasks.

### 3. Children Spawn More Children
**Scenario:** Child A spawns grandchild B while Ralph is waiting

**Handling:**
- Grandchild B has parent_run_id = A's run_id
- findActiveChildren() enumerates ALL runs (including B)
- PGID check on B's PGID → still alive
- Ralph continues waiting

**Deep trees handled correctly** via recursive enumeration.

### 4. Root Writes DONE Multiple Times
**Scenario:** Root writes DONE, exits, Ralph restarts (due to bug), root writes DONE again

**Handling:**
- File overwrite is idempotent
- Ralph loop behavior unchanged

**No harm**, idempotent DONE marker.

### 5. Child Exits But run-info Not Updated
**Scenario:** Child crashes, process exits, but run-info.yaml not written (disk full?)

**Handling:**
- findActiveChildren() checks PGID with kill()
- If PGID dead but end_time empty → log warning
- Update run-info to mark as crashed (best effort)
- Continue (child not counted as active)

**Graceful degradation**, prefers completion over perfection.

### 6. Root Agent Pattern Mismatch
**Scenario:** Root spawns children, writes DONE immediately (fire-and-forget)

**Handling:**
- Ralph sees DONE + children running
- Ralph waits for children to exit
- Task completes when children finish

**This is CORRECT behavior** - root declared itself done, children finishing work.

**If root INTENDED to aggregate:** Root should NOT write DONE until children finish. The agent prompt should be fixed, not the orchestration.

## Specification Updates

**File:** `subsystem-runner-orchestration.md`

**Line 42-50 (Current):**
```markdown
- **Ralph Loop Termination Logic**:
  - If **DONE exists** AND **No Active Children**: Task is complete. Exit successfully.
  - If **DONE exists** AND **Active Children** are running:
    - Log "Waiting for children..."
    - Wait for all children's process groups (PGID) to exit, polling every 1s.
    - Timeout after 300s (configurable). If timeout, log warning and proceed.
    - **Restart Root Agent** one final time to allow result aggregation.
  - If **DONE missing**: Restart Root Agent (subject to max_restarts).
```

**Replace with:**
```markdown
- **Ralph Loop Termination Logic**:
  - Check for DONE marker BEFORE starting/restarting root agent.
  - If **DONE exists**:
    - Enumerate all active children (runs with empty end_time and non-empty parent_run_id).
    - If **No Active Children**: Task is complete. Exit successfully.
    - If **Active Children Running**:
      - Log "Waiting for N children to complete: [run_ids...]"
      - Post INFO message to message bus with child run IDs.
      - Poll children every 1s using `kill(-pgid, 0)` checks.
      - Timeout after 300s (configurable via config.child_wait_timeout).
      - On timeout: Log WARNING to message bus, proceed to completion (orphan children).
      - Once all children exit: Task is complete. Exit successfully.
      - **Do NOT restart root agent** (root already declared completion via DONE).
  - If **DONE missing**: Start/restart Root Agent (subject to max_restarts).
  - Between restart attempts, pause 1s to avoid tight loops.
```

**Line 72 (Add to config items):**
```markdown
- Configurable items:
  - max restarts / time budget for Ralph loop
  - child_wait_timeout (default 300s, used when DONE exists but children running)
  - child_poll_interval (default 1s, used when waiting for children)
  - idle/stuck thresholds
  ...
```

**Line 108 (Update Root Agent Prompt Contract):**
```markdown
## Root Orchestrator Prompt Contract
The root agent prompt must include:
- Read TASK_STATE.md and TASK-MESSAGE-BUS.md on start.
- Use message bus only for communication.
- Regularly poll message bus for new messages.
- Write facts as FACT-*.md (YAML front matter required).
- Update TASK_STATE.md with a short free-text status.
- **IMPORTANT**: If delegating work to children, wait for children to complete and post results BEFORE writing DONE.
- Monitor message bus for CHILD_DONE or result messages from children if aggregation is needed.
- Write final results to output.md in the run folder.
- Create DONE file ONLY when the task is complete (all work finished, including children if applicable).
- Post an INFO/OBSERVATION message to the bus when writing DONE.
- Delegate sub-tasks by starting sub agents via run-agent.
```

**New Section (Add after line 140):**

```markdown
## Agent Design Patterns

### Pattern A: Parallel Delegation with Aggregation (Recommended)
```
Root: Spawn N children for parallel subtasks
Root: Monitor message bus for CHILD_DONE messages
Root: Wait for all children to report completion
Root: Aggregate results from children's outputs
Root: Write final output.md
Root: Write DONE
Root: Exit
```

**Key principle:** Root writes DONE only after children complete and results are aggregated.

### Pattern B: Fire-and-Forget Delegation
```
Root: Spawn N children for independent subtasks
Root: Write DONE immediately (root's work is done)
Root: Exit (children continue independently)
```

**Key principle:** Root does not need children's results. Ralph loop will wait for children to exit before completing task.

**Anti-pattern (DO NOT USE):**
```
Root: Spawn children
Root: Write DONE immediately
Root: Expect to be restarted to aggregate results
```
This is incorrect - root should wait for children BEFORE writing DONE if aggregation is needed.
```

## Comparison with Rejected Approach

### Why NOT "Restart After Children Exit"?

**Claude's approach (rejected):**
1. DONE exists + children running → wait for children
2. Once children exit → restart root agent
3. Root sees DONE → exits immediately
4. Task complete

**Problems:**
1. **Purposeless restart:** Root exits immediately upon seeing DONE (it already declared completion)
2. **No aggregation opportunity:** Root doesn't know WHY it was restarted or what to aggregate
3. **Race conditions:** Root might spawn new children after restart, causing infinite loop
4. **Restart complexity:** Need to handle restart failures, root crashes, etc.
5. **Violates DONE semantics:** DONE means "I'm finished," not "restart me later"

**The restart serves no purpose because:**
- Root has no memory of what children were spawned (separate process)
- Root has no way to know "this is the final restart after children finished"
- Root cannot aggregate results it didn't wait for
- Root will see DONE and exit, making the restart a no-op

**Correct solution:** If root needs to aggregate, it should WAIT for children before writing DONE (Pattern A).

## Summary

**Decision:** When DONE exists and children are running:
1. ✅ Wait for children to exit (300s timeout)
2. ❌ Do NOT restart root agent
3. ✅ Complete task after children exit

**Rationale:**
- DONE means root finished, restarting is nonsensical
- Root should wait for children BEFORE writing DONE if aggregation needed
- Simpler implementation, fewer edge cases
- Avoids restart loops and race conditions
- Correct agent patterns documented explicitly

**Implementation:** ~150 lines of Go code, zero external dependencies, handles all edge cases.
