# FACTS: Reconciled Truths (2026-02-24)

This file captures reconciliations of contradictory facts found across `docs/facts/`.
When in doubt, prioritizing **Code > This File > Other FACTS files**.

---

## Port Configuration
[2026-02-24 07:45:00] [tags: reconciliation, port]
*   **Canonical Default:** `14355` (Source: `cmd/run-agent/serve.go`, `cmd/run-agent/server.go`)
*   **Known Drift:** `bin/conductor --help` reports default `8080`. This is an artifact drift.
*   **Action:** Use `14355` as the standard documentation port.

## Host Binding
[2026-02-24 07:55:00] [tags: reconciliation, host]
*   **Canonical Default:** `0.0.0.0` (Source: `cmd/run-agent/serve.go`)
*   **Known Drift:** `FACTS-architecture.md` claims `127.0.0.1`. This is outdated.

## Command Availability
[2026-02-24 07:45:00] [tags: reconciliation, cli]
*   **`run-agent iterate`:** Command is `unknown` in the current `bin/run-agent` binary, despite previous task logs claiming implementation.
*   **Action:** Treat `iterate` as unavailable/experimental until binary is updated.

## Timestamps & IDs
[2026-02-24 07:45:00] [tags: reconciliation, naming]
*   **Task ID:** `task-<YYYYMMDD>-<HHMMSS>-<slug>` (Seconds precision).
*   **Run ID:** `YYYYMMDD-HHMMSSmmmm-<pid>` (Millisecond/Nanosecond precision).
*   **Correction:** `FACTS-runner-storage.md` claim of identical precision was imprecise for Task IDs.

## Configuration Path
[2026-02-24 07:45:00] [tags: reconciliation, config]
*   **Canonical:** `~/.config/conductor/` (Standard XDG-like path) or `./config.*`.
*   **Legacy/Drift:** `~/.conductor/` found in some user docs is outdated.

## Runtime Versions
[2026-02-24 07:45:00] [tags: reconciliation, env]
*   **Go:** `1.24.0` (Source: `go.mod`).
*   **Legacy:** Docs mentioning `1.21+` are outdated.

