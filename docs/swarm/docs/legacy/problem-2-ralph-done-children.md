# Problem: Ralph Loop DONE + Children Running → Race Condition

## Context
From subsystem-runner-orchestration.md:42:
> "If DONE exists but children are still running, wait and restart root to catch up"

## Problem
This specification has multiple critical ambiguities that make implementation impossible:

1. **HOW to wait?**
   - Poll every N seconds?
   - Block on `waitpid(-1, WNOHANG)`?
   - Use inotify on /proc?

2. **HOW to detect all children have exited?**
   - Recursive process tree traversal?
   - Check process table?
   - Track spawned PIDs in run-info.yaml?

3. **WHEN to restart?**
   - Immediately after last child exits?
   - After a timeout?
   - What if timeout expires before children exit?

4. **Will restarted root agent see DONE and immediately exit again?**
   - Does restarting root clear the DONE file?
   - Does root ignore DONE on restart?
   - Will this cause infinite restart loops?

## Impact
Ralph loop may:
- Spin-wait consuming CPU
- Miss child exits (children become orphans)
- Enter infinite restart loops
- Never properly complete tasks
- Break entire orchestration system

## Reviewer Consensus
**5 of 6 agents** flagged this as a critical blocker (Claude #1, #2, #3 + Gemini #2, #3)

## Your Task

Propose a concrete algorithm for Ralph loop behavior when DONE exists but children are running.

Consider the following approaches:

### Approach A: Don't Restart (Gemini #3 suggestion)
- When DONE exists and children running: just wait, don't restart
- Poll every 1s: check if all children exited
- Once all children exited: Ralph loop terminates
- No restart needed

### Approach B: Restart After Children Exit
- When DONE exists: wait for ALL children to exit
- Check every 500ms if children still exist
- Once all children exited: restart root agent
- Root agent sees DONE and processes final results

### Approach C: Timeout + Orphan Handling
- When DONE exists: wait up to 30 seconds for children
- If timeout expires: log warning, proceed anyway
- Orphaned children continue in background
- Root agent restarted to finalize

### Approach D: Message Bus Coordination
- Children post CHILD_DONE to message bus before exit
- Root agent tracks expected child count
- When all CHILD_DONE messages received: safe to restart
- No process table polling needed

For your chosen approach, specify:

1. **Exact algorithm**: Detailed pseudocode for wait + restart logic
2. **Child detection**: How to enumerate all running children (including deep subtrees)
3. **Timeout policy**: Maximum wait time, what happens on timeout
4. **Restart behavior**: Does root agent see DONE? Should DONE be cleared?
5. **Edge cases**: What if root crashes during wait? What if children spawn more children?
6. **Specification updates**: What exact changes to subsystem-runner-orchestration.md?

## Constraints
- Must work with detached processes (process group separation)
- Must handle deep process trees (child spawns child spawns child)
- Must not miss exits (no race conditions)
- Should not consume excessive CPU (no tight spin loops)
- Must terminate eventually (no infinite restart loops)

Provide a clear recommendation with implementation details.
