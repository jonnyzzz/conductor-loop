# Agent Backend: Codex

## Overview
Defines how the Go `run-agent` binary invokes the Codex (OpenAI) CLI for a single agent run.

## Goals
- Provide a stable invocation contract for Codex-based runs.
- Capture stdin/stdout/stderr behavior for run artifacts.

## Non-Goals
- Defining prompt content (handled by Runner & Orchestration).
- Managing Codex/OpenAI account setup or billing.

## Invocation (CLI)
- Command (used by run-agent):
  ```bash
  codex exec \
    --dangerously-bypass-approvals-and-sandbox \
    --json \
    -C <working_dir> \
    - \
    < <prompt.md>
  ```
- `exec`: The command to execute.
- `--dangerously-bypass-approvals-and-sandbox`: Disables interactive confirmation and sandboxing (required for autonomous operation).
- `--json`: **CRITICAL**. Forces output in NDJSON (newline-delimited JSON) format. This allows for deterministic parsing of events.
- `-C <working_dir>`: Sets the working directory.
- `-`: Reads the prompt from stdin.

## I/O Contract
- **Input**: Prompt text is piped to the process `stdin`.
- **Output (stdout)**: NDJSON stream (captured to `agent-stdout.txt`).
- **Output (stderr)**: Logs and debug information (captured to `agent-stderr.txt`).
- **Output Processing**:
    - The runner captures stdout.
    - A stream parser (`WriteOutputMDFromStream`) reads the JSON events to extract the final response into `output.md`.

## Environment / Config
- Requires Codex CLI available on PATH.
- Tokens/credentials configured in `config.yaml`:
  ```yaml
  agents:
    - type: codex
      token: "sk-..."                       # Inline token
      # OR:
      token_file: "~/.config/openai/token"  # File path
  ```
- Runner automatically injects token as `OPENAI_API_KEY` environment variable (hardcoded mapping).
- Runner sets working directory and handles all CLI flags automatically.
- No sandboxing; full tool access enabled by runner.

## Version Requirements
- The runner detects the version but specific minimums are generally handled by the `minVersions` map in `internal/runner/validate.go`.

## Implementation Status
- **Status**: Active.
- **Go Package**: `internal/agent/codex`.
- **Type String**: `"codex"`.
