# Agent Backend: Claude Code

## Overview
Defines how run-agent invokes the Claude CLI for a single agent run.

## Goals
- Provide a stable invocation contract for Claude-based runs.
- Capture stdin/stdout/stderr behavior for run artifacts.

## Non-Goals
- Defining prompt content (handled by Runner & Orchestration).
- Managing Claude account setup or billing.

## Invocation (CLI)
- Command (current run-agent.sh):
  - `claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < <prompt.md>`
- Prompt input is provided via stdin from the run folder prompt file.
- Working directory is set by run-agent based on task/sub-agent context.

## I/O Contract
- stdout: final response (captured into output.md).
- stderr: progress/logs (captured into agent-stderr.txt).
- exit code: 0 success, non-zero failure.

## Environment / Config
- Requires Claude CLI available on PATH.
- Tokens/credentials are injected by run-agent from config (backend-specific names).
- Tool access is enabled via `--tools default` and bypass permissions mode.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
