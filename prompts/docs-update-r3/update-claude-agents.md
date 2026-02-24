# Docs Update R3: CLAUDE.md, AGENTS.md, THE_PROMPT_v5.md

You are a documentation update agent (Round 3). Facts take priority over all existing content.

## Files to update (overwrite each in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/CLAUDE.md`
2. `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md`
3. `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5.md`

## Facts sources (read ALL first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-prompts-workflow.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
```

## Verify against source

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Check current project structure for CLAUDE.md accuracy
ls internal/ | head -20
ls cmd/

# Check build tool (CLAUDE.md may reference wrong one)
cat Makefile 2>/dev/null | head -10 || ls *.mk 2>/dev/null
head -5 go.mod

# Check test framework
grep -rn "testing\.\|testify\|assert\." internal/ --include="*_test.go" | head -5

# Check CLAUDE.md references
cat CLAUDE.md | head -50

# Check AGENTS.md subsystem registry
ls internal/

# Check Instructions.md (may reference old structure)
head -30 Instructions.md 2>/dev/null
```

## Rules

- **Facts override docs**
- For CLAUDE.md: update Build Tool, Language, Test Framework table if stale; ensure Subsystem Registry lists actual subsystem paths under `internal/`
- For AGENTS.md: update agent type table if roles/tools changed; update subsystem registry; verify communication protocol matches current MESSAGE-BUS.md implementation
- For THE_PROMPT_v5.md: this is the master workflow prompt — from FACTS-prompts-workflow.md, verify stage descriptions, max parallelism limits, quality gates are still accurate; add any corrections found in Validation Round 2 section of facts
- Do not rewrite from scratch — targeted corrections only
- Do not change the workflow if the facts confirm it's correct

## Output

Overwrite each file in-place. Write summary to `output.md`.
