# Questions to Resolve

Based on Codex Review Round 2 - these need answers to proceed with implementation.

---

## 🔴 CRITICAL (Implementation Blockers)

### 1. Config Credential Schema Approach
**File**: `subsystem-runner-orchestration-QUESTIONS.md` → Q1
**Issue**: Backend specs say `openai_api_key`, config schema uses `token` + `env_var`

**Quick Summary**:
- Config schema (CORRECT): `agent "codex" { token = "..."; env_var = "OPENAI_API_KEY" }`
- Backend specs (WRONG): Reference `openai_api_key` as a config key

**Question**: Should I update all backend specs to reference the config schema approach?

**Suggested Answer**: Yes, update backend specs to match config schema

---

### 2. output.md Generation Responsibility
**File**: `subsystem-agent-protocol-QUESTIONS.md` → Q1
**Issue**: Who creates output.md for CLI backends?

**Your Earlier Comment**:
> "Prompt recommends agents write output.md (best effort). Runner redirects stdout/stderr regardless."

**Question**: Which option should we document?

**Options**:
- **A**: Best effort + fallback (prompt asks, runner creates from stdout as backup)
- **B**: Best effort only (prompt asks, UI reads agent-stdout.txt if missing)
- **C**: Runner always (agents → stdout, runner → output.md)

**Suggested Answer**: Option A (matches your comment, guarantees UI has output.md)

---

## ⚠️ MEDIUM (Consistency Issues)

### 3. Perplexity output.md Behavior
**File**: `subsystem-agent-backend-perplexity-QUESTIONS.md` → Q1
**Issue**: ROUND-6 says "stdout only", current spec says "BOTH files"

**Question**: Which is correct?

**Suggested Answer**: BOTH files (current spec is correct, update ROUND-6-SUMMARY)

---

### 4. Codex cli_flags Example
**File**: `subsystem-runner-orchestration-QUESTIONS.md` → Q2
**Issue**: Example missing CWD value and stdin `-` marker

**Question**: How to document cli_flags that need runtime values?

**Suggested Answer**: Document that `-C` takes CWD from runner, `-` for stdin is automatic

---

## 📝 LOW (Documentation Cleanup)

### 5. ROUND-7-SUMMARY Gemini Contradiction
**File**: `CONSISTENCY-QUESTIONS.md` → Q1
**Issue**: Says both "pending" and "verified" for Gemini streaming

**Question**: Was this fixed in commit 2133bb2?

**Action**: Check file and fix any remaining contradictions

---

### 6. PLANNING-COMPLETE vs TOPICS Mismatch
**File**: `CONSISTENCY-QUESTIONS.md` → Q2
**Issue**: PLANNING-COMPLETE says "verified", TOPICS says "assumed"

**Question**: Which phrasing for Codex/Claude streaming?

**Suggested Answer**: "Assumed working" (more accurate, we didn't test)

---

### 7. Perplexity REST Adapter Details
**File**: `subsystem-agent-backend-perplexity-QUESTIONS.md` → Q2
**Issue**: Missing HTTP request/SSE parsing implementation details

**Question**: Add to spec or leave as implementation-specific?

**Suggested Answer**: Add basic outline (headers, request format, SSE event structure)

---

## 📋 Recommended Answer Order

**Step 1 - Critical** (blocks implementation):
1. ✅ Q1: Config schema approach
2. ✅ Q2: output.md responsibility

**Step 2 - Medium** (fixes inconsistencies):
3. ✅ Q3: Perplexity output.md
4. ✅ Q4: Codex cli_flags

**Step 3 - Low** (documentation cleanup):
5. ✅ Q5: ROUND-7 contradiction
6. ✅ Q6: PLANNING-COMPLETE wording
7. ✅ Q7: Perplexity details

---

## Quick Answer Template

If you agree with all suggested answers, just say:
> "Approve all suggested answers"

Or answer individually:
> Q1: Yes, update to match config schema
> Q2: Option A (best effort + fallback)
> Q3: BOTH files (fix ROUND-6)
> Q4: Document runtime injection
> Q5: Check and fix
> Q6: Change to "assumed"
> Q7: Add basic outline

---

## After Answers

Once you provide answers, I will:
1. Update all affected specification files
2. Update all summary documents
3. Commit changes
4. Run final review round 3 (Gemini + Codex)
5. Verify all issues resolved
