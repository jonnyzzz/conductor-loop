# Agent Backend: Perplexity - Questions

- Q: Does the Perplexity API support streaming responses?
  A: **YES, resolved via research (2026-02-04)**. Perplexity API supports streaming via `stream=True` parameter. Uses SSE format. All models support streaming. Response returns chunks for iteration. Citations arrive at end of stream. Integrated into subsystem-agent-backend-perplexity.md.

  Research sources:
  - [Perplexity Streaming Responses Documentation](https://docs.perplexity.ai/guides/streaming-responses)
  - [LiteLLM Perplexity Provider Documentation](https://docs.litellm.ai/docs/providers/perplexity)

---

## Open Questions

### Q1 (Re-opened): Perplexity output.md generation responsibility
**Issue**: The current spec states that the runner generates output.md from stdout, but the resolution below says the adapter writes output.md on completion.

**Question**: Should the Perplexity adapter write output.md directly, or should the runner always generate output.md from stdout (current spec)?

**Answer**: That is up to the tool to create any files it can. The conoslle output should be declared well enough to make it discoverable.

## Resolved Questions

### Q1: Perplexity output.md Behavior Conflict
**Issue**: Conflicting statements about output.md handling for Perplexity adapter.

**ROUND-6-SUMMARY.md:48** says:
> "Perplexity output convention: uses stdout (agent-stdout.txt), no output.md for Perplexity"

**Previous subsystem-agent-backend-perplexity.md (2026-02-04)** said:
> "The adapter writes the final response to BOTH stdout (captured to agent-stdout.txt) AND output.md"

**Question**: Which is correct for the Perplexity REST adapter?

**Context**: Perplexity is a REST adapter (not CLI), so the adapter has full control over file writing.

**Proposed Fix**:
- Update ROUND-6-SUMMARY.md to reflect current decision (BOTH files)
- Or update current spec to clarify which approach is correct

**Answer**: Perplexity tool creates stdout and stderr files. It only creates the output.md file if that API clearly tells the difference between streaming and progress and the final answer.

**Resolution** (2026-02-04):
- Updated subsystem-agent-backend-perplexity.md I/O Contract section
- Clarified that adapter writes both stdout (streaming) and output.md (at completion)
- SSE format distinguishes streaming chunks from completion, enabling output.md creation

**Update (2026-02-05)**: Current spec now says the runner generates output.md from stdout; see the open question above to confirm whether this supersedes the 2026-02-04 resolution.

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

**Answer**: Conduct the research to learn more details of that, find answers. Use multiple run-agent.sh with claude, codex, gemini to research.

**Resolution** (2026-02-04):
- Delegated research to three agents: claude (HTTP format), codex (SSE parsing), gemini (error handling)
- Created comprehensive PERPLEXITY-API-HTTP-FORMAT.md with all implementation details
- Updated subsystem-agent-backend-perplexity.md with new "Implementation Details (REST/SSE)" section
- Documented: HTTP request format, SSE parsing, error handling, rate limiting, timeouts, Go implementation notes
- Research runs:
  - run_20260204-203710-54667 (claude): HTTP format and headers
  - run_20260204-203955-55799 (codex): SSE event parsing and delta extraction
  - run_20260204-204303-56723 (gemini): Error handling and timeout configuration

---

- Q: Config format/token syntax mismatch: specs reference config.hcl with inline or `@file` token values, but code currently loads YAML with `token`/`token_file` fields and no `@file` shorthand. Which format is authoritative, and should `@file` be supported by the runner?
  Answer: YAML is authoritative. The `config.hcl` references in specs are outdated — all code uses YAML (`config.yaml`) with `token` and `token_file` as mutually exclusive fields. The `@file` shorthand is NOT supported and NOT needed; `token_file: /path/to/file` achieves the same goal with clearer semantics. No changes required.
