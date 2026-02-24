# Validation Task: Issues, Questions, TODOs & Decisions Facts

You are a validation agent. Cross-check existing facts against ALL revisions of docs/dev/issues.md, docs/dev/questions.md, docs/dev/todos.md, and MESSAGE-BUS.md.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: issue, decision, todo, <subsystem>]
<fact text>

```

## Step 1: Read existing facts
`cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md`

## Step 2: Read ALL revisions of key files

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Get full history of docs/dev/issues.md
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/dev/issues.md

# Read EVERY revision of docs/dev/issues.md
for sha in $(git log --format="%H" -- docs/dev/issues.md); do
  date=$(git log -1 --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" $sha)
  echo "=== $sha $date ==="
  git show $sha:docs/dev/issues.md | grep -E "^###|ISSUE-[0-9]+|Status:|Severity:|Resolved:" | head -30
done

# Get full history of docs/dev/questions.md
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/dev/questions.md
for sha in $(git log --format="%H" -- docs/dev/questions.md); do
  date=$(git log -1 --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" $sha)
  echo "=== $sha $date ==="
  git show $sha:docs/dev/questions.md | grep -E "^##|^Q[0-9]|Decision:|Resolved:|Status:" | head -20
done

# Get full history of docs/dev/todos.md
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/dev/todos.md
# Read latest docs/dev/todos.md fully
cat docs/dev/todos.md

# Read MESSAGE-BUS.md - extract DECISION and FACT entries
grep -A3 "DECISION:\|FACT:\|ANSWER\|QUESTION:" MESSAGE-BUS.md | head -100

# Check feature requests doc
cat docs/dev/feature-requests-project-goal-manual-workflows.md | head -60

# Check SUGGESTED-TASKS.md
cat docs/SUGGESTED-TASKS.md | head -60

# Read swarm ISSUES.md
cat docs/swarm/ISSUES.md
```

## Step 3: Check jonnyzzz-ai-coder for related decision history
```bash
cd /Users/jonnyzzz/Work/jonnyzzz-ai-coder
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- swarm/ISSUES.md 2>/dev/null | head -10
# Read earliest revision of swarm ISSUES.md
FIRST=$(git log --format="%H" -- swarm/ISSUES.md 2>/dev/null | tail -1)
[ -n "$FIRST" ] && git show $FIRST:swarm/ISSUES.md 2>/dev/null | head -40
```

## Step 4: Write corrected output
The output must include:
- All 21+ ISSUES with severity, status, and resolution date
- All QUESTIONS with their answers
- Key TODOs (open ones with priority)
- Key MESSAGE-BUS decisions

Add section: `## Validation Round 2 (codex)` for new/corrected entries.
