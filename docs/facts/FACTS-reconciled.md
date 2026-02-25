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
*   **Run ID:** `YYYYMMDD-HHMMSS0000-<pid>-<seq>` (Seconds precision in timestamp with literal `0000` suffix, uniqueness via PID and process-local atomic sequence).
*   **Correction:** Earlier claims of millisecond/nanosecond precision in the timestamp part were imprecise. Implementation uses `20060102-1504050000` format string.

## Configuration Path & Format
[2026-02-24 07:45:00] [tags: reconciliation, config]
*   **Canonical User Home Config:** `~/.run-agent/conductor-loop.hcl`
    - Directory: `~/.run-agent/` (also holds `binaries/` for versioned binary cache)
    - **Auto-created on first run** with commented template pointing to GitHub docs
    - Agent type inferred from HCL block name — no `type` field required
    - File permissions: `0600`; directory: `0700`
*   **Discovery order** (when `--config` / `CONDUCTOR_CONFIG` not set):
    1. `./config.yaml`
    2. `./config.yml`
    3. `~/.run-agent/conductor-loop.hcl`
*   **Dropped paths** (no longer in discovery): `~/.config/conductor/`, `~/.conductor/`
*   **Project-local `.hcl` files are NOT auto-discovered** — pass via `--config` explicitly.
*   **Format:** YAML (`config.yaml`) for project-level config; HCL for user home config.
*   **Documentation:** `docs/user/configuration.md` is the authoritative reference.
    Direct link: `https://github.com/jonnyzzz/conductor-loop/blob/main/docs/user/configuration.md`

## Binary Cache Directory
[2026-02-24] [tags: binary, install, deploy]
*   **Location:** `~/.run-agent/binaries/`
*   **Structure:** `~/.run-agent/binaries/<version>/run-agent` (versioned), `~/.run-agent/binaries/_latest/run-agent` (symlink to current version)
*   **Managed by:** `scripts/deploy_locally.sh` (local dev builds) and `scripts/fetch_release.sh` (future GitHub release downloads)
*   **`run-agent.cmd` resolution order:** `$RUN_AGENT_BIN` env → `~/.run-agent/binaries/_latest/run-agent` → script-sibling `run-agent` → `dist/run-agent-<os>-<arch>` → PATH

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

## Perplexity Default Model
[2026-02-24] [tags: reconciliation, perplexity, agents]
*   **`sonar-reasoning` is deprecated** by Perplexity AI and returns HTTP 400 (`invalid_model`).
*   **Recommended model:** `sonar-pro` — set via `model` field in config:
    ```hcl
    perplexity {
      token_file = "~/.perplexity"
      model      = "sonar-pro"
    }
    ```
*   Hard-coded default in `internal/agent/perplexity/perplexity.go` (`defaultPerplexityModel`) still says `sonar-reasoning` and needs updating in a follow-up.
*   **Workaround:** Always set `model = "sonar-pro"` (or another current model) in user config.

## HCL Config: `defaults.timeout` Is Required
[2026-02-24] [tags: reconciliation, config, hcl]
*   Validation enforces `defaults.timeout > 0` — the HCL home config **must** include a `defaults` block with a positive `timeout`.
*   A minimal `conductor-loop.hcl` without this block will fail to load with: `load config: defaults.timeout must be positive`.
*   The auto-generated starter template already includes an example `defaults` block (commented out). Users must uncomment and set a value.

## Test Suite: `test/local/`
[2026-02-24] [tags: testing, local, binary]
*   New package `test/local/` holds binary-path integration tests.
*   Invokes `bin/run-agent job` directly (not Go packages) for end-to-end coverage.
*   All four agents (claude, codex, gemini, perplexity) covered in a parallel table-driven test.
*   Gated by `RUN_LOCAL_AGENT_TESTS=1`; skip guard prevents accidental CI execution.
*   Run: `RUN_LOCAL_AGENT_TESTS=1 go test ./test/local/... -v -timeout 15m`
*   Optional fixed output dir: `RUN_LOCAL_ROOT=/tmp/my-runs` (overrides `t.TempDir()`).

## Runs Root Directory (Canonical Path)
[2026-02-25] [tags: reconciliation, storage, runs-dir]
*   **Canonical default:** `~/.run-agent/runs`
*   **Override mechanism:** `storage.runs_dir` field in `~/.run-agent/conductor-loop.hcl`:
    ```hcl
    storage {
      runs_dir = "~/my-custom-runs"
    }
    ```
*   **No environment variable override:** `RUNS_DIR` env var has been removed from all subcommands. There is no other supported way to change the runs directory outside the config file.
*   **`--root` flag:** retained on CLI commands as an explicit override for tests and development only; not intended for production use.
*   **Shared logic:** All commands (`list`, `status`, `watch`, `gc`, `output`, `bus`, `facts`, `task`, `job`, monitor, etc.) call `config.ResolveRunsDir(explicit)` — the same function — to resolve the path.
*   **Config error = hard fail:** if the config file exists but is malformed, all commands fail immediately with an error. No silent fallback to a wrong directory.
*   **Implementation:** `internal/config/runs_dir.go` — `ResolveRunsDir(explicit string) (string, error)`

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

## Run Status Summary (2026-02-23)
[2026-02-23 20:29:55] [tags: runs, summary]
*   **Total Tasks Processed:** 125
*   **Completed:** 100
*   **Blocked:** 12
*   **Open:** 13
*   **Source:** `docs/facts/FACTS-runs-jonnyzzz.md`

## Issue Count Summary (2026-02-23)
[2026-02-23 19:19:41] [tags: issues, summary]
*   **Total Issues:** 22 (ISSUE-000 to ISSUE-021)
*   **Resolved:** 19
*   **Partially Resolved:** 3 (ISSUE-002, ISSUE-003, ISSUE-009)
*   **Open:** 0
*   **Source:** `docs/facts/FACTS-issues-decisions.md`

[2026-02-24 10:05:00] [tags: reconciliation, status]
*   **Confirmed:** `FACTS-architecture.md` (host binding), `FACTS-user-docs.md` (Go version, config path, iterate cmd), and `FACTS-runner-storage.md` (ID precision and literal 0000 suffix) have been updated to reflect these reconciled truths.

[2026-02-24 10:30:00] [tags: reconciliation, docs, corrections]
*   **Architecture Docs:** `docs/dev/architecture.md` updated to correct Run ID format (seconds + 0000 suffix).
*   **Subsystems Docs:** `docs/dev/subsystems.md` updated to correctly describe Gemini as a CLI-based backend, matching implementation reality.
