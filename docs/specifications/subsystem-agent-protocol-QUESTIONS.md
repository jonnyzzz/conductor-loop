# Agent Protocol & Governance - Questions

## Open Questions

### Q2: Restart Prompt Prefix Enforcement
**Issue**: Protocol now requires run-agent to prefix restarts with `Continue working on the following`, but the runner always uses the same prompt text.

**Code Evidence**:
- `internal/runner/orchestrator.go` `buildPrompt()` always uses a fixed preamble.
- `internal/runner/task.go` passes the same prompt on every restart.

**Question**: Should run-agent inject the restart prefix only after the first attempt, and if so, should it be inserted before or after the TASK_FOLDER/RUN_FOLDER preamble?

**Answer**: (Pending - user)

### Q3: Delegation Depth Enforcement
**Issue**: Protocol specifies a max delegation depth of 16, but there is no runtime enforcement.

**Code Evidence**:
- No depth checks exist in `internal/runner/` or `internal/agent/`.

**Question**: Where should depth be tracked and enforced (runner spawn, CLI flags, or prompt-level convention only)?

**Answer**: (Pending - user)

## Resolved Questions

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

**Answer**:  When agent is started, we prepend to the begining of the prompt: "Write output.md to <run_id>/output.md".
In this message we keep the full path to output.md, so we instruct agent to create the file.
This is the best-effort approach. In addition to that, and fully independently, the run-agent functionality
will manage the stdin and stdout of the agent and create the related stdout.txt/stderr.txt files. These files will be
created in the run_id folder, and these files are the fallback. The run-agent tool source outpout the infomtation
about the target files to it's output to help the parent agent know what to do.

**Resolution** (2026-02-04):
- Updated subsystem-agent-protocol.md with new "Output Files & I/O Capture" section
- Updated agent protocol behavioral rules to clarify best-effort approach
- Updated subsystem-agent-backend-perplexity.md I/O contract
- All specs now consistent: prompt instructs output.md creation, runner captures stdout/stderr independently

---
