# Problem #2 Decision: Ralph Loop DONE + Children Running

## Two Agent Proposals

### Agent 1 (Gemini) - Approach A: Don't Restart
**Output:** `/Users/jonnyzzz/Work/jonnyzzz-ai-coder/runs/run_20260204-212649-67076/agent-stdout.txt`

**Key Points:**
- When DONE exists and children running: **DON'T restart root**
- Just wait up to 30 seconds for children to exit, then terminate
- Rationale: Root already finished, restarting would be redundant and could cause loops
- Child detection: Enumerate run-info.yaml files, check PIDs with `kill(pid, 0)`
- Timeout: 30 seconds, then log warning and proceed (orphan children)

**Algorithm:**
```
if DONE exists:
  wait for children (30s max)
  exit Ralph loop  // NO RESTART
```

### Agent 2 (Claude) - Approach B+C: Wait Then Restart
**Output:** `/Users/jonnyzzz/Work/jonnyzzz-ai-coder/runs/run_20260204-212649-67078/agent-stdout.txt`
**Decision Doc:** `/Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm/problem-2-decision.md`

**Key Points:**
- When DONE exists and children running: **wait then restart root**
- Wait up to 300 seconds for children to exit, then restart root once more
- Rationale: Gives root agent final chance to aggregate results
- Child detection: Check PGID with `kill(-pgid, 0)` to catch deep process trees
- Timeout: 300 seconds (5 minutes)

**Algorithm:**
```
if DONE exists AND children running:
  wait for children (300s max)
  restart root one final time  // RESTART
  root sees DONE and exits
  exit Ralph loop
```

## Key Difference

**Restart or Not?**
- **Gemini says NO**: Root already finished, don't restart
- **Claude says YES**: Root needs final opportunity to aggregate/finalize

## Your Task

Make the final decision and specify:

1. **Which approach?** Choose A (no restart) or B+C (restart once)

2. **Justification**: Why is your chosen approach better?
   - What's the downside of the rejected approach?
   - What use cases require/don't require the final restart?

3. **Child detection**: Which method?
   - `kill(pid, 0)` per process (Gemini)
   - `kill(-pgid, 0)` per process group (Claude)
   - OR both?

4. **Timeout value**: 30s (Gemini) or 300s (Claude)? Or different value?

5. **Exact algorithm**: Detailed pseudocode resolving all ambiguities

6. **Edge cases**:
   - What if root agent spawns NEW children after restart?
   - What if DONE exists but root never spawned children?
   - What if root crashes during the final restart?

7. **Specification updates**: Exact changes to subsystem-runner-orchestration.md:42

## Decision Criteria

Consider:
- **Correctness**: Does it handle all scenarios without data loss?
- **Simplicity**: Which is easier to implement and reason about?
- **Performance**: Which completes tasks faster?
- **Flexibility**: Which better supports different root agent patterns?
- **Robustness**: Which fails more gracefully?

Provide a clear, implementable decision that resolves the conflict.
