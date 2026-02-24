# Fix Security, Agent, Deployment Docs

Your task is to fix `security.md`, `agent-integration.md`, and `deployment.md`.

## Instructions
1. **security.md**:
   - UI Guard: Remove "User-Agent" from detection list (uses Origin/Referer/Headers).
   - Middleware order: Guard runs in handler, *after* Auth middleware. Unauth requests fail 401 first.
2. **agent-integration.md**:
   - CLI Invocation: Remove `-C <cwd>` flags. Runner sets CWD via process attributes.
3. **deployment.md**:
   - Mark `TASK-CONFIG.yaml` as (Optional) in directory tree.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/review-notes-C.md`
