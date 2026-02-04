# Cross-Document Consistency Questions

These questions address contradictions between summary/planning documents.

---

## Q1: ROUND-7-SUMMARY Gemini Streaming Status Contradiction

**Issue**: Same document contradicts itself about Gemini streaming verification status.

**Line 117** (Implementation Readiness table):
```
| Gemini | ... | ⚠️ Pending | ... | 🟡 MOSTLY READY |
```

**Line 160** (Open Questions section):
```
- Gemini CLI streaming behavior verification (experimental testing needed)
```

But **Line 175** and specs say:
- Gemini streaming WAS verified via experiments
- Results: ~1s chunk intervals, progressive output
- Status should be ✅ Verified and 🟢 READY

**Question**: This was supposed to be fixed in commit 2133bb2. Did the fix not apply correctly?

**Action**: Verify ROUND-7-SUMMARY.md current content and fix any remaining contradictions.

**Answer**: [PENDING - Need to check file]

---

## Q2: PLANNING-COMPLETE vs TOPICS Streaming Claims Mismatch

**Issue**: Different documents make contradictory claims about streaming verification.

**PLANNING-COMPLETE.md:160** (Agent Backends table) claims:
```
| Codex   | ... | Streaming | ✅ Assumed working | Status: 🟢 READY |
| Claude  | ... | Streaming | ✅ Assumed working | Status: 🟢 READY |
```
Text: "streaming verified for all backends"

**TOPICS.md:107** (Topic #8) says:
```
- Codex/Claude streaming: Assumed working based on standard CLI behavior
```

**Question**: Which phrasing is more accurate?
- "Verified" implies we tested it (like Gemini/Perplexity)
- "Assumed working" is more honest (we didn't test, we assume)

**Proposed Fix**: Change PLANNING-COMPLETE.md to say "assumed working" to match TOPICS.md (more accurate).

**Answer**: [PENDING]

---

## Q3: Config Key References in Multiple Files

**Issue**: Multiple files reference the incorrect config key approach.

**Files referencing `openai_api_key`/`anthropic_api_key`/etc**:
- subsystem-agent-backend-codex.md:30
- subsystem-agent-backend-claude.md:30
- subsystem-agent-backend-gemini.md:30
- subsystem-agent-backend-perplexity.md:34
- TOPICS.md:107 (environment variable mappings section)
- SUBSYSTEMS.md (Subsystem #7 description)
- ROUND-7-SUMMARY.md (Environment Variables section)

**Correct approach** (from config schema):
- Use per-agent `token` field in agent blocks
- Use `env_var` field to specify environment variable name

**Question**: Should all these files be updated in one coordinated fix?

**Answer**: [PENDING - Depends on Q1 in subsystem-runner-orchestration-QUESTIONS.md]
