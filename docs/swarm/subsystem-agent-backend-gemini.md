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
- stdout: final response (captured into output.md).
- stderr: progress/logs (captured into agent-stderr.txt).
- exit code: 0 success, non-zero failure.

## Environment / Config
- Requires Gemini CLI available on PATH.
- Tokens/credentials are injected by run-agent from config:
  - Environment variable: `GEMINI_API_KEY`
  - Config key: `gemini_api_key` (in `config.hcl`)
  - Supports @file reference for token file paths (e.g., `gemini_api_key = "@/path/to/key.txt"`)
- Approval mode is set to yolo in current runner scripts (full access).
- Screen reader mode (`--screen-reader true`) may affect output verbosity; streaming behavior needs experimental verification.
- run-agent does not set a model; host CLI defaults/config are used.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
