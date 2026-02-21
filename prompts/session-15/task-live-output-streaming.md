# Task: Add Live Output Streaming for Running Tasks

## Context

The conductor-loop project is at /Users/jonnyzzz/Work/conductor-loop.

The web UI currently loads file content once when a tab is selected (via `loadTabContent()`). For running tasks, the agent is actively writing to files (agent-stdout.txt, output.md). The 5-second auto-refresh helps but is not ideal for monitoring long-running tasks.

**Goal:** Add a streaming/live-tail endpoint that continuously sends new content from a file as it grows. This would enable the web UI to show live output as agents write to their stdout.

## What to Implement

### 1. New SSE Endpoint: `/api/projects/{projectId}/tasks/{taskId}/runs/{runId}/file/stream`

Add a new endpoint in `internal/api/handlers_projects.go` alongside `serveRunFile`:

```go
// serveRunFileStream streams a growing file using SSE (text/event-stream).
// It tails the file from the beginning and sends new content as it arrives.
func (s *Server) serveRunFileStream(w http.ResponseWriter, r *http.Request, run *storage.RunInfo) *apiError {
    name := r.URL.Query().Get("name")
    if name == "" {
        name = "stdout"
    }

    var filePath string
    switch name {
    case "stdout":   filePath = run.StdoutPath
    case "stderr":   filePath = run.StderrPath
    case "prompt":   filePath = run.PromptPath
    case "output.md":
        if run.StdoutPath != "" {
            filePath = filepath.Join(filepath.Dir(run.StdoutPath), "output.md")
        }
    default:
        return apiErrorNotFound("unknown file: " + name)
    }

    if filePath == "" {
        return apiErrorNotFound("file path not set for " + name)
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", 500)
        return nil
    }

    // Stream file content, polling for new data every 500ms
    var offset int64
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-r.Context().Done():
            return nil
        case <-ticker.C:
            f, err := os.Open(filePath)
            if err != nil {
                if os.IsNotExist(err) {
                    continue // file not yet created
                }
                fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
                flusher.Flush()
                return nil
            }

            fi, _ := f.Stat()
            if fi != nil && fi.Size() > offset {
                if _, err := f.Seek(offset, io.SeekStart); err == nil {
                    buf := make([]byte, fi.Size() - offset)
                    n, _ := f.Read(buf)
                    if n > 0 {
                        chunk := string(buf[:n])
                        offset += int64(n)
                        // Escape for SSE: each line needs "data: " prefix
                        for _, line := range strings.Split(chunk, "\n") {
                            fmt.Fprintf(w, "data: %s\n", line)
                        }
                        fmt.Fprintf(w, "\n")
                        flusher.Flush()
                    }
                }
            }
            f.Close()

            // If run is done, send final event and stop
            if run.Status == "completed" || run.Status == "failed" {
                // Check if we've read all the content
                if fi == nil || fi.Size() <= offset {
                    fmt.Fprintf(w, "event: done\ndata: run %s\n\n", run.Status)
                    flusher.Flush()
                    return nil
                }
            }
        }
    }
}
```

### 2. Register the New Endpoint

In `handleProjectTask()` in `internal/api/handlers_projects.go`, add routing for the stream endpoint:

```go
// parts[5] == "file" vs parts[5] == "stream"
if len(parts) >= 6 && parts[5] == "file" {
    // check for /stream sub-path
    if len(parts) >= 7 && parts[6] == "stream" {
        return s.serveRunFileStream(w, r, found)
    }
    return s.serveRunFile(w, r, found)
}
```

Wait - actually the URL structure is:
- `/api/projects/{projectId}/tasks/{taskId}/runs/{runId}/file` → serveRunFile
- `/api/projects/{projectId}/tasks/{taskId}/runs/{runId}/stream` → serveRunFileStream

Use `stream` as a separate sub-path (not nested under `file`), so `parts[5]` can be `"stream"`.

### 3. Update the Web UI

In `web/src/app.js`, add a "LIVE" button or automatically use streaming when the run is active:

In `loadTabContent()`, when the run status is "running" and the tab is "stdout" or "output.md", offer a live streaming option.

Or simply: Add an auto-refresh for running tasks every 2 seconds for the active tab. The current 5-second global refresh is sufficient but you can make the tab content area refresh faster:

```javascript
async function loadTabContent() {
    // ... existing code ...

    // For running tasks: refresh every 2 seconds
    clearTimeout(tabRefreshTimer);
    if (state.activeTab !== 'messages') {
        const run = state.taskRuns.find(r => r.id === state.selectedRun);
        if (run && run.status === 'running') {
            tabRefreshTimer = setTimeout(loadTabContent, 2000);
        }
    }
}
```

Add `let tabRefreshTimer = null;` to the state declarations.

Also clear the timer when the run is deselected/detail closed:
```javascript
function hideRunDetail() {
    clearTimeout(tabRefreshTimer);
    document.getElementById('run-detail').classList.add('hidden');
}
```

### 4. Tests

Add tests for the streaming endpoint in `test/integration/` or `internal/api/`:

- `TestServeRunFileStream_BasicContent`: Verify streaming endpoint returns file content
- `TestServeRunFileStream_UnknownFile`: Verify 404 for unknown file name
- Keep tests focused and simple

## Important Notes

- The streaming endpoint should handle `r.Context().Done()` to clean up when the client disconnects
- For simplicity, the initial implementation can just poll every 500ms rather than using `inotify`/`fsevents`
- The run status re-read is tricky: the handler has a cached `run *storage.RunInfo`. For the streaming endpoint, re-read the run-info.yaml periodically to check if status changed.

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Build must pass
go build ./...

# All tests must pass
go test ./...

# Race detector
go test -race ./internal/api/ ./test/integration/
```

## Files to Change

- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` — add `serveRunFileStream` + routing
- `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js` — add auto-refresh for running tasks
- Tests as appropriate

## Commit Format

```
feat(api): add SSE streaming endpoint for run file tailing
feat(web): auto-refresh tab content for running tasks every 2s
```

## Signal Completion

When done, create the DONE file:
```bash
touch "$TASK_FOLDER/DONE"
```
