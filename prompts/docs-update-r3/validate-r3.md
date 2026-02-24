# Docs Validation Round 3: Cross-Check All R3 Changes

You are a validation agent. Round 3 update agents changed README.md, decisions/, CLAUDE.md, AGENTS.md,
user docs, dev docs, and spec QUESTIONS files. Your job: verify the key changes and fix remaining issues.

## Step 1: Verify port, Go version, config path are consistent everywhere

```bash
cd /Users/jonnyzzz/Work/conductor-loop

echo "=== Port references ==="
grep -rn "14355\|8080" README.md DEVELOPMENT.md CLAUDE.md AGENTS.md docs/ | grep -v "facts\|swarm\|\.git\|binary.*8080\|drift" | head -30

echo "=== Go version ==="
grep -rn "go 1\.\|golang 1\.\|Go 1\." README.md DEVELOPMENT.md CLAUDE.md docs/ | grep -v "facts\|swarm" | head -20

echo "=== Config path ==="
grep -rn "\.conductor\b\|~/.conductor\|config.yaml\|XDG" README.md DEVELOPMENT.md CLAUDE.md docs/ | grep -v "facts\|swarm\|stale\|was\|old" | head -20
```

## Step 2: Verify key features appear in dev docs

```bash
cd /Users/jonnyzzz/Work/conductor-loop

for feature in "depends_on" "webhook" "prometheus\|metrics" "liveness\|reconcile" "audit" "max_concurrent_root" "diversif"; do
  count=$(grep -rn "$feature" docs/dev/ | grep -v "facts" | wc -l)
  echo "$feature: $count occurrences in docs/dev/"
done
```

## Step 3: Verify QUESTIONS files have answers

```bash
cd /Users/jonnyzzz/Work/conductor-loop
grep -rn "OPEN\|TBD\|UNKNOWN\|\[ \]" docs/specifications/*QUESTIONS* | head -20
```

## Step 4: Verify README accuracy spot-checks

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Check README quick-start example uses port 14355
grep -n "14355\|localhost\|http://" README.md | head -10

# Check README features match actual --help
./bin/run-agent --help 2>&1 | grep -E "Available Commands|Commands:" -A 20 | head -25
grep -n "run-agent\|conductor\b" README.md | grep -E "^\s*-\s" | head -20
```

## Step 5: Apply any remaining fixes directly

For each discrepancy found above, fix the file in-place.

## Step 6: Write validation report to output.md

Format:
- What was verified
- What was already correct
- What was fixed in this pass
- Any remaining issues

Create DONE file when complete.
