# Research Task: Issues, Questions, TODOs & Decisions Facts

You are a research agent. Extract key facts from issues, questions, TODOs, and decision documents — tracking what was decided, what remains open, and what was changed.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: issue, decision, todo, <subsystem>]
<fact text — issue description, resolution, open question, or task status>

```

## Files to Research

### Primary files (ALL revisions):
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/questions.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/todos.md`
- `/Users/jonnyzzz/Work/conductor-loop/TODO.md` (legacy, if present in history)

### Message bus (decisions logged there):
- `/Users/jonnyzzz/Work/conductor-loop/MESSAGE-BUS.md`

### Decision docs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-2-FINAL-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-4-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-5-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-7-DECISION.md`

### Feature requests:
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/feature-requests-project-goal-manual-workflows.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/SUGGESTED-TASKS.md`

### Swarm issues:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/ISSUES.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/AGGREGATED-REVIEW-FEEDBACK.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/RESEARCH-FINDINGS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/review-execution-model.md`

## Instructions

1. Get full git history for docs/dev issues/questions/todos files:
   `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/dev/issues.md docs/dev/questions.md docs/dev/todos.md TODO.md MESSAGE-BUS.md`

2. Read EACH revision of docs/dev/issues.md to track: when each issue was filed, when resolved, what severity

3. Read EACH revision of docs/dev/questions.md to track: questions and their answers over time

4. Read EACH revision of docs/dev/todos.md to see what tasks were added/completed at each point

5. Read MESSAGE-BUS.md fully — extract all DECISION, FACT, and QUESTION entries

6. Extract facts: issue numbers and status, open vs resolved, design question resolutions, task completions, key decisions

7. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md`

## Start now.
