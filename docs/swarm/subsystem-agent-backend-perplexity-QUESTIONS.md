# Agent Backend: Perplexity - Questions

- Q: Does the Perplexity API support streaming responses?
  A: Research needed for Perplexity API streaming capabilities.
  - If supported: Use streaming to emit tokens as they arrive
  - If not supported: Emit "[Perplexity] Generating..." keep-alive messages every 30 seconds to prevent stuck detection
