# Docs Update: Dev Docs — Agent Protocol, Message Bus, Adding Agents, Logging

You are a documentation update agent. Update docs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/agent-protocol.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/message-bus.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/adding-agents.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/logging-observability.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
```

## Verify against source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Message bus implementation
ls internal/messagebus/
cat internal/messagebus/messagebus.go | head -80
grep -n "fsync\|Fsync\|sync\|lock\|Lock\|rotate\|Rotate" internal/messagebus/messagebus.go | head -20

# Agent implementations
ls internal/agent/
cat internal/agent/claude/claude.go | head -60
cat internal/agent/codex/codex.go | head -60
cat internal/runner/job.go | grep -A5 "gemini\|claude\|codex\|isRestAgent" | head -40

# Env var names
grep -rn "ANTHROPIC_API_KEY\|OPENAI_API_KEY\|GEMINI_API_KEY\|PERPLEXITY_API_KEY\|XAI_API_KEY" internal/ --include="*.go" | head -20

# Agent protocol: env injection
grep -rn "RUNS_DIR\|MESSAGE_BUS\|JRUN_\|RUN_ID" internal/runner/ --include="*.go" | head -20

# Adding a new agent: agent factory/registry
cat internal/agent/factory.go 2>/dev/null | head -60
ls internal/agent/

# Logging
grep -rn "slog\|zerolog\|zap\|logrus\|log\." internal/ --include="*.go" | grep "import\|NewLogger\|log\.New" | head -10
```

## Rules

- **Facts override docs** — if a fact contradicts a doc, update the doc to match the fact
- Key fixes for message-bus.md (from FACTS-messagebus.md Round 2):
  - Lock/write strategy: document actual approach from code
  - fsync default: document actual default with correct value
  - Message format: verify ID format, field names
  - Rotation: verify if/when rotation happens
  - CLI auto-discovery: document env inference behavior
- Key fixes for adding-agents.md (from FACTS-agents-ui.md Round 2):
  - Gemini: clarify CLI agent (runner uses CLI) vs unused REST impl in gemini.go
  - Exact CLI flags for each agent type
  - isRestAgent: document which agents use REST vs CLI
  - Add xAI backend status (deferred/placeholder)
- Key fixes for agent-protocol.md:
  - Exact env vars injected: RUNS_DIR, MESSAGE_BUS, JRUN_* — verify names
  - output.md fallback behavior
  - Run lifecycle events
- Do not rewrite from scratch — targeted corrections only

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
