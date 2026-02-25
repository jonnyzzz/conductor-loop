# Task: Add Task Creation Dialog to React Frontend

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

This is a Go-based multi-agent orchestration framework. The project has:
- A Go backend with REST API
- A React 18 + TypeScript frontend in frontend/ using @jetbrains/ring-ui-built components
- The React frontend is built to frontend/dist/ and served by the conductor at /ui/

## Current Gap

The React frontend (`frontend/src/`) has NO task creation UI. Tasks can only be started from:
- The old simple HTML UI at `web/src/index.html` (has a `<dialog>` form)
- The `conductor job submit` CLI command

The `web/src/index.html` has a task creation form with these fields:
- project_id (text, required)
- task_id (text, auto-generated with timestamp format)
- agent_type (select: claude/codex/gemini)
- prompt (textarea, required)
- project_root (text)
- attach_mode (select: create/attach/resume)

## API Endpoint

POST `/api/v1/tasks` with body:
```json
{
  "project_id": "my-project",
  "task_id": "task-20260220-190000-my-task",
  "agent_type": "claude",
  "prompt": "Do something useful",
  "project_root": "/path/to/project",
  "attach_mode": "create"
}
```

Returns 201 on success with `{"task_id": "...", "run_id": "..."}`.

The backend handler is `handleTasks` in `internal/api/handlers.go`.

## What To Implement

### Add task creation to React frontend

Look at the existing components in `frontend/src/components/TaskList.tsx`. This component shows
the list of projects and tasks. Add a "New Task" button that opens a creation dialog.

Also read:
- `frontend/src/api/client.ts` — the API client (add `startTask` method if not present)
- `frontend/src/hooks/useAPI.tsx` — hooks that wrap the API client (add mutation hook)
- `frontend/src/components/TaskList.tsx` — where to add the button + dialog
- `frontend/src/types/index.ts` — add TaskCreateRequest type if needed

### Implementation Details

1. **Add `startTask` to `APIClient` in `frontend/src/api/client.ts`**:
   ```typescript
   async startTask(req: TaskStartRequest): Promise<{ task_id: string; run_id: string }> {
     return this.request<{ task_id: string; run_id: string }>('/api/v1/tasks', {
       method: 'POST',
       body: req,
     })
   }
   ```
   (Note: `TaskStartRequest` interface may already exist in client.ts — check before adding)

2. **Add `useStartTask` hook in `frontend/src/hooks/useAPI.tsx`**:
   A mutation hook that calls `apiClient.startTask()`.

3. **Add task creation dialog to `TaskList.tsx`**:
   - A "+" or "New Task" button in the header of the TaskList panel
   - Opens a dialog/modal with the form fields
   - Use Ring UI `Button`, `Dialog` or `<dialog>` element, `Input`, `Select`, `Textarea` components
   - Fields: project_id (pre-filled from selected project), task_id (auto-generated), agent_type, prompt, project_root, attach_mode
   - Task ID auto-generation: `task-${dateStr}-${timeStr}-${randomSuffix}` where dateStr is YYYYMMDD and timeStr is HHMMSS
   - On submit: call `useStartTask`, show success/error toast
   - On success: refresh the task list

4. **Use Ring UI components** - look at how other components use them (Button, etc.)
   Import pattern: `import Button from '@jetbrains/ring-ui-built/components/button/button'`
   Check what's available: `ls frontend/node_modules/@jetbrains/ring-ui-built/components/`

### Task ID Format

Task IDs MUST follow: `task-<YYYYMMDD>-<HHMMSS>-<slug>`
Example: `task-20260220-190000-my-task`

Auto-generate with something like:
```typescript
function generateTaskId(): string {
  const now = new Date()
  const date = now.toISOString().slice(0, 10).replace(/-/g, '')
  const time = now.toTimeString().slice(0, 8).replace(/:/g, '')
  const rand = Math.random().toString(36).slice(2, 8)
  return `task-${date}-${time}-${rand}`
}
```

## Files to Modify

- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/api/client.ts` — add startTask method
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/hooks/useAPI.tsx` — add useStartTask hook
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/components/TaskList.tsx` — add create button + dialog
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/types/index.ts` — add types if needed

After making changes, rebuild:
```bash
cd /Users/jonnyzzz/Work/conductor-loop/frontend
npm run build
```

## Quality Gates

1. `go build ./...` must pass (should be unaffected by React-only changes)
2. React app rebuilds without TypeScript errors: `cd frontend && npm run build`
3. `frontend/dist/index.html` is updated

## Completion

Create a `DONE` file in `$JRUN_TASK_FOLDER` when complete. Write a summary to `$JRUN_RUN_FOLDER/output.md`.

Commit your changes with:
```
feat(frontend): add task creation dialog to React UI
```

Follow the project's commit convention from AGENTS.md.
