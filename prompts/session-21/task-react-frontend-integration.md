# Task: Fix and Integrate React Frontend

## Context

There is a React 18 + TypeScript frontend in `frontend/` that is NOT currently served by
the conductor server. The active web UI is `web/src/` (plain HTML/JS). The React frontend is
more polished and has better features (LogViewer with filtering, RunTree, etc.).

The React app has API endpoint mismatches vs. what the backend actually provides.
Your task is to fix these mismatches, rebuild the React app, and connect it to the conductor.

## Step 1: Read these files first

1. /Users/jonnyzzz/Work/conductor-loop/AGENTS.md (code style, commit format)
2. /Users/jonnyzzz/Work/conductor-loop/frontend/src/api/client.ts (API endpoint issues)
3. /Users/jonnyzzz/Work/conductor-loop/frontend/src/App.tsx (SSE URL issues)
4. /Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go (findWebDir)
5. /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go (task creation endpoint)

## Step 2: Fix API endpoint mismatches in frontend/src/api/client.ts

### Fix 1: getMessages() - wrong path (/bus → /messages)
Change:
```ts
const data = await this.request<{ messages: BusMessage[] }>(
  `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/bus`
)
```
To:
```ts
const data = await this.request<{ messages: BusMessage[] }>(
  `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/messages`
)
```

### Fix 2: postProjectMessage() - wrong path (/bus → /messages)
Change:
```ts
return this.request<MessageResponse>(`/api/projects/${encodeURIComponent(projectId)}/bus`, {
```
To:
```ts
return this.request<MessageResponse>(`/api/projects/${encodeURIComponent(projectId)}/messages`, {
```

### Fix 3: postTaskMessage() - wrong path (/bus → /messages)
Change:
```ts
return this.request<MessageResponse>(
  `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/bus`,
```
To:
```ts
return this.request<MessageResponse>(
  `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/messages`,
```

### Fix 4: startTask() - wrong endpoint (project-scoped POST doesn't exist)
Change:
```ts
return this.request(`/api/projects/${encodeURIComponent(projectId)}/tasks`, {
  method: 'POST',
  body: payload,
})
```
To:
```ts
return this.request(`/api/v1/tasks`, {
  method: 'POST',
  body: { ...payload, project_id: projectId },
})
```

Note: The `TaskStartRequest` needs `project_id` added. Update the interface:
```ts
export interface TaskStartRequest {
  task_id: string
  prompt: string
  project_root: string
  attach_mode: 'restart' | 'new'
  project_id?: string  // injected by startTask
  agent_type?: string  // set to 'claude' as default
}
```

## Step 3: Fix SSE URL mismatches in frontend/src/App.tsx

### Fix 5: busStreamUrl - wrong path (/bus/stream → /messages/stream)
Change:
```ts
if (busScope === 'project') {
  return `/api/projects/${effectiveProjectId}/bus/stream`
}
...
return `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/bus/stream`
```
To:
```ts
if (busScope === 'project') {
  return `/api/projects/${effectiveProjectId}/messages/stream`
}
...
return `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/messages/stream`
```

### Fix 6: logStreamUrl - endpoint doesn't exist (drop for now)
Change:
```ts
const logStreamUrl = useMemo(() => {
  if (!effectiveProjectId || !effectiveTaskId) {
    return undefined
  }
  return `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/logs/stream`
}, [effectiveProjectId, effectiveTaskId])
```
To just:
```ts
const logStreamUrl = undefined  // task-level log streaming not yet implemented
```

### Fix 7: taskStateQuery - wrong filename (TASK_STATE.md → TASK.md)
Change:
```ts
const taskStateQuery = useTaskFile(effectiveProjectId, effectiveTaskId, 'TASK_STATE.md')
```
To:
```ts
const taskStateQuery = useTaskFile(effectiveProjectId, effectiveTaskId, 'TASK.md')
```

## Step 4: Build the React frontend

```bash
cd /Users/jonnyzzz/Work/conductor-loop/frontend
npm install  # install dependencies if not done
npm run build  # builds to frontend/dist/
```

Verify that `frontend/dist/index.html` exists after build.

## Step 5: Connect React app to conductor server

In `internal/api/routes.go`, update `findWebDir()` to also search for `frontend/dist/`:

After the existing candidates, add:
```go
candidates = append(candidates,
  filepath.Join(base, "frontend", "dist"),
  filepath.Join(base, "..", "frontend", "dist"),
  filepath.Join(base, "..", "..", "frontend", "dist"),
)
```
And in the cwd section:
```go
candidates = append(candidates,
  filepath.Join(cwd, "frontend", "dist"),
  filepath.Join(cwd, "web", "src"),  // existing
)
```

IMPORTANT: Make `frontend/dist` appear BEFORE `web/src` in the candidates list, so it takes priority if the React app is built. This allows the built React app to override the simple web UI.

## Step 6: Add note to docs/dev/architecture.md about both UIs

Add a section to the "Web UI" part of architecture.md explaining:
- **Active (simple) UI**: `web/src/` - plain HTML/CSS/JS served by conductor, zero build step
- **Advanced React UI**: `frontend/` - React 18 + TypeScript, needs `npm run build` in `frontend/`, served from `frontend/dist/` when present

## Step 7: Verify and commit

```bash
# From repo root:
go build ./...  # must pass
go test ./...   # must pass (18 packages)
```

Commit with: `feat(frontend): fix React app API endpoints and integrate with conductor`

## Important Notes

- The React app uses Ring UI (JetBrains) components — don't change UI styling
- If `npm run build` fails, fix TypeScript errors until it succeeds
- The React app uses `import.meta.env.VITE_API_BASE_URL` which defaults to empty string (relative URLs) — this is correct when served from conductor
- Do NOT delete `web/src/` — it remains as the fallback

## Done File

When complete: `echo "done" > "$JRUN_TASK_FOLDER/DONE"`
