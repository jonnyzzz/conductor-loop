# Task: Add /api/v1/status Endpoint

## Required Reading (absolute paths)
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/questions.md (see Q8)
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui-QUESTIONS.md

## Context
Per human answer in monitoring-ui-QUESTIONS.md: "yes" to project-scoped API endpoints.
The decision in QUESTIONS.md Q8 states: Add `/api/v1/status` endpoint that returns active runs count, server uptime, and configured agents. The `/api/v1/health` endpoint stays for simple liveness checks.

## Task
1. Read the existing API code in `/Users/jonnyzzz/Work/conductor-loop/internal/api/` to understand the handler patterns
2. Add a new `/api/v1/status` endpoint to the REST API that returns:
   - `active_runs_count`: number of currently running (non-completed) runs
   - `uptime_seconds`: server uptime since start
   - `configured_agents`: list of agent names from config
   - `version`: current version string
3. Follow existing handler patterns in the API package
4. Add a test for the new endpoint
5. Verify: `go build ./...` passes
6. Verify: `go test ./...` passes

## Constraints
- Follow existing code patterns and conventions from AGENTS.md
- Use lowercase error messages
- Keep the change minimal and focused
- Do NOT modify MESSAGE-BUS.md or ISSUES.md
