# Validation Task: Prompts, Workflow & Methodology Facts

You are a validation agent. Cross-check existing facts against ALL revisions of THE_PROMPT_v5.md and related workflow files.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-prompts-workflow.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: workflow, prompt, methodology, <subsystem>]
<fact text>

```

## Step 1: Read existing facts
`cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-prompts-workflow.md`

## Step 2: Read ALL revisions of workflow files

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Get full history of THE_PROMPT_v5.md (now at docs/workflow/)
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/workflow/THE_PROMPT_v5.md THE_PROMPT_v5.md

# Read EVERY revision to track evolution
for sha in $(git log --format="%H" -- docs/workflow/THE_PROMPT_v5.md THE_PROMPT_v5.md); do
  date=$(git log -1 --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" $sha)
  echo "=== $sha $date ==="
  git show $sha:docs/workflow/THE_PROMPT_v5.md 2>/dev/null || git show $sha:THE_PROMPT_v5.md 2>/dev/null | grep -E "^##|Stage [0-9]|Priority|Max [0-9]|MUST NOT|CRITICAL|Quality Gate" | head -20
done

# Read current full THE_PROMPT_v5.md
cat docs/workflow/THE_PROMPT_v5.md

# Read all specialized prompt variants
for f in docs/workflow/THE_PROMPT_v5_conductor.md docs/workflow/THE_PROMPT_v5_orchestrator.md docs/workflow/THE_PROMPT_v5_implementation.md docs/workflow/THE_PROMPT_v5_research.md docs/workflow/THE_PROMPT_v5_review.md docs/workflow/THE_PROMPT_v5_test.md docs/workflow/THE_PROMPT_v5_monitor.md docs/workflow/THE_PROMPT_v5_debug.md; do
  echo "=== $f ==="
  git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- "$f" | head -3
  cat "$f" | grep -E "^##|Stage|Priority|constraint|must|MUST|max|Max" | head -20
done

# Read AGENTS.md history
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- AGENTS.md | head -10
# Read each revision
for sha in $(git log --format="%H" -- AGENTS.md); do
  date=$(git log -1 --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" $sha)
  echo "=== AGENTS.md $date ==="
  git show $sha:AGENTS.md | head -40
done

# Read CLAUDE.md
cat CLAUDE.md

# Read Instructions.md
cat Instructions.md | head -60

# Check RLM orchestration docs
cat docs/user/rlm-orchestration.md | head -60
```

## Step 3: Check jonnyzzz-ai-coder for original prompt history
```bash
cd /Users/jonnyzzz/Work/jonnyzzz-ai-coder
# Find THE_PROMPT files
find . -name "THE_PROMPT*" 2>/dev/null | head -10
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- swarm/runs/run_20260220-132442-61579/prompt.md 2>/dev/null | head -5
```

## Step 4: Write corrected output
Focus on:
- Exact stage numbers and descriptions in the workflow
- Max parallelism limits (exact numbers)
- Quality gates (exact checks required)
- Agent role definitions (exact constraints)
- Commit format requirements

Add section: `## Validation Round 2 (gemini)` for new/corrected entries.
