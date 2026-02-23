# Research Task: Prompts, Workflow & Methodology Facts

You are a research agent. Extract key facts from workflow prompt files and methodology documents, tracing their evolution through git history.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-prompts-workflow.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: workflow, prompt, methodology, <subsystem>]
<fact text — workflow step, agent constraint, quality gate, or methodology principle>

```

## Files to Research

### Primary workflow files (ALL revisions):
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_conductor.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_orchestrator.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_implementation.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_research.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_review.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_test.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_monitor.md`
- `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_debug.md`

### Configuration & conventions:
- `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md`
- `/Users/jonnyzzz/Work/conductor-loop/CLAUDE.md`
- `/Users/jonnyzzz/Work/conductor-loop/Instructions.md`

### Dev process:
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/contributing.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/testing.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/rlm-orchestration.md`

### Swarm legacy orchestration prompts:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/ideas.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/TOPICS.md`

## Instructions

1. For each file, get ALL revisions:
   `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- THE_PROMPT_v5.md THE_PROMPT_v5_conductor.md THE_PROMPT_v5_orchestrator.md AGENTS.md CLAUDE.md Instructions.md`

2. Read each revision to see how the methodology evolved

3. For THE_PROMPT_v5.md specifically — read EVERY revision as it contains the core orchestration methodology

4. Extract facts:
   - Agent role definitions (orchestrator vs implementation vs review vs test)
   - Quality gates (what must pass before commit)
   - Workflow stages (numbered steps)
   - Max parallelism limits
   - Commit format requirements
   - RLM methodology application rules
   - CWD guidance for different agent types
   - Constraints (root agent must not modify code directly, etc.)

5. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-prompts-workflow.md`

## Start now.
