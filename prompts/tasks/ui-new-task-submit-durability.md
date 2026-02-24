# Task: New Task Submit Durability — Persist Draft, Audit Submit Lifecycle

## Context

- **ID**: `task-20260223-155330-ui-new-task-submit-durability-regression-guard`
- **Priority**: P1
- **Source**: `docs/dev/todos.md`

The "New Task" dialog in the Web UI has two known durability problems:
1. **Form data loss on reload**: If the user refreshes the page while the dialog is open,
   all entered data (task name, prompt text) is lost with no recovery path.
2. **Submit lifecycle ambiguity**: After clicking Submit, there is no clear indication of
   whether the task was created, queued, failed, or is pending. The UI may briefly flash
   success then clear the form — but if the API call fails, the form clears anyway.

## Requirements

1. **Draft persistence**: Auto-save form contents to `localStorage` while typing. On page
   reload, if a draft is present, offer to restore it ("Restore unsaved task draft?").
   Draft is cleared on successful submit or explicit discard.

2. **Submit state machine**: Implement explicit submit states:
   - `idle` (default) → `submitting` (spinner + disabled button) → `success` (green confirmation)
     → auto-close after 2s, OR
   - `submitting` → `error` (red inline error + retry button, form data preserved)

3. **Error display**: On API error, show the error message inline (not just a toast that
   disappears) and keep the form open for retry.

4. **Tests**: Add Vitest/React Testing Library tests for draft save/restore behavior;
   submit state transitions; and error recovery.

## Acceptance Criteria

- Typing in the New Task dialog and refreshing the page shows restore prompt.
- Clicking Submit while API is pending shows spinner and disables the button.
- API failure keeps form open with error message; retry button re-submits without re-typing.
- All new tests pass (`npm test -- --run`).
- No regressions in existing form behavior.

## Verification

```bash
cd frontend
npm test -- --run --reporter verbose 2>&1 | grep -E 'submit|draft|PASS|FAIL'
# Manual: type in New Task dialog, refresh page, verify restore prompt
# Manual: submit with invalid data, verify error shows inline
```

## Reference Files

- `frontend/src/components/NewTaskDialog.*` — New Task dialog component
- `frontend/src/` — React app entrypoint and routing
- `internal/api/tasks.go` — task creation API endpoint
- `docs/dev/todos.md` — feature request origin
