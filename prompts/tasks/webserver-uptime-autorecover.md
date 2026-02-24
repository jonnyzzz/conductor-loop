# Task: Webserver Uptime Auto-Recovery (Watchdog Restart + Health Probes)

## Context

- **ID**: `task-20260223-155250-webserver-uptime-autorecover`
- **Priority**: P0
- **Source**: `docs/dev/todos.md`; observed in conductor production runs

The `run-agent serve` web server process has been observed to stop responding without
process exit — the binary remains alive but the HTTP port becomes unreachable. This manifests
as "webserver is no longer up" errors in monitoring output.

Current behavior:
- No health probe: the server has no `/health` or `/readyz` endpoint
- No watchdog: if the HTTP listener silently fails, no restart occurs
- No failure reason logging: the crash mode is silent

## Requirements

1. **Health probe endpoint**: Add `GET /healthz` (returns `200 OK` + `{"status":"ok","uptime":"<duration>"}`)
   available without authentication, low-latency.

2. **Internal watchdog goroutine**: Start a lightweight goroutine on server startup that:
   - Pings `http://127.0.0.1:<port>/healthz` every 30s
   - After 3 consecutive failures, logs `"WARN: server health probe failed N times"` and
     attempts to restart the HTTP listener (re-bind on same port)
   - After 10 consecutive failures, exits with a clear error and non-zero exit code

3. **Failure reason logging**: On any unhandled server error, log the error type and stack
   before exiting so operators can diagnose root cause.

4. **CLI flag**: `--watchdog-interval <duration>` (default: `30s`), `--watchdog-max-failures <int>` (default: `3`)
   to make watchdog behavior configurable without code changes.

5. **Tests**: Unit test for health endpoint; integration test that simulates listener failure
   and verifies watchdog logs the right messages.

## Acceptance Criteria

- `GET http://localhost:<port>/healthz` returns `200` with JSON body.
- If listener fails (simulated via close of listener), watchdog logs warning within 2x interval.
- After max failures, process exits with non-zero code and error message in stderr.
- `go test ./internal/api ./cmd/run-agent -run 'TestHealth|TestWatchdog' -count=1` passes.
- `go build ./...` passes.

## Verification

```bash
go build -o bin/run-agent ./cmd/run-agent

# Start server and verify health endpoint
./bin/run-agent serve --host 127.0.0.1 --port 18080 --root runs &
sleep 2
curl -s http://127.0.0.1:18080/healthz
# Expected: {"status":"ok","uptime":"..."}

# Unit tests
go test ./internal/api -run 'TestHealth' -count=1

# Kill with SIGINT to verify clean shutdown
kill -INT <pid>
```

## Reference Files

- `cmd/run-agent/serve.go` — serve command entrypoint
- `internal/api/server.go` — HTTP server setup
- `internal/api/routes.go` — route registration (add `/healthz`)
- `docs/dev/todos.md` — feature request origin
