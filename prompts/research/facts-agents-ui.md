# Research Task: Agent Backends & Monitoring UI Facts

You are a research agent. Extract key facts from agent backend and monitoring UI specifications, tracing their evolution.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: agent-backend, ui, <agent-name>]
<fact text — CLI flags, API keys, env vars, UI layout decisions>

```

## Files to Research

### Current agent backend specs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-claude.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-claude-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-codex.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-codex-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-gemini.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-gemini-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-perplexity.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-perplexity-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-xai.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-xai-QUESTIONS.md`

### UI specs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-frontend-backend-api.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/adding-agents.md`

### Legacy swarm specs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-backend-claude.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-backend-codex.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-backend-gemini.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-backend-perplexity.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-backend-xai.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-monitoring-ui.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-frontend-backend-api.md`

## Instructions

1. For each file, trace git history: `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- <file>`

2. Read each significant revision

3. Extract facts:
   - Claude: exact CLI flags used, env var name, version detected, streaming behavior
   - Codex: exact CLI flags, env var, dangerous-bypass flag, CWD handling
   - Gemini: CLI flags, streaming behavior, output format
   - Perplexity: REST endpoint, SSE streaming, model names
   - xAI: Grok model, Agent Tools API, code execution sandbox
   - UI: 3-panel layout (tree/bus/output), SSE streaming, task creation dialog, message bus display
   - API: endpoints, pagination, SSE paths, auth

4. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`

## Start now.
