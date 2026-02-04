# Specification Fixes Required

**Date**: 2026-02-04
**Based on**: Gemini + Codex final reviews
**Status**: Codex REJECTED, Gemini APPROVED

---

## Review Summary

| Reviewer | Score | Status | Key Issue |
|----------|-------|--------|-----------|
| **Gemini** | 10/10/10 | ✅ APPROVED | No issues found |
| **Codex** | 7/7/6 | ❌ REJECTED | 2 critical blockers |

---

## CRITICAL ISSUES (Implementation Blockers)

### 1. Config Key Naming Mismatch 🔴 BLOCKER

**Problem**: Backend specs contradict config schema

**Current State**:
- **Config Schema** (subsystem-runner-orchestration-config-schema.md): ✅ CORRECT
  ```hcl
  agent "codex" {
    token = "@~/.config/openai/token"
    env_var = "OPENAI_API_KEY"
  }
  ```

- **Backend Specs** (subsystem-agent-backend-*.md): ❌ WRONG
  ```
  - Config key: `openai_api_key` (in `config.hcl`)
  ```

**Impact**: Blocks Go config loading and environment variable injection

**Files to Fix**:
- [ ] subsystem-agent-backend-codex.md
- [ ] subsystem-agent-backend-claude.md
- [ ] subsystem-agent-backend-gemini.md
- [ ] subsystem-agent-backend-perplexity.md
- [ ] TOPICS.md (Topic #8)
- [ ] SUBSYSTEMS.md (Subsystem #7)
- [ ] ROUND-7-SUMMARY.md (Environment Variables section)

**Fix Action**:
Replace:
```markdown
- Environment variable: `OPENAI_API_KEY`
- Config key: `openai_api_key` (in `config.hcl`)
- Supports @file reference for token file paths (e.g., `openai_api_key = "@/path/to/key.txt"`)
```

With:
```markdown
- Environment variable: `OPENAI_API_KEY`
- Config: Set in agent block per subsystem-runner-orchestration-config-schema.md
  ```hcl
  agent "codex" {
    token = "@~/.config/openai/token"  # Supports @file references
    env_var = "OPENAI_API_KEY"         # Optional, defaults to OPENAI_API_KEY
  }
  ```
```

---

### 2. output.md Ownership Ambiguity 🔴 BLOCKER

**Problem**: Unclear who is responsible for creating output.md

**Conflicting Statements**:
1. **Agent Protocol** (subsystem-agent-protocol.md): Suggests agents should write output.md
2. **Backend Specs** (subsystem-agent-backend-*.md): "stdout captured into output.md"
3. **run-agent.sh**: Only captures to agent-stdout.txt, no output.md creation
4. **Storage Layout** (subsystem-storage-layout.md): Lists output.md as expected file

**Impact**: Blocks implementation of output handling in Go runner

**Decision Needed**: Choose ONE approach:

#### Option A: Runner Creates output.md (Recommended)
- Runner captures agent stdout → writes to both agent-stdout.txt and output.md
- Agents only write to stdout (simpler for agents)
- Consistent with current run-agent.sh behavior (with enhancement)

#### Option B: Agents Create output.md
- Agents responsible for writing output.md directly
- Runner only captures stdout → agent-stdout.txt
- Requires all agent adapters to implement file writing

**Files to Update** (after decision):
- [ ] subsystem-agent-protocol.md (clarify agent responsibility)
- [ ] subsystem-runner-orchestration.md (specify runner behavior)
- [ ] subsystem-storage-layout.md (clarify output.md source)
- [ ] All subsystem-agent-backend-*.md files (update I/O contract)

---

## MINOR ISSUES (Non-Blocking, Should Fix)

### 3. Stale Text in TOPICS.md

**Issue**: Topics #7 and #8 still list "Open questions" that are now resolved

**Files to Fix**:
- [ ] TOPICS.md

**Fix Action**:
- Topic #7 (Environment Contract): Change "Open questions: None at this time" (already correct)
- Topic #8 (Agent Backend Integrations): Change "Open questions: None at this time"
- Remove or update any remaining "⚠️ Pending" markers

---

### 4. ROUND-7-SUMMARY.md Inconsistency

**Issue**: Says "Gemini streaming pending" but it's actually verified

**Location**: ROUND-7-SUMMARY.md, Implementation Readiness table

**Current**:
```
| Gemini | ... | ⚠️ Pending | ... | 🟡 MOSTLY READY |
```

**Should be**:
```
| Gemini | ... | ✅ Verified | ... | 🟢 READY |
```

**Files to Fix**:
- [ ] ROUND-7-SUMMARY.md (Implementation Readiness table)
- [ ] ROUND-7-SUMMARY.md (Final Status section)

---

### 5. Stale References in QUESTIONS Files

**Issue**: Some QUESTIONS files say schema docs "to be created" but they exist

**Files to Fix**:
- [ ] subsystem-runner-orchestration-QUESTIONS.md
- [ ] subsystem-storage-layout-QUESTIONS.md

**Fix Action**: Remove or update references to non-existent schema docs

---

## MISSING INFORMATION (Should Document)

### 6. Streaming Behavior for Codex/Claude

**Issue**: Codex and Claude backend specs don't explicitly document streaming behavior

**Files to Update**:
- [ ] subsystem-agent-backend-codex.md
- [ ] subsystem-agent-backend-claude.md

**Add to I/O Contract section**:
```markdown
- Streaming behavior: CLI streams output to stdout progressively (assumed based on standard CLI behavior, similar to Gemini verified behavior).
```

---

### 7. Default env_var Mappings

**Issue**: Config schema doesn't specify default env_var for each agent type

**File to Update**:
- [ ] subsystem-runner-orchestration-config-schema.md

**Add to env_var section**:
```markdown
##### env_var (string, optional)

Environment variable name for token injection (e.g., "OPENAI_API_KEY").

**Defaults by agent type** (if omitted):
- codex: `OPENAI_API_KEY`
- claude: `ANTHROPIC_API_KEY`
- gemini: `GEMINI_API_KEY`
- perplexity: `PERPLEXITY_API_KEY`

If omitted and no default exists, token is not injected via environment.
```

---

## Fix Priority

### P0 - Must Fix Before Implementation
1. ✅ Config key naming mismatch (CRITICAL #1)
2. ✅ output.md ownership ambiguity (CRITICAL #2)

### P1 - Should Fix Before Implementation
3. TOPICS.md open questions cleanup
4. ROUND-7-SUMMARY.md Gemini status
5. QUESTIONS files stale references

### P2 - Nice to Have (Document Assumptions)
6. Codex/Claude streaming behavior documentation
7. Default env_var mappings specification

---

## Recommended Approach

### Step 1: Make Decisions (Required)
- [ ] Decide output.md ownership: Runner (Option A) or Agent (Option B)?
- [ ] Confirm config key approach: Use config schema's `token` + `env_var` approach

### Step 2: Fix Critical Issues (P0)
- [ ] Update all 4 backend specs with correct config approach
- [ ] Update TOPICS.md and SUBSYSTEMS.md
- [ ] Update ROUND-7-SUMMARY.md
- [ ] Clarify output.md ownership in 4 specification files

### Step 3: Fix Minor Issues (P1)
- [ ] Clean up TOPICS.md open questions
- [ ] Fix ROUND-7-SUMMARY.md Gemini status
- [ ] Update QUESTIONS files

### Step 4: Document Assumptions (P2)
- [ ] Add streaming behavior notes to Codex/Claude specs
- [ ] Add default env_var mappings to config schema

### Step 5: Re-verify
- [ ] Run Codex review again
- [ ] Commit all changes
- [ ] Update PLANNING-COMPLETE.md status

---

## Time Estimate

- P0 fixes: 15-20 minutes
- P1 fixes: 10 minutes
- P2 documentation: 5 minutes
- Re-verification: 5 minutes
- **Total**: ~40 minutes

---

## Decision Required

Before proceeding, please decide:

1. **output.md ownership**: Runner creates (Option A) or Agent creates (Option B)?
2. **Config approach**: Confirm we use config schema's `token` + `env_var` design?

Default recommendation if no preference:
- ✅ output.md: **Runner creates** (Option A) - simpler for agents
- ✅ Config: **Use config schema as-is** - it's already correct
