# Task: Add OUTPUT Tab to Web UI

## Context

The conductor-loop project is at /Users/jonnyzzz/Work/conductor-loop.

The web UI is at `web/src/index.html` and `web/src/app.js`.

Currently the run detail panel has 4 tabs:
- STDOUT → loads `file?name=stdout` (agent-stdout.txt)
- STDERR → loads `file?name=stderr` (agent-stderr.txt)
- PROMPT → loads `file?name=prompt` (prompt.md)
- MESSAGES → loads messages API

**The gap:** Agents are instructed to write their work product to `output.md` in the run directory (via the prompt preamble: "Write output.md to <path>"). The server's `serveRunFile` function (internal/api/handlers_projects.go:246) already supports `name=output.md`. But the web UI has no tab for it!

This means the most important artifact (the agent's actual work output) is not visible in the web UI.

## What to Implement

1. **Add an OUTPUT tab** to `web/src/index.html` between PROMPT and MESSAGES:
   ```html
   <button class="tab-btn" data-tab="output.md">OUTPUT</button>
   ```

2. **The `loadTabContent()` function in `web/src/app.js`** already handles file loading generically:
   ```javascript
   const data = await apiFetch(`${prefix}/runs/${enc(state.selectedRun)}/file?name=${enc(tab)}`);
   el.textContent = data.content || '(empty)';
   ```
   This will just work — `tab` will be `"output.md"` and the server already handles it.

3. **Also add CSS class for `output.md`** - In `msgTypeClass()` and `stClass()`, no changes needed. But in `switchTab()` the tab comparison uses `b.dataset.tab === name` which will work.

4. **Set OUTPUT as the default activeTab** instead of `stdout`. The output.md is more useful than raw stdout for most use cases. Change `activeTab: 'stdout'` to `activeTab: 'output.md'` in the state object.

5. **Auto-refresh for running tasks** - When the selected run has `status === 'running'` and the OUTPUT tab is active, consider refreshing more frequently. Actually, the existing 5-second `fullRefresh()` already handles this via the refresh timer. No additional code needed.

## Quality Gates

1. After changes:
   ```
   go build ./...  # should still pass (no Go changes)
   go test ./...   # should still pass (no Go changes)
   ```

2. Verify the OUTPUT tab HTML appears in the correct position in index.html.

3. Verify the `state.activeTab` default is changed to `'output.md'`.

## Files to Change

- `/Users/jonnyzzz/Work/conductor-loop/web/src/index.html` — add OUTPUT tab button
- `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js` — change default activeTab to 'output.md'

## Commit Format (from AGENTS.md)

```
feat(web): add OUTPUT tab for output.md in run detail view
```

## Signal Completion

When done, create the DONE file:
```bash
touch "$TASK_FOLDER/DONE"
```
