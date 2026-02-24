# Fix High-Level & Data Flow Docs

Your task is to fix issues in `overview.md`, `components.md`, `data-flow-task-lifecycle.md`, and `data-flow-api.md`.

## Instructions
1. **overview.md**:
   - Clarify the "API server never calls back" statement. It implies no *network* callback. Acknowledge that API server imports runner code to trigger stops (via process signals).
2. **components.md**:
   - Ensure consistency with overview regarding API->Runner dependency (it exists for control actions like Stop).
3. **data-flow-task-lifecycle.md**:
   - Phase 3: Add mention of Agents posting `FACT`, `PROGRESS`, `DECISION` messages.
   - Phase 3: Clarify `output.md` is primarily written by Agent; runner ensures existence/fallback.
4. **data-flow-api.md**:
   - Add note: On Windows, mandatory file locking may cause API reads to block behind writers.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/review-notes-A.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/review-notes-B.md`
