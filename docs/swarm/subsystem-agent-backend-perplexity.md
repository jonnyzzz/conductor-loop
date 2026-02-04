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
- **Streaming Support**: Perplexity API supports streaming via the `stream=True` parameter (Python) or `stream: true` (TypeScript).
  - Uses Server-Sent Events (SSE) format
  - All models support streaming: sonar-pro, sonar-reasoning, sonar-reasoning-pro, sonar-deep-research, r1-1776
  - Response returns chunks that can be iterated over for real-time display
  - Example: `{"model": "sonar", "messages": [...], "stream": True}`
- The adapter should enable streaming and emit tokens to stdout as they arrive to provide real-time progress and avoid idle/stuck detection.
- Citations: Perplexity citations arrive at the end of the stream and should be included inline in the final response text.

## I/O Contract
- Input: prompt text (from prompt.md).
- Output: plain text response written to BOTH:
  - output.md (canonical result file for UI display)
  - stdout (captured to agent-stdout.txt for streaming progress)
- Errors: logged to agent-stderr.txt; non-zero exit code for failures.
- Streaming behavior: adapter streams tokens to stdout as they arrive from the API, providing real-time progress visibility.
- Citations: Search results and citations arrive at the end of the stream and are appended to the final response.

## Environment / Config
- Tokens/credentials are injected by run-agent from config:
  - Environment variable: `PERPLEXITY_API_KEY`
  - Config key: `perplexity_api_key` (in `config.hcl`)
  - Supports @file reference for token file paths (e.g., `perplexity_api_key = "@/path/to/key.txt"`)
- API key is injected into the backend REST client (not exposed to agents for workflow use).
- Backend selection uses the same round-robin/weights as other agent types.
- Config includes a Perplexity section; use the most capable model by default unless overridden.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
