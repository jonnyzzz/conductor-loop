# Agent Protocol Specification

This document describes the runtime contract between the runner and agent backends.

## Overview

All backends implement the `Agent` interface in `internal/agent/agent.go`:

```go
type Agent interface {
    Execute(ctx context.Context, runCtx *RunContext) error
    Type() string
}
```

`RunContext` fields:

```go
type RunContext struct {
    RunID       string
    ProjectID   string
    TaskID      string
    Prompt      string
    WorkingDir  string
    StdoutPath  string
    StderrPath  string
    Environment map[string]string
}
```

## Run ID and File Layout

- Run IDs are generated as: `YYYYMMDD-HHMMSSMMMM-<pid>-<seq>` (`internal/runner/orchestrator.go`).
- Per-run files are created under `<task>/runs/<run_id>/`.
- Runner-managed output files:
  - `prompt.md`
  - `agent-stdout.txt`
  - `agent-stderr.txt`
  - `output.md`
  - `run-info.yaml`

## Environment Contract

Runner injects these variables into the agent subprocess (`internal/runner/job.go`, `internal/runner/wrap.go`):

- `JRUN_PROJECT_ID`
- `JRUN_TASK_ID`
- `JRUN_ID`
- `JRUN_PARENT_ID` (may be empty for root runs)
- `JRUN_RUNS_DIR`
- `JRUN_MESSAGE_BUS`
- `JRUN_TASK_FOLDER`
- `JRUN_RUN_FOLDER`
- `JRUN_CONDUCTOR_URL` (when available)

Token mapping is hardcoded (`internal/runner/orchestrator.go`):

- `claude` -> `ANTHROPIC_API_KEY`
- `codex` -> `OPENAI_API_KEY`
- `gemini` -> `GEMINI_API_KEY`
- `perplexity` -> `PERPLEXITY_API_KEY`
- `xai` -> `XAI_API_KEY`

## Execution Lifecycle

For each run (`internal/runner/job.go`):

1. Create run dir and `run-info.yaml` in `running` state.
2. Write `prompt.md` with the runner preamble.
3. Inject environment variables and token env var.
4. Execute backend (CLI or REST path).
5. Ensure `output.md` exists (see fallback behavior below).
6. Update `run-info.yaml` with final status.
7. Post lifecycle message-bus event.

### Lifecycle Events

Runner posts these message types (`internal/messagebus/messagebus.go`):

- `RUN_START`
- `RUN_STOP`
- `RUN_CRASH`

Behavior:

- `RUN_START`: posted when the run begins.
- `RUN_STOP`: posted on successful completion.
- `RUN_CRASH`: posted on failures / non-zero exits.

## CLI vs REST Execution

Runner dispatch is in `internal/runner/job.go`:

- CLI agents: `claude`, `codex`, `gemini`
- REST agents: `perplexity`, `xai`

`isRestAgent()` currently returns `true` only for `perplexity` and `xai`.

## Stdio and output.md Contract

- Agent stdout is captured to `agent-stdout.txt`.
- Agent stderr is captured to `agent-stderr.txt`.
- `output.md` is required as run summary output.

### output.md fallback behavior

If `output.md` is missing, runner calls `agent.CreateOutputMD(runDir, "")` (`internal/agent/executor.go`), which copies `agent-stdout.txt` to `output.md`.

This fallback is used in both:

- normal `run-agent job` execution (`internal/runner/job.go`)
- wrapped execution (`internal/runner/wrap.go`)

## Exit and Status Semantics

Run statuses in `run-info.yaml` (`internal/storage/runinfo.go`):

- `running`
- `completed`
- `failed`

Typical mapping:

- exit code `0` -> `completed`
- non-zero / execution error -> `failed`

## Notes for Backend Authors

If you add a backend, see `docs/dev/adding-agents.md` for required runner integration points (`commandForAgent`, `isRestAgent`, `executeREST`, token mapping, validation).

---

Last updated: 2026-02-23
