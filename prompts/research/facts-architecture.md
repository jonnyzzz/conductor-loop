# Research Task: Architecture & Core Design Facts

You are a research agent. Your job is to extract key facts from documents in the conductor-loop project, tracing their evolution through git history.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: architecture, <subsystem>]
<fact text — concrete decision, design choice, or key finding>

```

Newer information overrides older. The author is always Eugene Petrenko (eugene.petrenko@jetbrains.com).

## Files to Research

For each file below, run `git log -- <file>` to get all commits, then read key revisions using `git show <sha>:<file>` to trace evolution. Use `git log -p -- <file>` for diff history.

### Primary files:
- `/Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PLAN_v5.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture-review.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/dependency-analysis.md`
- `/Users/jonnyzzz/Work/conductor-loop/IMPLEMENTATION-README.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-2-FINAL-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-4-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-5-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/problem-7-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/ralph-loop.md`

### Also check swarm legacy for related design history:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/CRITICAL-PROBLEMS-RESOLVED.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/PLANNING-COMPLETE.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/SUBSYSTEMS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/TOPICS.md`

## Instructions

1. Run `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/workflow/THE_PLAN_v5.md THE_PLAN_v5.md docs/dev/architecture-review.md docs/dev/dependency-analysis.md docs/decisions/ docs/dev/architecture.md docs/dev/ralph-loop.md docs/dev/subsystems.md` to get the commit list

2. For each significant commit, read the file state at that commit with `git show <sha>:<file>`

3. Extract facts: key architectural decisions, design choices, phase plans, component responsibilities, critical problems and their resolutions

4. For every fact, include the commit timestamp and relevant tags

5. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

6. Focus on DECISIONS and FACTS, not descriptions. E.g.: "Ralph loop: restarts root agent when DONE file is absent. DONE + running children → wait 300s then stop."

## Start now. Read each file, trace git history, extract facts, write the output file.
