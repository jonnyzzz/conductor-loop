# Task: output.md fallback to agent-stdout.txt in File API

## Context

You are working in the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

The project has a file-serving API endpoint at:
`GET /api/projects/{project}/tasks/{task}/runs/{runID}/file?name=<filename>`

This serves files from the run directory. The web UI defaults to the `output.md` tab, but agents don't always create output.md — especially older runs or runs that crashed.

Currently, when `output.md` doesn't exist, the API returns 404 and the UI shows "(output.md not available)".

## Goal

Modify the file endpoint to fall back to `agent-stdout.txt` when `output.md` is requested but not found.

### Changes to `handlers_projects.go`

Find the handler for `GET .../file?name=...` (likely called `handleRunFile` or similar).

Add fallback logic:
```
When name == "output.md" AND the file doesn't exist:
  Try reading "agent-stdout.txt" instead
  If agent-stdout.txt exists: return its content with an extra field "fallback": "agent-stdout.txt"
  If agent-stdout.txt also doesn't exist: return 404 as before
```

The response JSON should have an additional `"fallback"` field when the fallback was used:
```json
{
  "name": "output.md",
  "content": "...(stdout content)...",
  "fallback": "agent-stdout.txt"
}
```

This allows the web UI to optionally show a notice that it's showing stdout instead.

### Changes to `app.js` (web/src/)

In the `loadTabContent()` function, when the file API response includes `data.fallback`:
- Add a visual notice at the top of the tab content:
  `"[Note: output.md not found, showing agent-stdout.txt]"`
- Then show the content below

This is a simple string prepend before `el.textContent = data.content`.

### Also: SSE Streaming Fallback

The SSE streaming endpoint at `GET .../stream?name=...` should also support the same fallback.
When `name=output.md` and the file doesn't exist, try streaming `agent-stdout.txt` instead.

Find the stream handler in `handlers_projects.go` and add the same fallback logic.

## File Locations

- Project handlers: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`
- Project handlers tests: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go`
- Web UI: `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js`

## How to Find the Right Code

Look in `handlers_projects.go` for a function that:
1. Extracts `name` from the query string
2. Reads a file from the run directory
3. Returns JSON with `content` field

The fallback logic should be:
```go
filePath := filepath.Join(runDir, name)
if _, err := os.Stat(filePath); os.IsNotExist(err) && name == "output.md" {
    // Try fallback
    fallbackPath := filepath.Join(runDir, "agent-stdout.txt")
    if _, err2 := os.Stat(fallbackPath); err2 == nil {
        filePath = fallbackPath
        fallbackName = "agent-stdout.txt"
    }
}
```

## Tests Required

Add 2 tests to `handlers_projects_test.go`:
1. `TestRunFile_OutputMdFallback` — create a temp run dir with agent-stdout.txt but no output.md, verify the response includes fallback field
2. `TestRunFile_OutputMdNoFallback` — create a temp run dir with neither file, verify 404

## Quality Gates

Before creating DONE file:
1. `go build ./...` — must pass
2. `go test ./internal/api/...` — must pass
3. `go test -race ./internal/api/...` — must pass

## Output

Write a summary to output.md describing what you implemented, what files you changed, and the test results.

After quality gates pass, create a DONE file in $JRUN_TASK_FOLDER to signal completion.
