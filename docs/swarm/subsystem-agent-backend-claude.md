# Agent Backend: Claude Code

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
  - `claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < <prompt.md>`
- Prompt input is provided via stdin from the run folder prompt file.
- Working directory is set by run-agent based on task/sub-agent context.

## I/O Contract
- stdout: final response (captured to agent-stdout.txt; runner creates output.md from this if missing).
- stderr: progress/logs (captured to agent-stderr.txt).
- exit code: 0 success, non-zero failure.
- Streaming behavior: CLI output assumed to stream progressively (standard CLI behavior, similar to verified Gemini behavior).

## Environment / Config
- Requires Claude CLI available on PATH.
- Tokens/credentials configured in `config.hcl`:
  ```hcl
  agent "claude" {
    token_file = "~/.config/claude/token"  # Recommended: file-based
    # OR: token = "sk-ant-..."              # Inline value
  }
  ```
- Runner automatically injects token as `ANTHROPIC_API_KEY` environment variable (hardcoded mapping).
- Runner sets working directory and handles all CLI flags automatically (hardcoded: `--tools default --permission-mode bypassPermissions`).
- Model selection uses CLI defaults (not overridden by runner).

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
