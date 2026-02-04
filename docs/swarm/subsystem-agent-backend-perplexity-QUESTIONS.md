# Agent Backend: Perplexity - Questions

- Q: Does the Perplexity API support streaming responses?
  A: **YES, resolved via research (2026-02-04)**. Perplexity API supports streaming via `stream=True` parameter. Uses SSE format. All models support streaming. Response returns chunks for iteration. Citations arrive at end of stream. Integrated into subsystem-agent-backend-perplexity.md.

  Research sources:
  - [Perplexity Streaming Responses Documentation](https://docs.perplexity.ai/guides/streaming-responses)
  - [LiteLLM Perplexity Provider Documentation](https://docs.litellm.ai/docs/providers/perplexity)

No open questions at this time.
