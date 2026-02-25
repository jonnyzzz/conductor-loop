# Validation Task: Runner, Storage & Environment Facts

You are a validation agent. Cross-check existing facts against source code and git history. Fix errors, add missing facts.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: runner, storage, <subsystem>]
<fact text>

```

## Step 1: Read existing facts
`cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`

## Step 2: Verify against actual source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Verify storage paths
cat internal/storage/atomic.go | head -50
cat internal/storage/layout.go 2>/dev/null || find internal/storage -name "*.go" | head -10

# Verify run ID format
grep -r "runID\|run_id\|RunID\|YYYYMMDD" internal/ --include="*.go" | grep -v test | head -20

# Verify Ralph loop parameters
cat internal/runner/ralph.go | head -80
grep -n "300\|idle\|stuck\|timeout\|5m\|15m" internal/runner/ralph.go | head -20

# Verify DONE marker semantics
grep -rn "DONE\|done" internal/runner/ --include="*.go" | grep -v "_test" | head -20

# Verify GC command
cat cmd/run-agent/gc.go | head -50

# Check run-info.yaml fields
cat internal/storage/run_info.go 2>/dev/null || grep -rn "RunInfo\|run-info" internal/ --include="*.go" | head -20

# Check config schema
cat internal/config/config.go | head -60

# Verify environment variables injected
grep -rn "JRUN_\|JRUN_RUN_FOLDER\|JRUN_RUNS_DIR\|JRUN_MESSAGE_BUS" internal/ --include="*.go" | grep -v test | head -30

# Check git log for all spec files
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-runner-orchestration.md docs/specifications/subsystem-storage-layout.md docs/specifications/subsystem-env-contract.md | head -20

# Read each spec at first and latest revision
for f in docs/specifications/subsystem-runner-orchestration.md docs/specifications/subsystem-storage-layout.md docs/specifications/subsystem-env-contract.md; do
  echo "=== $f ==="
  FIRST=$(git log --format="%H" -- "$f" | tail -1)
  FIRST_DATE=$(git log --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" -- "$f" | tail -1)
  echo "First revision: $FIRST_DATE"
  git show $FIRST:"$f" | head -40
  echo "--- CURRENT ---"
  cat "$f" | head -40
done
```

## Step 3: Check jonnyzzz-ai-coder git history for original spec dates
```bash
cd /Users/jonnyzzz/Work/jonnyzzz-ai-coder
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- swarm/docs/legacy/subsystem-runner-orchestration.md swarm/docs/legacy/subsystem-storage-layout.md 2>/dev/null | head -10
```

## Step 4: Check ALL revisions of the spec files in conductor-loop
```bash
cd /Users/jonnyzzz/Work/conductor-loop
# For runner spec - read every revision
git log --format="%H %ad" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-runner-orchestration.md
```

## Step 5: Write corrected output to FACTS-runner-storage.md
Add section: `## Validation Round 2 (gemini)` for new/corrected entries.
