# Agent Backend: Gemini

## Overview
Defines how the Go `run-agent` binary invokes the Gemini CLI for a single agent run.

**Important Implementation Detail**:
While the codebase contains a REST API client in `internal/agent/gemini/gemini.go`, the **runner currently uses the CLI implementation** defined in `internal/runner/job.go`. The REST client is currently unused for the main execution loop.

## Goals
- Provide a stable invocation contract for Gemini-based runs.
- Capture stdin/stdout/stderr behavior for run artifacts.

## Invocation (CLI)
- Command (used by run-agent):
  ```bash
  gemini \
    --screen-reader true \
    --approval-mode yolo \
    --output-format stream-json \
    < <prompt.md>
  ```
- `--screen-reader true`: Enables detailed/streaming output suitable for screen readers (and automation).
- `--approval-mode yolo`: Auto-approves all actions/prompts (unsafe mode for automation).
- `--output-format stream-json`: **CRITICAL**. Requests output as a stream of JSON events. This is a recent addition to support deterministic parsing.
    - *Note*: The runner may need a fallback for older CLI versions that do not support this flag.

## I/O Contract
- **Input**: Prompt text is piped to `stdin`.
- **Output (stdout)**: JSON stream (when `--output-format stream-json` is used). Captured to `agent-stdout.txt`.
- **Output (stderr)**: Logs (captured to `agent-stderr.txt`).
- **Output Processing**:
    - The runner uses `gemini.WriteOutputMDFromStream` (from `internal/agent/gemini/stream_parser.go`) to parse the output and generate `output.md`.

## Environment / Config
- Requires Gemini CLI available on PATH.
- Tokens/credentials configured in `config.yaml`:
  ```yaml
  agents:
    - type: gemini
      token: "..."                          # Inline token
      # OR:
      token_file: "~/.config/gemini/token"  # File path
  ```
- Runner automatically injects token as `GEMINI_API_KEY` environment variable (hardcoded mapping).
- Runner sets working directory and handles all CLI flags automatically.

## Version Requirements
- Runner detects version via `gemini --version`.
- Requires a version compatible with the flags used.

## Implementation Status
- **Status**: Active / Production.
- **Go Package**: `internal/agent/gemini` (contains helper functions and unused REST client).
- **Runner Logic**: `internal/runner/job.go` (CLI construction).
- **Type String**: `"gemini"`.
