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
- The adapter writes the final response to output.md and logs to agent-stderr.txt.
- Adapter should emit periodic progress (or stream tokens if supported) to avoid idle/stuck detection.

## I/O Contract
- Input: prompt text (from prompt.md).
- Output: plain text response captured into output.md.
- Errors: logged to agent-stderr.txt; non-zero exit code for failures.

## Environment / Config
- API key is stored in config.hcl and injected into the backend client (not exposed to agents for workflow use).
- Backend selection uses the same round-robin/weights as other agent types.
- Config includes a Perplexity section; use the most capable model by default unless overridden.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
