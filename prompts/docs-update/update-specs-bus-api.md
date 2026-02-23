# Docs Update: Specifications — Message Bus, Agent Protocol, Frontend-Backend API, Monitoring UI

You are a documentation update agent. Update specs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-object-model.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-object-model-QUESTIONS.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools-QUESTIONS.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol.md`
6. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol-QUESTIONS.md`
7. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-frontend-backend-api.md`
8. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui.md`
9. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui-QUESTIONS.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md
```

## Verify against source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Message bus: actual message struct fields
grep -rn "type.*Message\|MessageID\|ParentID\|BusType\|MsgType\|Timestamp" internal/messagebus/ --include="*.go" | head -30
cat internal/messagebus/messagebus.go | head -100

# API routes
cat internal/api/routes.go 2>/dev/null
grep -rn "HandleFunc\|router\.\|GET\|POST\|DELETE" internal/api/ --include="*.go" | grep -v test | head -40

# SSE endpoint
grep -rn "stream\|/sse\|EventSource\|text/event-stream\|flusher\|Flusher" internal/api/ --include="*.go" | head -20

# Default port
grep -rn "14355\|8080\|DefaultPort\|defaultPort" cmd/ internal/ --include="*.go" | head -10

# UI: React components
ls frontend/src/ 2>/dev/null
cat frontend/package.json 2>/dev/null | grep -E "\"name\"|\"version\"|\"react\"|\"@jetbrains" | head -10
```

## Rules

- **Facts override specs** — if a fact contradicts a spec, update the spec
- Key fixes (from FACTS-messagebus.md and FACTS-agents-ui.md Round 2):
  - Message format: verify actual field names from code structs
  - API port: reconcile 8080 vs 14355 — use verified canonical value
  - SSE: document actual SSE endpoint path and payload structure
  - Monitoring UI stack: React + JetBrains Ring UI + Vite
  - For QUESTIONS files: mark questions as answered where facts provide the answer
  - Remove or resolve stale assumptions documented in Round 2 validation
- API spec: update all endpoint paths, HTTP methods, response shapes to match routes.go

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
