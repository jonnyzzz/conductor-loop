# FACTS: Reconciled Truths (2026-02-24)

This file captures reconciliations of contradictory facts found across `docs/facts/`.
**Prioritization Order:** Code > This File > Other FACTS files.

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
*   **Run ID:** `YYYYMMDD-HHMMSS0000-<pid>-<seq>` (Seconds precision in timestamp with literal `0000` suffix, uniqueness via PID and process-local atomic sequence).
*   **Correction:** Earlier claims of millisecond/nanosecond precision in the timestamp part were imprecise. Implementation uses `20060102-1504050000` format string.

## Configuration Path & Format
[2026-02-24 07:45:00] [tags: reconciliation, config]
*   **Canonical Path:** `~/.config/conductor/` (Standard XDG-like path) or `./config.*`.
*   **Legacy/Drift:** `~/.conductor/` found in some user docs is outdated.
*   **Format Priority:** YAML (`config.yaml`, `config.yml`) takes precedence over HCL (`config.hcl`). HCL is supported but secondary.

## Runtime Versions
[2026-02-24 07:45:00] [tags: reconciliation, env]
*   **Go:** `1.24.0` (Source: `go.mod`).
*   **Legacy:** Docs mentioning `1.21+` are outdated.

## Documentation Structure
[2026-02-24 08:30:00] [tags: reconciliation, workflow, prompts]
*   **Workflow Prompts:** `THE_PROMPT_v5*.md` and `THE_PLAN_v5.md` are located in `docs/workflow/`.
*   **Legacy References:** CLI code may still reference root paths (drift).
*   **User Docs:** `docs/dev/` contains developer guides; `docs/workflow/` contains orchestration prompts.

## Agent Backends
[2026-02-24 09:45:00] [tags: reconciliation, agents]
*   **Gemini:** Runner uses CLI implementation (`gemini` command). A REST implementation exists in code (`internal/agent/gemini/gemini.go`) but is **unused** by the runner.
*   **xAI:** Implemented as a REST adapter (`internal/agent/xai`).
*   **Perplexity:** Implemented as a REST adapter.
*   **Claude/Codex:** Implemented as CLI wrappers.

## Message Bus Schema
[2026-02-24 09:45:00] [tags: reconciliation, messagebus]
*   **Attachments:** The `Message` struct in Go does NOT have `attachment_path` or `attachments` fields. `FACTS-runner-storage.md` claim is drift.
*   **Links:** `parents` field supports linking. `Links` field exists for external references.

## Architecture
[2026-02-24 09:45:00] [tags: reconciliation, architecture]
*   **Subsystems:** 16 verified subsystems documented in `docs/dev/subsystems.md`, superseding the original 8.
*   **Subsystem List:** Storage, Config, Message Bus, Agent Protocol, Agent Backends, Runner Orchestration, API Server, Frontend UI, Webhook, CLI (List, Output, Watch), API (Delete Run, Task), UI (Search, Stats).

## Message Type Defaults
[2026-02-24 10:05:00] [tags: reconciliation, messagebus]
*   **CLI Default:** `INFO` (Source: `cmd/run-agent/bus.go`)
*   **API Default:** `USER` (Source: `internal/api/handlers.go`)
*   **Action:** Note the inconsistency; prefer explicit `--type` in automation.

[2026-02-24 10:05:00] [tags: reconciliation, status]
*   **Confirmed:** `FACTS-architecture.md` (host binding), `FACTS-user-docs.md` (Go version, config path, iterate cmd), and `FACTS-runner-storage.md` (ID precision and literal 0000 suffix) have been updated to reflect these reconciled truths.
