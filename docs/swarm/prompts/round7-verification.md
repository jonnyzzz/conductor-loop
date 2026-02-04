# Round 7 Verification: Agent Backend and Perplexity Updates

## Context
Round 7 focused on completing remaining agent backend specifications with emphasis on:
1. Perplexity API streaming research (WebSearch-based, Perplexity MCP unavailable)
2. Agent backend CLI flags verification from run-agent.sh
3. Environment variable mappings for all backends
4. Consistency updates across SUBSYSTEMS.md and TOPICS.md

## Your Task
Review the following updated specifications for:
- **Completeness**: Are all decisions documented? Any gaps?
- **Consistency**: Do specifications align with run-agent.sh implementation?
- **Correctness**: Are technical details accurate (especially Perplexity streaming)?
- **Cross-references**: Do SUBSYSTEMS.md and TOPICS.md reflect latest updates?

## Updated Files (Round 7)
1. subsystem-agent-backend-perplexity.md (streaming SSE details added)
2. subsystem-agent-backend-perplexity-QUESTIONS.md (question resolved)
3. subsystem-agent-backend-claude.md (env vars confirmed)
4. subsystem-agent-backend-claude-QUESTIONS.md (flags confirmed from run-agent.sh)
5. subsystem-agent-backend-codex.md (OPENAI_API_KEY + @file support)
6. subsystem-agent-backend-codex-QUESTIONS.md (resolved with run-agent.sh details)
7. subsystem-agent-backend-gemini.md (GEMINI_API_KEY + @file support)
8. subsystem-agent-backend-gemini-QUESTIONS.md (streaming verification pending)
9. SUBSYSTEMS.md (subsystem #7 updated)
10. TOPICS.md (Topics #7 and #8 updated with decisions)
11. MESSAGE-BUS.md (Round 7 entry added)

## Key Decisions Documented (Round 7)

### Perplexity Streaming
- API supports streaming via `stream=True` parameter
- Uses SSE (Server-Sent Events) format
- All models support streaming
- Citations arrive at end of stream
- Sources verified: Perplexity docs, LiteLLM docs

### Environment Variables
- Codex: `OPENAI_API_KEY` (config key: `openai_api_key`)
- Claude: `ANTHROPIC_API_KEY` (config key: `anthropic_api_key`)
- Gemini: `GEMINI_API_KEY` (config key: `gemini_api_key`)
- Perplexity: `PERPLEXITY_API_KEY` (config key: `perplexity_api_key`)
- All support @file reference (e.g., `@/path/to/key.txt`)

### CLI Flags (from run-agent.sh)
- Claude: `claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions`
- Codex: `codex exec --dangerously-bypass-approvals-and-sandbox -C "$CWD" -`
- Gemini: `gemini --screen-reader true --approval-mode yolo`

## Verification Checklist

Please verify:
1. ✓ Perplexity streaming research accurately documented?
2. ✓ All agent backend specs include env var + @file support?
3. ✓ CLI flags match run-agent.sh implementation exactly?
4. ✓ SUBSYSTEMS.md subsystem #7 reflects latest status?
5. ✓ TOPICS.md Topics #7 and #8 include all resolved decisions?
6. ✓ Only open question remaining is Gemini streaming verification?
7. ✓ Cross-references between specs are correct?
8. ✓ Any technical inaccuracies in Perplexity streaming details?

## Output Format
Provide structured feedback:
- **Issues Found**: List any problems, inconsistencies, or gaps
- **Technical Concerns**: Flag any technical inaccuracies
- **Suggestions**: Improvements for clarity or completeness
- **Approval**: State if specifications are implementation-ready

## Related Documents
Read these files for context:
- ../run-agent.sh (implementation reference)
- subsystem-agent-backend-*.md (all 5 backend specs)
- SUBSYSTEMS.md (subsystem registry)
- TOPICS.md (cross-cutting topics)
- ROUND-6-SUMMARY.md (previous round context)
