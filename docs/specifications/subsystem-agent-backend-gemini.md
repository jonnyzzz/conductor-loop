# Agent Backend: Gemini

## Overview
Defines how the Go `run-agent` binary invokes the Gemini CLI for a single agent run.

## Goals
- Provide a stable invocation contract for Gemini-based runs.
- Capture stdin/stdout/stderr behavior for run artifacts.

## Non-Goals
- Defining prompt content (handled by Runner & Orchestration).
- Managing Gemini account setup or billing.

## Invocation (CLI)
- Command (used by run-agent):
  - `gemini --screen-reader true --approval-mode yolo < <prompt.md>`
- Prompt input is provided via stdin from the run folder prompt file.
- Working directory is set by run-agent based on task/sub-agent context.

## I/O Contract
- stdout: final response (captured to agent-stdout.txt; runner creates output.md from this if missing); streams progressively in chunks (~1s intervals).
- stderr: progress/logs (captured into agent-stderr.txt).
- exit code: 0 success, non-zero failure.
- Streaming behavior: Gemini CLI streams output to stdout progressively (line/block buffered), suitable for real-time UI display.

## Environment / Config
- Requires Gemini CLI available on PATH.
- Tokens/credentials configured in `config.hcl`:
  ```hcl
  agent "gemini" {
    token = "..."                          # Inline token
    # OR: token = "@/path/to/token.txt"     # @file reference (preferred)
    # OR: token_file = "~/.config/gemini/token"  # File-based field (alternative)
  }
  ```
- Runner automatically injects token as `GEMINI_API_KEY` environment variable (hardcoded mapping).
- Runner sets working directory and handles all CLI flags automatically (hardcoded: `--screen-reader true --approval-mode yolo`).
- Screen reader mode provides detailed output and works with streaming (verified via experiments).
- Model selection uses CLI defaults (not overridden by runner).
- When config is loaded, run-agent validates agent types and rejects unknown backends.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
