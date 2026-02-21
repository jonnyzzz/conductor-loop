# Agent Backend: Codex

## Overview
Defines how the Go `run-agent` binary invokes the Codex CLI for a single agent run.

## Goals
- Provide a stable invocation contract for Codex-based runs.
- Capture stdin/stdout/stderr behavior for run artifacts.

## Non-Goals
- Defining prompt content (handled by Runner & Orchestration).
- Managing Codex account setup or billing.

## Invocation (CLI)
- Command (used by run-agent):
  - `codex exec --dangerously-bypass-approvals-and-sandbox -C <cwd> - < <prompt.md>`
- Prompt input is provided via stdin from the run folder prompt file.
- Working directory is set by run-agent based on task/sub-agent context.

## I/O Contract
- stdout: final response (captured to agent-stdout.txt; runner creates output.md from this if missing).
- stderr: progress/logs (captured to agent-stderr.txt).
- exit code: 0 success, non-zero failure.
- Streaming behavior: CLI output assumed to stream progressively (standard CLI behavior, similar to verified Gemini behavior).

## Environment / Config
- Requires Codex CLI available on PATH.
- Tokens/credentials configured in `config.hcl`:
  ```hcl
  agent "codex" {
    token_file = "~/.config/openai/token"  # Recommended: file-based
    # OR: token = "sk-..."                 # Inline value
  }
  ```
- Runner automatically injects token as `OPENAI_API_KEY` environment variable (hardcoded mapping).
- Runner sets working directory and handles all CLI flags automatically (hardcoded for unrestricted mode).
- No sandboxing; full tool access enabled by runner.
- Model/reasoning settings use CLI defaults (not overridden by runner).

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
