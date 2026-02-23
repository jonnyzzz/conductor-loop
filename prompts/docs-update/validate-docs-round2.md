# Docs Validation Round 2: Cross-Check All Updated Docs Against Facts

You are a validation agent. Your job is to verify that ALL docs were correctly updated by Round 1 agents.

## Input: all facts files

```bash
ls /Users/jonnyzzz/Work/conductor-loop/docs/facts/
```

Read each FACTS-*.md file fully.

## Step 1: Sample-check each doc group

For each group, pick 2-3 specific facts that were supposed to be fixed and verify they appear correctly in the doc.

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Check 1: Go version in docs vs go.mod
go_ver=$(head -3 go.mod | grep "^go " | awk '{print $2}')
echo "go.mod version: $go_ver"
grep -rn "go 1\." docs/user/ docs/dev/ | grep -v "facts\|swarm" | head -20

# Check 2: Port 14355 vs 8080
echo "=== Port references ==="
grep -rn "14355\|8080\|default.*port\|port.*default" docs/user/ docs/dev/ docs/specifications/ | grep -v "facts\|swarm" | head -20

# Check 3: RunID format
echo "=== RunID format ==="
grep -rn "RunID\|run.id\|YYYYMMDD\|run_[0-9]" docs/ | grep -v "facts\|swarm" | head -20

# Check 4: Config path
echo "=== Config path ==="
grep -rn "\.conductor\|config.yaml\|config.hcl\|XDG" docs/user/ docs/dev/ | grep -v "facts\|swarm" | head -20

# Check 5: Gemini CLI flags
echo "=== Gemini flags ==="
grep -rn "screen.reader\|approval.mode\|stream.json\|yolo" docs/ | grep -v "facts\|swarm" | head -20

# Check 6: Claude flags
echo "=== Claude flags ==="
grep -rn "\-\-input.format\|permission.mode\|output.format" docs/ | grep -v "facts\|swarm" | head -20

# Check 7: xAI status
echo "=== xAI status ==="
grep -rn "xai\|XAI\|grok" docs/ | grep -v "facts\|swarm" | head -15

# Check 8: JRUN_ env vars
echo "=== JRUN env vars ==="
grep -rn "JRUN_\|MESSAGE_BUS\|RUNS_DIR" docs/ | grep -v "facts\|swarm" | head -20

# Check 9: Security fixes
echo "=== Security review ab5ea6e ==="
grep -rn "GHA\|sha.*pin\|linter.*pin\|inline.*token\|permission.*scope" docs/dev/ | head -15
```

## Step 2: Identify remaining issues

For each discrepancy found, note:
- Which doc file still has the stale value
- What fact says it should be
- The specific line/section to fix

## Step 3: Apply remaining fixes directly

For any remaining issues found in Step 2, fix them in the doc files directly.

## Step 4: Write validation report to output.md

List:
1. What was checked
2. What was correct (already updated by Round 1)
3. What was still wrong and was fixed in this round
4. Any remaining issues that could not be resolved
