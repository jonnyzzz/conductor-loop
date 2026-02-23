# Adding New Agent Backends

This guide explains where to plug a new backend into the current runner implementation.

## Current Backend Topology

Runner dispatch logic is in `internal/runner/job.go`.

- CLI agents (`isRestAgent() == false`):
  - `claude`
  - `codex`
  - `gemini`
- REST agents (`isRestAgent() == true`):
  - `perplexity`
  - `xai`

Important correction: there is no active `internal/agent/factory.go` registry in this codebase today.
Integration is done through runner switches (`commandForAgent`, `isRestAgent`, `executeREST`, validation helpers).

## Exact CLI Commands and Flags

From `commandForAgent()` (`internal/runner/job.go`):

- `codex`
  - command: `codex`
  - args: `exec --dangerously-bypass-approvals-and-sandbox --json -`
- `claude`
  - command: `claude`
  - args: `-p --input-format text --output-format stream-json --verbose --tools default --permission-mode bypassPermissions`
- `gemini`
  - command: `gemini`
  - args: `--screen-reader true --approval-mode yolo --output-format stream-json`

### Gemini clarification

`internal/agent/gemini/gemini.go` contains a REST implementation, but the main runner path currently executes Gemini via CLI (`isRestAgent("gemini") == false`).

## REST Backends

`executeREST()` in `internal/runner/job.go` currently instantiates:

- `perplexity.NewPerplexityAgent(...)`
- `xai.NewAgent(...)`

### xAI status

- Historical docs called xAI deferred/placeholder.
- Current code has an implemented REST backend in `internal/agent/xai/` and runner integration in `executeREST()` and `isRestAgent()`.

## Token Environment Mapping

Hardcoded mapping in `tokenEnvVar()` (`internal/runner/orchestrator.go`):

- `claude` -> `ANTHROPIC_API_KEY`
- `codex` -> `OPENAI_API_KEY`
- `gemini` -> `GEMINI_API_KEY`
- `perplexity` -> `PERPLEXITY_API_KEY`
- `xai` -> `XAI_API_KEY`

## Config Shape for Backends

`internal/config/config.go` uses generic `AgentConfig` fields:

- `type`
- `token`
- `token_file`
- `base_url`
- `model`

Both YAML and HCL configs are supported by loader logic.

## How to Add a New CLI Agent

1. Add/extend backend package under `internal/agent/<name>/` (optional but recommended for parser/env helpers).
2. Update `commandForAgent()` in `internal/runner/job.go` with command + args.
3. Keep `isRestAgent()` returning `false` for this type.
4. Update `cliCommand()` and related validation in `internal/runner/validate.go`.
5. Add token mapping in `tokenEnvVar()` if needed.
6. If output is structured stream JSON/NDJSON, add a parser similar to:
   - `internal/agent/claude/stream_parser.go`
   - `internal/agent/codex/stream_parser.go`
   - `internal/agent/gemini/stream_parser.go`
7. Add tests in runner and backend packages.

## How to Add a New REST Agent

1. Create backend package in `internal/agent/<name>/` implementing `agent.Agent`.
2. Update `isRestAgent()` in `internal/runner/job.go` to include new type.
3. Update `executeREST()` switch in `internal/runner/job.go` to construct and run backend.
4. Add token mapping in `tokenEnvVar()`.
5. Update validation paths in `internal/runner/validate.go` if needed.
6. Add tests for:
   - backend execution and error handling
   - runner dispatch path
   - token/env behavior

## Required Behavioral Contract

Regardless of CLI or REST path, backend execution must preserve:

- stdout to `agent-stdout.txt`
- stderr to `agent-stderr.txt`
- `output.md` fallback support via runner (`agent.CreateOutputMD`)
- run lifecycle events (`RUN_START`, `RUN_STOP`, `RUN_CRASH`)
- `run-info.yaml` status transitions (`running` -> `completed`/`failed`)

## Validation Checklist

Before merging a new backend:

1. `go build ./...`
2. `go test ./...`
3. Verify runner dispatch for the new type (CLI vs REST).
4. Verify env injection and token mapping.
5. Verify `output.md` exists even when backend does not write it directly.
6. Verify message-bus lifecycle events are emitted.

---

Last updated: 2026-02-23
