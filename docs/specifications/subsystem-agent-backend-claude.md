# Agent Backend: Claude

## Overview
Defines how the Go `run-agent` binary invokes the Claude CLI for a single agent run.

## Goals
- Provide a stable invocation contract for Claude-based runs.
- Capture stdin/stdout/stderr behavior for run artifacts.

## Non-Goals
- Defining prompt content (handled by Runner & Orchestration).
- Managing Claude account setup or billing.

## Invocation (CLI)
- Command (used by run-agent):
  ```bash
  claude -C <working_dir> \
    -p \
    --input-format text \
    --output-format stream-json \
    --verbose \
    --tools default \
    --permission-mode bypassPermissions \
    < <prompt.md>
  ```
- `-C <working_dir>`: Sets the working directory for the agent process.
- `-p`: Prompt mode (reads from stdin).
- `--input-format text`: Specifies the input is plain text.
- `--output-format stream-json`: **CRITICAL**. Forces output as a stream of JSON events. This is required for reliable parsing of the response text separate from tool usage or logs.
- `--verbose`: Enables verbose logging (captured in stderr).
- `--tools default`: Enables default tools (e.g., file system access).
- `--permission-mode bypassPermissions`: Bypasses user confirmation prompts for tool execution (required for autonomous operation).

## I/O Contract
- **Input**: Prompt text is piped to the process `stdin`.
- **Output (stdout)**: JSON stream events (captured to `agent-stdout.txt`).
- **Output (stderr)**: Logs and debug information (captured to `agent-stderr.txt`).
- **Output Processing**:
    - The runner captures stdout.
    - A stream parser (`ParseStreamJSON` in `stream_parser.go`) reads the JSON events, extracts the final textual response, and writes it to `output.md` in the run directory.
    - This ensures `output.md` contains only the model's answer, cleaner than raw stdout.

## Environment / Config
- Requires Claude CLI available on PATH.
- Tokens/credentials configured in `config.yaml`:
  ```yaml
  agents:
    - type: claude
      token: "sk-ant-..."                   # Inline token
      # OR:
      token_file: "~/.config/claude/token"  # File path
  ```
- Runner automatically injects token as `ANTHROPIC_API_KEY` environment variable (hardcoded mapping).
- Runner sets working directory and handles all CLI flags automatically.
- Model selection uses CLI defaults (not overridden by runner).

## Version Requirements
- The runner attempts to detect the `claude` version via `claude --version`.
- Minimum version requirements may be enforced by `internal/runner/validate.go`.

## Implementation Status
- **Status**: Active / Production.
- **Go Package**: `internal/agent/claude`.
- **Type String**: `"claude"`.
