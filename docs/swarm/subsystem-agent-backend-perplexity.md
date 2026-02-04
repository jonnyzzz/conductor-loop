# Agent Backend: Perplexity

## Overview
Defines the native Perplexity backend integration for run-agent. Perplexity is treated as an agent type and invoked via REST API (not CLI wrappers).

## Goals
- Provide a stable adapter for Perplexity-backed agent runs.
- Align prompt/input/output handling with other agent backends.

## Non-Goals
- Defining Perplexity account setup or billing.
- Providing advanced vendor-specific features beyond text completion.

## Invocation (REST)
- run-agent sends a prompt to the Perplexity API using an API key from config.hcl (inline or @file reference).
- The adapter writes the final response to BOTH stdout (captured to agent-stdout.txt) AND output.md for UI consistency.
- Adapter should emit periodic progress logs to stdout (or stream tokens if supported) to avoid idle/stuck detection.
- Keep-alive mechanism: If streaming is not supported, emit "[Perplexity] Generating..." messages every 30 seconds to prevent stuck detection.
- Citations: Perplexity citations should be included inline in the response text if supported by the API.

## I/O Contract
- Input: prompt text (from prompt.md).
- Output: plain text response written to BOTH:
  - output.md (canonical result file for UI display)
  - stdout (captured to agent-stdout.txt for streaming progress logs)
- Errors: logged to agent-stderr.txt; non-zero exit code for failures.
- Progress logs: adapter emits "[Perplexity] <status>" messages to stdout during generation.
- Keep-alive: If API doesn't support streaming, emit progress every 30s to prevent stuck detection.

## Environment / Config
- API key is stored in config.hcl and injected into the backend client (not exposed to agents for workflow use).
- Backend selection uses the same round-robin/weights as other agent types.
- Config includes a Perplexity section; use the most capable model by default unless overridden.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
