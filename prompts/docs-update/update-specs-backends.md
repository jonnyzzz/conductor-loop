# Docs Update: Specifications — Agent Backends (Claude, Codex, Gemini, Perplexity, xAI)

You are a documentation update agent. Update specs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-claude.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-claude-QUESTIONS.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-codex.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-codex-QUESTIONS.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-gemini.md`
6. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-gemini-QUESTIONS.md`
7. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-perplexity.md`
8. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-perplexity-QUESTIONS.md`
9. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-xai.md`
10. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-xai-QUESTIONS.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
```

## Verify against source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Claude: exact CLI flags
cat internal/agent/claude/claude.go

# Codex: exact CLI flags
cat internal/agent/codex/codex.go

# Gemini: CLI implementation vs REST
cat internal/agent/gemini/gemini.go
# Also check runner usage of gemini
grep -A3 "gemini" internal/runner/job.go | head -30

# Perplexity: REST implementation
cat internal/agent/perplexity/perplexity.go 2>/dev/null | head -60

# xAI: current state
cat internal/agent/xai/xai.go 2>/dev/null | head -60
ls internal/agent/xai/ 2>/dev/null

# Env var names per agent
grep -rn "ANTHROPIC_API_KEY\|OPENAI_API_KEY\|GEMINI_API_KEY\|PERPLEXITY_API_KEY\|XAI_API_KEY" internal/ --include="*.go" | head -20

# Version detection
grep -n "version\|Version\|parseVersion\|minVersion" internal/runner/validate.go | head -30

# isRestAgent function
grep -n "isRestAgent\|RestAgent\|rest.*agent" internal/runner/job.go | head -10

# Round-robin / diversification
grep -rn "diversif\|RoundRobin\|round.robin\|weighted\|fallback" internal/ --include="*.go" | head -20
```

## Rules

- **Facts override specs** — if a fact contradicts a spec, update the spec
- Key fixes (from FACTS-agents-ui.md Round 2):
  - **Claude**: document exact flags `-p`, `--input-format`, `--output-format`, `--tools`, `--permission-mode`
  - **Codex**: document exact flags, `-C` working directory flag, `--dangerously-skip-permissions`
  - **Gemini**: CRITICAL — runner uses `gemini` CLI with `--screen-reader true --approval-mode yolo --output-format stream-json`; note the unused REST GeminiAgent in gemini.go. Add TODO about stream-json fallback for older versions.
  - **Perplexity**: REST-only agent, isRestAgent=true, confirm API endpoint and model
  - **xAI**: document as deferred/placeholder; state current status clearly; plan: OpenCode targeting xAI models, default grok-4
  - **Env vars**: use exact names verified from code
  - **Version detection**: document min versions if any
- For QUESTIONS files: mark resolved questions; add answers from facts
- Do not invent unverified details

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
