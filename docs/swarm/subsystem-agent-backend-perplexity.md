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
  - stdout (captured to agent-stdout.txt) - streams tokens as they arrive from SSE
  - output.md (written at completion with full response including citations)
- Errors: logged to agent-stderr.txt; non-zero exit code for failures.
- Streaming behavior: adapter streams tokens to stdout as they arrive from the API, providing real-time progress visibility.
- Citations: Search results and citations arrive at the end of the stream and are included in both stdout and output.md.
- Output file creation: adapter writes output.md when stream completes successfully (SSE distinguishes streaming chunks from completion).

## Environment / Config
- Tokens/credentials configured in `config.hcl`:
  ```hcl
  agent "perplexity" {
    token_file = "~/.config/perplexity/token"  # Recommended: file-based
    # OR: token = "..."                         # Inline value
    # Optional REST settings:
    # api_endpoint = "https://api.perplexity.ai/chat/completions"
    # model = "sonar-reasoning"
  }
  ```
- Runner automatically injects token as `PERPLEXITY_API_KEY` environment variable (hardcoded mapping).
- Token passed to REST adapter (not exposed to agents for workflow use).
- Backend selection uses same round-robin/weights as other agent types.
- Model defaults to most capable (sonar-reasoning) unless overridden in config.

## Implementation Details (REST/SSE)

### HTTP Request Format
- **URL**: `https://api.perplexity.ai/chat/completions`
- **Method**: POST
- **Headers**:
  - `Authorization: Bearer {PERPLEXITY_API_KEY}`
  - `Content-Type: application/json`
  - `Accept: text/event-stream` (for streaming)
- **Body**:
  ```json
  {
    "model": "sonar-reasoning",
    "messages": [{"role": "user", "content": "..."}],
    "stream": true
  }
  ```

### SSE Stream Parsing
- **Format**: Server-Sent Events with `data:` prefix
- **Events separated by**: Double newlines (`\n\n`)
- **Termination signal**: `data: [DONE]`
- **Content extraction**: `choices[0].delta.content` from each JSON chunk
- **Citations**: Appear in final chunks; collect from `search_results` and `citations` fields
- **Stream modes**:
  - `stream_mode="full"` (default): All chunks are `chat.completion.chunk`
  - `stream_mode="concise"`: Chunks include `chat.reasoning`, `chat.completion.chunk`, `chat.completion.done`
- **Delta behavior**: Perplexity may send accumulated full text (not just deltas like OpenAI)

### Error Handling
- **HTTP Status Codes**:
  - 400: Bad Request (invalid parameters) - do not retry
  - 401: Unauthorized (invalid API key) - do not retry
  - 429: Rate Limit Exceeded - retry with backoff
  - 500+: Server Errors - retry with exponential backoff
- **Streaming errors**: Check `finish_reason == "error"` or error objects mid-stream
- **Rate limiting headers**: `x-ratelimit-limit`, `x-ratelimit-remaining`, `x-ratelimit-reset`
- **Retry strategy**: Use `retry-after` header or exponential backoff (1s to 32s) with jitter

### Timeout Configuration
- **Connect timeout**: 10 seconds
- **TLS handshake**: 10 seconds
- **Response header timeout**: 10 seconds (time to first byte)
- **Idle timeout (streaming)**: 30-60 seconds (models can pause during "thinking")
- **Total request timeout**: 120+ seconds (complex queries take time)

### Go Implementation Notes
- Use `bufio.Scanner` for line-based SSE parsing
- Strip `data:` prefix and check for `[DONE]` marker
- Handle both delta and accumulated content modes
- Collect search_results/citations from final chunks
- Implement retry loop for 429/5xx errors with exponential backoff

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
- PERPLEXITY-API-HTTP-FORMAT.md (detailed implementation guide)
