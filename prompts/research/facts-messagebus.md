# Research Task: Message Bus & Agent Protocol Facts

You are a research agent. Extract key facts from message bus and agent protocol documents, tracing their evolution through git history.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: messagebus, agent-protocol, <subsystem>]
<fact text — concrete decision, format spec, or implementation detail>

```

## Files to Research

### Current specifications:
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-object-model.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-object-model-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol-QUESTIONS.md`

### Dev docs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/message-bus.md`

### Root level:
- `/Users/jonnyzzz/Work/conductor-loop/MESSAGE-BUS.md` (actual project message bus — read for decisions and facts logged there)

### Legacy swarm specs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-message-bus-tools.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-message-bus-tools-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-message-bus-object-model.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-message-bus-object-model-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-protocol.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-agent-protocol-QUESTIONS.md`

### Problem decisions (message bus race condition):
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-1-decision.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-1-message-bus-race.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/problem-1-review.md`

### jonnyzzz-ai-coder project (message bus MCP):
- `/Users/jonnyzzz/Work/jonnyzzz-ai-coder/message-bus-mcp/` (read any relevant specs here)

## Instructions

1. For each file, get git history: `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- <file>`

2. Read each significant revision

3. Also read the jonnyzzz-ai-coder message-bus-mcp directory: `ls /Users/jonnyzzz/Work/jonnyzzz-ai-coder/message-bus-mcp/`

4. Extract facts: message format (YAML front-matter), msg_id format, message types (FACT/QUESTION/ANSWER/START/STOP/etc.), threading via parents[], locking strategy (O_APPEND + flock), file scope (PROJECT vs TASK bus), size limits, rotation policy, bus auto-discovery rules, environment variable names, CLI commands

5. Note evolution: what changed from legacy swarm specs to current conductor-loop implementation

6. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md`

## Start now.
