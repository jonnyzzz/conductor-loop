# Task: Frontend - Fix Task Visibility & Add Search

## Context

You are a sub-agent working on the conductor-loop project. This is a Go-based multi-agent
orchestration framework with a React/TypeScript frontend.

**Working directory**: /Users/jonnyzzz/Work/conductor-loop

## Problem

The task list in the web UI only shows 50 tasks (API default pagination limit), but the
`runs/conductor-loop/` directory has 71 tasks. The 21 newest tasks are hidden because:

1. `frontend/src/api/client.ts` `getTasks()` calls `/api/projects/{p}/tasks` with no explicit limit
2. The API defaults to limit=50
3. The frontend has no "Load More" button or search to work around this

## What To Fix

### Fix 1: Request all tasks (up to 500)

In `frontend/src/api/client.ts`, change `getTasks()` to request with `?limit=500`:

```typescript
async getTasks(projectId: string): Promise<TaskSummary[]> {
  const data = await this.request<TasksResponse>(
    `/api/projects/${encodeURIComponent(projectId)}/tasks?limit=500`
  )
  return data.items
}
```

### Fix 2: Add text search bar to TaskList

In `frontend/src/components/TaskList.tsx`, add a text search input above the status filter
buttons. Filter tasks by task ID substring match (case-insensitive).

Add state:
```typescript
const [searchText, setSearchText] = useState('')
```

Modify `filteredTasks` to also filter by search text:
```typescript
const filteredTasks = useMemo(() => {
  let filtered = tasks
  if (searchText.trim()) {
    const q = searchText.trim().toLowerCase()
    filtered = filtered.filter((task) => task.task_id.toLowerCase().includes(q))
  }
  if (statusFilter !== 'all') {
    filtered = filtered.filter((task) => task.status === statusFilter)
  }
  return [...filtered].sort((a, b) => parseDate(b.last_activity) - parseDate(a.last_activity))
}, [statusFilter, searchText, tasks])
```

Add a search input element (text input, styled consistently with the existing UI):
- Placeholder: "Search tasks..."
- Clear button (×) when text is non-empty
- Width: full width of the task list panel

### Fix 3: Show task count

Below the search/filter row, show: "Showing N of M tasks" when filtering is active.

## Files to modify

1. `frontend/src/api/client.ts` - add `?limit=500` to getTasks
2. `frontend/src/components/TaskList.tsx` - add search bar and task count
3. `frontend/src/index.css` - add any new CSS needed for search bar

## After making code changes

1. Build the frontend: `cd /Users/jonnyzzz/Work/conductor-loop/frontend && npm run build`
2. Verify build succeeds (no TypeScript errors)
3. Run Go tests: `cd /Users/jonnyzzz/Work/conductor-loop && go test ./...`
4. Commit with: `feat(ui): add task search and fix pagination limit`

## Quality gates

- No TypeScript errors in `npm run build`
- `go test ./...` still passes
- Search input works correctly (tested by reading code, not browser)

## Important notes

- Use Ring UI components where possible (already used in the project)
- Follow existing code style in TaskList.tsx
- DO NOT add server-side search to the API - client-side filtering is sufficient
- DO NOT break existing status filter functionality

## Commit format

```
feat(ui): add task search bar and fix pagination limit

- Request limit=500 in getTasks to show all tasks (was 50)
- Add text search bar to TaskList to filter by task ID
- Show task count when search/filter is active
```

Write a summary of your changes to: /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/JRUN_TASK_FOLDER/DONE
(JRUN_TASK_FOLDER will be set as an env var in your runtime environment)
