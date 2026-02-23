# Docs Update: Dev Docs — Architecture, Subsystems, Storage Layout, Ralph Loop

You are a documentation update agent. Update docs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/storage-layout.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/ralph-loop.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
```

## Verify against source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Architecture: binary structure
ls cmd/
ls cmd/run-agent/ | head -20
ls cmd/conductor/ | head -10

# LOC counts
find . -name "*.go" | grep -v "_test\|vendor\|node_modules" | xargs wc -l | tail -1
find . -name "*_test.go" | grep -v "vendor\|node_modules" | xargs wc -l 2>/dev/null | tail -1

# Storage layout
ls internal/storage/ 2>/dev/null
cat internal/runner/orchestrator.go | head -80

# Ralph loop
cat internal/runner/ralph.go | head -80
grep -n "MaxRestarts\|max_restarts\|restarts\|DONE\|exitCode" internal/runner/ralph.go | head -20

# Run-info schema
cat internal/runner/runinfo.go 2>/dev/null | head -60
grep -rn "RunInfo\|run-info\|runinfo" internal/ --include="*.go" | grep "struct\|yaml:" | head -20

# RunID format
grep -n "RunID\|runID\|time.Now\|format.*run\|run.*format" internal/runner/orchestrator.go | head -20
```

## Rules

- **Facts override docs** — if a fact contradicts a doc, update the doc to match the fact
- Key fixes needed (from FACTS-architecture.md Round 2):
  - LOC counts: update to actual current numbers
  - `conductor` binary: fix the alias/wrapper claim vs. reality
  - fsync default: document the actual default (on/off)
  - xAI status: update deferral status per facts
  - Ralph loop stop condition: use exact wording from code
  - Root task queueing: add if missing
  - Diversification fallback: add if missing
  - Webhook notifications: add if implemented
- Key fixes for storage-layout.md (from FACTS-runner-storage.md Round 2):
  - RunID format: must be `YYYYMMDD-HHMMSSMMMM-PID` (4-digit fractional seconds)
  - Config precedence: config.yaml > config.hcl (update if wrong)
  - run-info.yaml is YAML not HCL
- For ralph-loop.md: verify max restarts default, DONE file mechanics, exit code handling

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
