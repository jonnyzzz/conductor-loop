# Fix Frontend & Observability Docs

Your task is to fix `frontend-architecture.md` and `observability.md`.

## Instructions
1. **frontend-architecture.md**:
   - Change API routes from `/api/v1/projects...` to `/api/projects...`.
   - Clarify embedding: Fallback UI (`web/src`) is embedded. Primary React UI (`frontend/dist`) is served from filesystem if present (not embedded).
2. **observability.md**:
   - Fix metric names:
     - `conductor_active_runs` -> `conductor_active_runs_total`
     - `conductor_message_bus_append_total` -> `conductor_messagebus_appends_total`
     - `conductor_queued_runs` -> `conductor_queued_runs_total`
   - Add `conductor_completed_runs_total`.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/review-notes-C.md`
