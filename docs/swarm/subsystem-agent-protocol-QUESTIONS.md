# Agent Protocol & Governance - Questions

## Open Questions (From Codex Review Round 2)

### Q1: output.md Generation Responsibility
**Issue**: Multiple conflicting statements about who creates output.md for CLI backends.

**Conflicting Specs**:
1. **Agent Protocol** (subsystem-agent-protocol.md:16): Implies agents should write output.md
2. **Backend Specs** (subsystem-agent-backend-codex.md:21): "stdout captured to agent-stdout.txt; runner may create output.md"
3. **Storage Layout** (subsystem-storage-layout.md:32): Lists output.md as expected file
4. **run-agent.sh**: Only captures to agent-stdout.txt, never creates output.md

**User Clarification** (2026-02-04):
> "output.md is recommended behavior for the agent, we add to the top of the prompts and ask the agent to write the result of work to the output.md in the run_id folder. No guarantees it would do. The tooling will anyways redirect streams to the files."

**Question**: How should this be documented clearly?

**Proposed Fix Options**:

**Option A - Best Effort + Fallback**:
- Prompt asks agents to write output.md (best effort)
- Runner always creates output.md from stdout as fallback
- UI reads output.md (guaranteed to exist)

**Option B - Best Effort Only**:
- Prompt asks agents to write output.md
- Runner only captures stdout to agent-stdout.txt
- UI checks output.md first, falls back to agent-stdout.txt

**Option C - Runner Always Creates**:
- Agents write to stdout only
- Runner always creates output.md from captured stdout
- Simpler, guaranteed behavior

**Answer**: [PENDING - Need to choose A, B, or C]

---

No other open questions at this time. All previous questions have been resolved and integrated into subsystem-agent-protocol.md and subsystem-runner-orchestration.md.
