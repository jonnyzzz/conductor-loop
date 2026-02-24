# Docs Update R3: decisions/ folder + DEPENDENCY_ANALYSIS.md

You are a documentation update agent (Round 3). Facts take priority over all existing content.

## Files to update (overwrite each in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-2-FINAL-DECISION.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-4-DECISION.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-5-DECISION.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-7-DECISION.md`
6. `/Users/jonnyzzz/Work/conductor-loop/DEPENDENCY_ANALYSIS.md`

## Facts sources (read ALL first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
```

## Verify decisions against actual code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Problem 2: message bus lock strategy
grep -n "flock\|O_APPEND\|fsync\|lock\|Lock" internal/messagebus/messagebus.go | head -20
grep -n "fsync\|Fsync\|sync\b" internal/messagebus/ -r --include="*.go" | head -10

# Problem 4: storage layout decision
ls internal/storage/
cat internal/storage/storage.go 2>/dev/null | head -40

# Problem 5: run-id collision
grep -n "RunID\|PID\|sequence\|atomic\|counter" internal/runner/orchestrator.go | head -20

# Problem 7: process management / ralph loop
grep -n "pgid\|PGID\|process.*group\|Kill\|Signal\|syscall" internal/runner/ -r --include="*.go" | head -20

# DEPENDENCY_ANALYSIS: actual go.mod deps
cat go.mod
```

## Rules

- **Facts override docs** — these are decision records, so preserve historical decisions but add implementation status
- For each decision doc: add a short "Implementation Status" section at the top if missing, reflecting whether the decision was implemented as planned or deviated
- Check if the actual code matches the decision (e.g., fsync: is it always-on or configurable?)
- For DEPENDENCY_ANALYSIS.md: update any stale dependency versions or analysis from current go.mod
- These are historical docs — preserve the decision text, only add status updates
- Keep existing structure — add annotations, don't rewrite

## Output

Overwrite each file in-place. Write summary to `output.md`.
