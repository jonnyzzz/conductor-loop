# Task: Add Project Dashboard Panel to Web UI

## Context

This is a Go-based multi-agent orchestration framework with a React+TypeScript frontend.
The current UI shows:
- TaskList panel (list of tasks with status filters and search)
- TaskDetail panel (task message bus and run list)
- RunDetail panel (run output and details)

The project already has a `/api/projects/{p}/stats` endpoint that returns:
```json
{
  "project_id": "conductor-loop",
  "total_tasks": 75,
  "total_runs": 33,
  "running_runs": 0,
  "completed_runs": 33,
  "failed_runs": 0,
  "crashed_runs": 0,
  "message_bus_files": 75,
  "message_bus_total_bytes": 104891
}
```

This data is not surfaced in the UI. Adding a project dashboard would make the
tool more useful for monitoring ongoing orchestration work.

## Your Task

Add a project dashboard/stats panel to the frontend.

### Read First

Before implementing, read these files to understand the existing codebase:
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/App.tsx` - main app structure
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/components/TaskList.tsx` - current left panel
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/components/TaskDetail.tsx` - task view
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/api/` - API client functions (look for how stats might be called or would be called)
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/types/` - TypeScript types
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` - backend stats handler

### Implementation Plan

#### 1. Add TypeScript type for ProjectStats
In the appropriate types file, add:
```typescript
interface ProjectStats {
  project_id: string;
  total_tasks: number;
  total_runs: number;
  running_runs: number;
  completed_runs: number;
  failed_runs: number;
  crashed_runs: number;
  message_bus_files: number;
  message_bus_total_bytes: number;
}
```

#### 2. Add API function to fetch stats
In the API client file, add a `getProjectStats(projectId: string)` function
that calls `GET /api/projects/{p}/stats`.

#### 3. Create ProjectStats component

Create `/Users/jonnyzzz/Work/conductor-loop/frontend/src/components/ProjectStats.tsx`

The component should:
- Accept `projectId: string` as a prop
- Fetch stats from the API on mount and refresh every 10 seconds (use useEffect + setInterval)
- Display stats in a compact summary bar or card at the top of the TaskList panel
- Show: Total Tasks, Completed Runs, Running Runs, Failed Runs
- Format message_bus_total_bytes as human-readable (KB/MB)
- Show a loading state while fetching
- Handle errors gracefully (show "Stats unavailable" on error)

Design considerations:
- Keep it compact - it should fit above the task search bar without taking too much space
- Use consistent styling with the existing UI (check existing CSS/style patterns)
- Color-code status counts: running=blue, completed=green, failed=red

#### 4. Integrate into TaskList or App

- Add `<ProjectStats projectId={selectedProject} />` near the top of the TaskList
  OR at the top of the main content area when a project is selected
- Position it ABOVE the search bar and status filter buttons

#### 5. Style it

Add CSS to the appropriate stylesheet. Keep it minimal and consistent with existing styles.

### Build and Test

After making changes:
```bash
cd /Users/jonnyzzz/Work/conductor-loop/frontend
npm run build
```

Verify the build succeeds. The built output goes to `frontend/dist/` which the
Go server serves.

Test manually by checking that:
1. The stats panel appears when a project is selected
2. Numbers update every 10 seconds
3. The layout doesn't break existing functionality

### Backend Enhancement (Optional)

If the stats API is missing useful data, you can enhance it. Look at:
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`
- The `handleProjectStats` function
- Consider adding: avg_run_duration, last_activity timestamp

If you add fields, also add corresponding tests:
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/` test files

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop
go build ./...              # must pass (if backend changes made)
go test ./...               # must pass (if backend changes made)
cd frontend
npm run build               # must succeed
```

## Output

Write a summary to the output.md file in your run directory.

Commit:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
git add frontend/ internal/api/
git commit -m "feat(ui): add project stats dashboard panel showing task and run counts"
```

Follow AGENTS.md commit format.

## Important Notes

- The frontend is at `/Users/jonnyzzz/Work/conductor-loop/frontend/`
- It uses React 18 + TypeScript
- The built output must be regenerated: `npm run build`
- The server serves `frontend/dist/` at `/ui/`
- Use `fetch()` for API calls or whatever the existing pattern is in the codebase
- Keep changes focused and avoid refactoring unrelated code
