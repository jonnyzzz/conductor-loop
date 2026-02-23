# Problem 2 Decision: Ralph Loop Termination Logic

## Context
The Ralph Loop (Root Orchestrator) faced a race condition: if the `DONE` marker exists but child processes are still running, the system's behavior was ambiguous ("wait and restart root to catch up"). This could lead to infinite loops, missed results, or orphaned processes.

## Selected Approach: Wait-Then-Restart with Timeout (Modified Approach B+C)

We will implement a robust "Wait, then Restart" strategy. This ensures that:
1.  **Completeness**: All work (including deep sub-tasks) is finished before the task is declared complete.
2.  **Aggregation**: The Root Agent is given one final chance (the restart) to aggregate results after children exit.
3.  **Safety**: Timeouts and loop termination conditions prevent infinite hangs.

### Algorithm

The Ralph Loop (supervisor) logic is updated as follows:

```pseudo
Loop:
  1. Start/Monitor Root Agent.
  2. If Root Agent exits:
     a. Check for DONE file.
     b. Check for Active Children (scan runs/ directory).
     
     Case 1: DONE exists AND No Active Children
       -> Task Complete. Exit Ralph Loop (Success).
     
     Case 2: DONE exists AND Active Children exist
       -> Log "Waiting for X children to finish..."
       -> Enter Wait Loop (Max Timeout: 300s [configurable]):
          - Sleep 1s
          - Re-scan Active Children
          - If All Children Finished -> Break Wait Loop
          - If Timeout -> Log Warning "Timeout waiting for children", Break Wait Loop
       -> Restart Root Agent (Continue to Step 1).
          *Rationale: Allow Root Agent to see final results and update DONE/Output.*
     
     Case 3: DONE missing
       -> Restart Root Agent (Standard restart).
          *Check restart limits/backoff.*
```

### Child Detection Implementation

To handle deep process trees and detached processes, we rely on **Process Groups (PGID)** managed by `run-agent`.

1.  **Discovery**: Scan `runs/` directory for `run-info.yaml` files.
2.  **Filtering**: Select runs where:
    - `task_id` matches current task.
    - `run_id` is NOT the current Root Agent's run_id.
3.  **Liveness Check**:
    - Read `pgid` from `run-info.yaml`.
    - Check signal 0 to the process group: `kill(-pgid, 0)`.
    - *Note*: Checking the specific `pid` is insufficient if the direct child exited but left grandchildren running in the same group. `run-agent job` manages this PGID.

### Timeout Policy

- **Default Timeout**: 300 seconds (5 minutes).
- **On Timeout**:
  - Log error to Message Bus: `TYPE=ERROR MSG="Timeout waiting for children: [run_ids...]"`
  - Proceed to Restart Root Agent.
  - *Constraint*: We do NOT aggressively kill orphans by default in MVP, to avoid data corruption, but the Root Agent might decide to clean up.

### Edge Case: Infinite Restart Prevention

If the Root Agent is "dumb" (sees DONE, immediately exits), the loop works correctly:
1.  Ralph sees DONE + Children -> Waits -> Restarts Root.
2.  Root runs -> Sees DONE -> Exits immediately.
3.  Ralph sees DONE + No Children -> Exits Success.

If the Root Agent keeps spawning *new* children despite DONE, it is buggy. The standard `max_restarts` limit (applied to the Ralph Loop) will eventually catch this and fail the task.

## Specification Updates

The `subsystem-runner-orchestration.md` file will be updated to reflect this precise logic, replacing the ambiguous "wait and restart root to catch up" text.
