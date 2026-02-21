# Sub-Agent Task: Add Run Status Filter to Web UI (Session #28)

## Role
You are an implementation agent. Your CWD is /Users/jonnyzzz/Work/conductor-loop.

## Context

The conductor-loop web UI (React + TypeScript + Ring UI) shows a list of tasks and runs.
Currently there is no way to filter runs/tasks by status. When a project has many runs
(e.g., 100+ runs across many tasks), it's hard to find just the failing ones.

The frontend is at `/Users/jonnyzzz/Work/conductor-loop/frontend/`.
The built output goes to `/Users/jonnyzzz/Work/conductor-loop/frontend/dist/`.

## Task

Add status filter controls to the React frontend. The filter should:

1. **Location**: Add filter buttons/tabs at the top of the run list in the TaskDetail panel
   (the panel that shows runs when you click on a task)

2. **Filter options**:
   - "All" - show all runs (default)
   - "Running" - show only runs with status=running
   - "Completed" - show only runs with status=completed
   - "Failed" - show only runs with status=failed or crashed

3. **Visual style**: Use Ring UI Tab or Button components to match existing UI style.
   Look at existing Ring UI usage in the codebase for patterns.

4. **Behavior**: Filter is applied client-side (already have the run data, just filter display).

5. **Also add to TaskList panel**: Show a status summary badge next to each task showing
   counts like "3 running / 5 completed / 1 failed". This helps see project health at a glance.

## Technical Details

### Key Files to Examine First:
- `frontend/src/components/TaskList.tsx` - task list panel
- `frontend/src/components/TaskDetail.tsx` or similar - task detail/run list
- `frontend/src/types.ts` - RunSummary and RunInfo types
- `frontend/src/App.tsx` - main app structure

### Ring UI Components to Consider:
```typescript
import { Button, Tabs, TabsItem } from '@jetbrains/ring-ui';
// or
import Tabs from '@jetbrains/ring-ui/dist/tabs/tabs';
```

### Run Status Types (from types.ts):
```typescript
type RunStatus = 'running' | 'completed' | 'failed' | 'crashed';
```

## Implementation Steps

1. First, read the existing component files to understand the structure
2. Add a status filter state to the appropriate component
3. Add filter UI (buttons/tabs)
4. Apply filter to the runs list
5. Add status summary badges to task list items
6. Build the frontend: `cd frontend && npm run build`
7. Verify `frontend/dist/` is updated

## Quality Gates

After implementation:
1. `cd frontend && npm run build` must succeed (no TypeScript errors)
2. The built `frontend/dist/index.html` must exist
3. Visual inspection: filter buttons appear and work correctly

## Commit Message Format

Use format: `feat(ui): add run status filter and task status badges to web UI`

When done, create a DONE file at the task directory to signal completion:
`touch <TASK_FOLDER>/DONE`
