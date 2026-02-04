# Agent Backend: Perplexity - Questions

- Q: Does the Perplexity API support streaming responses?
  A: **YES, resolved via research (2026-02-04)**. Perplexity API supports streaming via `stream=True` parameter. Uses SSE format. All models support streaming. Response returns chunks for iteration. Citations arrive at end of stream. Integrated into subsystem-agent-backend-perplexity.md.

  Research sources:
  - [Perplexity Streaming Responses Documentation](https://docs.perplexity.ai/guides/streaming-responses)
  - [LiteLLM Perplexity Provider Documentation](https://docs.litellm.ai/docs/providers/perplexity)

---

## Open Questions (From Codex Review Round 2)

### Q1: Perplexity output.md Behavior Conflict
**Issue**: Conflicting statements about output.md handling for Perplexity adapter.

**ROUND-6-SUMMARY.md:48** says:
> "Perplexity output convention: uses stdout (agent-stdout.txt), no output.md for Perplexity"

**Current subsystem-agent-backend-perplexity.md:14** says:
> "The adapter writes the final response to BOTH stdout (captured to agent-stdout.txt) AND output.md"

**Question**: Which is correct for the Perplexity REST adapter?

**Context**: Perplexity is a REST adapter (not CLI), so the adapter has full control over file writing.

**Proposed Fix**:
- Update ROUND-6-SUMMARY.md to reflect current decision (BOTH files)
- Or update current spec to clarify which approach is correct

**Answer**: [PENDING]

---

### Q2: Perplexity REST Adapter Implementation Details
**Issue**: Current spec lacks concrete REST/SSE parsing details needed for implementation.

**Missing Information**:
- Exact HTTP request format (headers, body structure)
- SSE event parsing (how to extract delta/content from events)
- Stream termination detection
- Error response handling
- Timeout handling

**Question**: Should these implementation details be added to subsystem-agent-backend-perplexity.md, or are they implementation-specific?

**Answer**: [PENDING]
