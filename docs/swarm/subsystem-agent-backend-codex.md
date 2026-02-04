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
- stdout: final response (captured into output.md).
- stderr: progress/logs (captured into agent-stderr.txt).
- exit code: 0 success, non-zero failure.

## Environment / Config
- Requires Codex CLI available on PATH.
- Tokens/credentials are injected by run-agent from config:
  - Environment variable: `OPENAI_API_KEY`
  - Config key: `openai_api_key` (in `config.hcl`)
  - Supports @file reference for token file paths (e.g., `openai_api_key = "@/path/to/key.txt"`)
- No sandboxing enforced; full tool access is required (keep `--dangerously-bypass-approvals-and-sandbox`).
- run-agent does not override model/reasoning settings; host CLI defaults apply.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
