# Research Task: Swarm Ideas & Legacy Design Facts

You are a research agent. Extract key facts from the swarm legacy documentation and ideas files, which represent the original design thinking before conductor-loop was built.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-swarm-ideas.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: swarm, idea, legacy, <subsystem>]
<fact text — original idea, design principle, or architectural intent>

```

Use git timestamps from: `cd /Users/jonnyzzz/Work/jonnyzzz-ai-coder && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- swarm/docs/legacy/ideas.md swarm/docs/legacy/TOPICS.md swarm/docs/legacy/SUBSYSTEMS.md`

(Note: these files no longer exist in jonnyzzz-ai-coder but their history remains accessible via git)

## Files to Research

### Primary ideas files:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/ideas.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/TOPICS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/SUBSYSTEMS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/ROUND-6-SUMMARY.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/ROUND-7-SUMMARY.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/PLANNING-COMPLETE.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/AGGREGATED-REVIEW-FEEDBACK.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/RESEARCH-FINDINGS.md`

### Problem decisions:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-1-decision.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-2-FINAL-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-4-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-5-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-7-DECISION.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/prompt-project-naming.md`

### Also check jonnyzzz-ai-coder git history for earliest versions:
`cd /Users/jonnyzzz/Work/jonnyzzz-ai-coder && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- swarm/docs/legacy/ideas.md | head -20`

Read earliest revision of ideas.md:
`cd /Users/jonnyzzz/Work/jonnyzzz-ai-coder && git show $(git log --format="%H" -- swarm/docs/legacy/ideas.md | tail -1):swarm/docs/legacy/ideas.md`

## Instructions

1. Read all legacy swarm files listed above

2. Get git history from jonnyzzz-ai-coder for the ideas.md file (all revisions)

3. Read the earliest revision of ideas.md to capture original vision

4. Extract facts from ideas.md:
   - Core concepts (Ralph Loop, run-agent, swarm pattern)
   - Technology choices (Go, React, HCL config, filesystem-first)
   - Original feature requirements (what Eugene wanted from day 1)
   - Constraints and non-goals
   - Naming decisions (why "conductor-loop" was chosen)

5. Extract facts from TOPICS.md and SUBSYSTEMS.md:
   - All major subsystems and their responsibilities
   - Design topics and their resolved decisions

6. Extract facts from problem decisions:
   - Each problem number, description, and resolution

7. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-swarm-ideas.md`

## Start now.
