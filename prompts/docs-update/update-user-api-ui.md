# Docs Update: User Docs — API Reference, Web UI, FAQ, Troubleshooting, RLM Orchestration

You are a documentation update agent. Update docs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/user/api-reference.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/user/web-ui.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/user/faq.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/user/troubleshooting.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/user/rlm-orchestration.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
```

## Verify API routes against source

```bash
cd /Users/jonnyzzz/Work/conductor-loop
cat internal/api/routes.go 2>/dev/null || find internal/api -name "*.go" | head -5
grep -rn "HandleFunc\|router\.\|mux\.\|GET\|POST\|DELETE\|PUT" internal/api/ --include="*.go" | grep -v test | head -40

# Verify port
grep -rn "14355\|8080\|DefaultPort\|default.*port" cmd/ internal/api/ --include="*.go" | head -10

# Verify SSE endpoints
grep -rn "stream\|/sse\|EventSource\|text/event-stream" internal/api/ --include="*.go" | head -20

# Verify frontend stack
cat frontend/package.json | head -30
ls frontend/src/ | head -10
```

## Rules

- **Facts override docs** — if a fact contradicts a doc, update the doc to match the fact
- Key fixes needed:
  - Default port: reconcile 8080 vs 14355 — use the verified canonical value
  - API endpoint paths: update to match actual routes from code
  - SSE endpoints: verify paths and payloads from source
  - Frontend stack: confirm React + JetBrains Ring UI + Vite
  - Authentication: document the actual auth state (no auth vs token)
  - Pagination: document the actual pagination state
- For troubleshooting.md: add any known issues from FACTS-issues-decisions.md and FACTS-runs-conductor.md (recurring blockers)
- For rlm-orchestration.md: verify against actual RLM methodology and conductor-loop implementation
- Preserve correct content — do not rewrite from scratch
- Only fix what is factually wrong or missing

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
