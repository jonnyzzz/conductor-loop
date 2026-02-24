# Docs Update R3: Answer Open Questions in Specs QUESTIONS files

You are a documentation update agent (Round 3). The specs were updated in Round 2.
Your job is to find every UNANSWERED question remaining in the QUESTIONS.md spec files and answer them using facts.

## Files to process

```bash
ls /Users/jonnyzzz/Work/conductor-loop/docs/specifications/*QUESTIONS*.md
```

Read each QUESTIONS file and identify all questions that are still marked as OPEN, TBD, UNKNOWN, or unanswered.

## Facts sources (read ALL first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md
```

## Also verify specific answers from code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Message bus: max message size, ordering guarantees
grep -n "max.*size\|MaxSize\|order\|seq\|sequence" internal/messagebus/messagebus.go | head -20

# Agent protocol: what happens on agent crash vs clean exit
grep -n "exitCode\|ExitCode\|exit.*code\|crashed\|panic" internal/runner/ -r --include="*.go" | head -20

# Storage: cleanup / GC policy
grep -n "gc\|GC\|cleanup\|Cleanup\|delete\|Delete\|ttl\|TTL" internal/ -r --include="*.go" | grep -v "_test\|vendor" | head -20

# Frontend-backend API: pagination
grep -n "page\|Page\|limit\|offset\|cursor" internal/api/ -r --include="*.go" | head -20

# Monitoring UI: what data refreshes via SSE vs poll
grep -n "poll\|Poll\|interval\|Interval\|SSE\|stream" frontend/src/ -r 2>/dev/null | head -20
```

## Rules

- **Facts override docs**
- For each QUESTIONS file: find open questions, look up the answer in facts or source code, update the question to include the answer
- Mark answered questions clearly: add `**Answer (2026-02-24):**` before the answer
- If a question truly cannot be answered from available facts/code, note `**Status: Open — insufficient facts to answer**`
- Update the main spec file (non-QUESTIONS) if the answer reveals the spec itself is wrong

## Output

Overwrite each QUESTIONS file in-place. Write summary to `output.md` listing which questions were answered.
