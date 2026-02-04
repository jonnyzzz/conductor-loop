# Decision: SSE Stream Run Discovery via Polling

## Context
The backend (`run-agent serve`) streams logs to the UI via SSE. It must detect new run directories (`runs/<run_id>/`) created by the runner (`run-agent job`) to include their logs in the stream automatically.

## Decision
**Mechanism:** **Polling** (Interval-based Directory Scanning)
**Interval:** 1 second

## Rationale
While filesystem watchers (`fsnotify`/`inotify`) offer event-driven updates, they introduce significant cross-platform complexity (especially recursively) and can be unreliable for directory creation events on some OS/filesystem combinations (e.g., Docker mounts, network shares).

Polling runs directories is:
1.  **Robust:** Self-healing; doesn't break if a "create" event is missed.
2.  **Simple:** Uses standard `os.ReadDir`, reducing dependencies and "watcher hell".
3.  **Predictable:** Constant overhead regardless of OS.
4.  **Sufficient:** Run creation is a rare event (seconds/minutes). A 1s latency for detecting a *new* run is acceptable for a human-facing UI. Log content within the run will still be tailed in real-time once discovered.

## Algorithm

### Data Structures
- `activeRuns`: Map<RunID, CancelFunc> (Tracks currently streaming runs)
- `streamChan`: Channel for merging log lines from all runs

### Loop Logic
1.  **Start:** Launch `DiscoveryLoop` goroutine.
2.  **Ticker:** Fires every 1.0 seconds.
3.  **Scan:**
    - Call `os.ReadDir(runs_directory)`.
    - Filter entries: Must be IsDir() and match `run_id` pattern (`YYYYMMDD-HHMMSSMMMM-PID`).
4.  **Diff:**
    - Identify `newRuns`: Present in Scan but not in `activeRuns`.
    - (Optional) Identify `staleRuns`: In `activeRuns` but removed from disk (if cleanup is implemented).
5.  **Act:**
    - For each `newRun`:
        - Create a sub-context/cancellation.
        - Start `go streamRunLogs(ctx, runPath, streamChan)`.
        - Add to `activeRuns`.
6.  **Stream (per run):**
    - The `streamRunLogs` function uses `nxadm/tail` (or similar) on `stdout.txt` / `stderr.txt`.
    - Pushes formatted events (`{run_id, stream, line}`) to `streamChan`.

## Mid-Stream Handling
- The SSE endpoint listens to `streamChan`.
- When `DiscoveryLoop` finds a new run, it spawns a worker that starts pouring data into `streamChan`.
- The client receives `event: run_start` (synthesized from reading `run-info.yaml`) followed immediately by log lines.
- No client-side reconnection required.

## Performance Implications
- **CPU:** `os.ReadDir` on a directory with <10,000 items is negligible (microseconds).
- **IO:** Minimal. Only metadata is read. Content reading is handled by efficient tailing.
- **Latency:** Max 1s delay between run creation and UI appearance.
- **Concurrency:** One goroutine per active run + 1 discovery goroutine. Efficient for anticipated scale (<50 concurrent runs).

## Cross-Platform Compatibility
- **Linux/macOS/Windows:** `os.ReadDir` is part of the Go standard library and works identically on all supported platforms.
- **Filesystems:** Works reliably on local disks, NFS, SMB, and FUSE mounts where `inotify` might fail.

## Implementation Details
- **Location:** `subsystem-runner-orchestration` (Runner) owns the directory structure, but `subsystem-frontend-backend-api` (Server) owns the streaming logic. This logic belongs in the **Server** component.
- **Safety:** Ignore errors during `ReadDir` (transient locks). Ignore partial writes (check for `run-info.yaml` existence before starting stream to ensure run is initialized).
