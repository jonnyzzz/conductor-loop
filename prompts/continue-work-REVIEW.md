# Review: `prompts/continue-work.md`

**Date**: 2026-02-20  
**Reviewer**: Codex prompt review

## Verification Performed

- Verified all absolute paths referenced in `prompts/continue-work.md` exist.
- Ran `go build ./...` in `<project-root>` (pass).
- Ran `go test ./...` in `<project-root>` (pass; all listed packages green).
- Checked `run-agent.sh` existence and invocation contract.
- Validated monitor command form with `uv run python <project-root>/monitor-agents.py --help` (works).
- Cross-checked prompt statements against:
  - `<project-root>/docs/workflow/THE_PROMPT_v5.md`
  - `<project-root>/docs/dev/issues.md`
  - `<project-root>/AGENTS.md`
  - `<project-root>/docs/dev/instructions.md`
  - `<project-root>/docs/dev/development.md`

## Section-by-Section Feedback

### 1) Title + Context (`prompts/continue-work.md:1-8`)

- Clear intent and project context.
- Minor ambiguity previously present in line 3 (`~/Work/conductor-loop`) was removed by switching to `<project-root>` placeholders.

### 2) Required Reading (`prompts/continue-work.md:9-20`)

- Good: absolute paths are provided and valid.
- Gap: this list omits role prompt files required by the base methodology (`THE_PROMPT_v5.md:31-42`).
- Gap: no mention of task-state artifacts (`TASK_STATE.md`, task/project message buses) that are expected in orchestration docs (`AGENTS.md:236-243`, `THE_PROMPT_v5_orchestrator.md:40-44`).

### 3) Current State (`prompts/continue-work.md:21-30`)

- Verified true at review time:
  - `go build ./...` passes (`prompts/continue-work.md:23`)
  - `go test ./...` passes (`prompts/continue-work.md:24`)
- Risky/inconsistent:
  - Issue-severity/count statement relies on an inconsistent `ISSUES.md` state (`prompts/continue-work.md:29`, `ISSUES.md:77`, `ISSUES.md:723-731`).

### 4) Goals/Priorities (`prompts/continue-work.md:31-55`)

- Priority structure is understandable.
- Critical list mismatch: `ISSUE-003` is treated as CRITICAL here, but marked HIGH in `ISSUES.md` (`prompts/continue-work.md:38`, `ISSUES.md:76-78`).
- Scope is too broad for one autonomous pass (all criticals + all open questions + docs + highs) without explicit stop criteria.

### 5) Working Plan (`prompts/continue-work.md:56-67`)

- Mostly aligned with orchestration pattern.
- Misalignment with source methodology:
  - Missing explicit IntelliJ MCP Steroid-first requirement (`THE_PROMPT_v5.md:20-22`, `THE_PROMPT_v5.md:166-167`).
  - Missing role-prompt requirement for each sub-agent (`THE_PROMPT_v5.md:31-42`).
- Message bus instruction is ambiguous versus project conventions:
  - Prompt says write directly to `MESSAGE-BUS.md` (`prompts/continue-work.md:65`)
  - AGENTS says use task/project message bus with tooling (`AGENTS.md:281-284`)
  - Instructions says bus CLI is not yet implemented (`Instructions.md:229-235`)

### 6) Agent Execution (`prompts/continue-work.md:68-80`)

- `run-agent.sh` invocation pattern is correct (`run-agent.sh:3`, `run-agent.sh:15-18`).
- Run folder naming and artifact description are mostly correct (`run-agent.sh:26-40`, `run-agent.sh:75-90`).
- Monitor command is valid in this repo (`monitor-agents.py:10-15` + tested).

### 7) Quality Gates (`prompts/continue-work.md:81-90`)

- Includes major gates (`build`, `test`, `test -race`).
- Missing quality checks required in project conventions (`AGENTS.md:299-304`):
  - `go fmt`
  - `golangci-lint run`

### 8) Development Flow (`prompts/continue-work.md:92-108`)

- Strong alignment with `THE_PROMPT_v5.md` stage model.
- Improvement needed: line 94 references required flow but omits “role prompt per stage/run” requirement, which is part of the same methodology.

### 9) Constraints (`prompts/continue-work.md:109-118`)

- Reasonable constraints and max parallelism aligned with methodology.
- Missing conflict-resolution guidance when documents disagree (which is currently happening across `ISSUES.md` severity counts and bus protocol guidance).

## Specific Issues Found (With Line References)

1. **CRITICAL**: Priority-1 critical issue list is inconsistent with the source issue severities.  
   - Prompt: `prompts/continue-work.md:33-40`  
   - Source conflict: `ISSUES.md:76-78` marks `ISSUE-003` as HIGH.

2. **CRITICAL**: Prompt relies on `ISSUES.md` as a clean source, but the file currently has trailing non-issue content.  
   - Prompt dependency: `prompts/continue-work.md:15`, `prompts/continue-work.md:34`  
   - Corruption/noise observed: `ISSUES.md:749-781` (error logs/diff fragments appended).

3. **HIGH**: Missing required role-prompt workflow from `THE_PROMPT_v5.md`.  
   - Prompt omission: `prompts/continue-work.md:56-75`  
   - Required by base methodology: `THE_PROMPT_v5.md:31-42`.

4. **HIGH**: Missing IntelliJ MCP Steroid-first instruction required by methodology/conventions.  
   - Prompt omission: `prompts/continue-work.md:56-108`  
   - Required by base: `THE_PROMPT_v5.md:20-22`, `THE_PROMPT_v5.md:166-167`.

5. **HIGH**: Message-bus write protocol is ambiguous across docs; prompt does not resolve it.  
   - Prompt direct-write instruction: `prompts/continue-work.md:65`, `prompts/continue-work.md:88`  
   - Conflicting docs: `AGENTS.md:281-284`, `Instructions.md:229-235`.

6. **MEDIUM**: Scope is unbounded and can lead to non-terminating autonomous execution.  
   - Overloaded goals: `prompts/continue-work.md:33-55`  
   - Missing explicit completion boundary/timebox for the session.

7. **MEDIUM**: Quality gates are incomplete vs project conventions.  
   - Prompt gates: `prompts/continue-work.md:81-90`  
   - Missing from AGENTS baseline: `AGENTS.md:299-304`.

## Suggested Improvements

1. Normalize and fix issue priorities in one place before execution:
   - Either change `ISSUES.md` severity for `ISSUE-003` or update this prompt to match `ISSUES.md`.
   - Add a startup validation step that checks issue severity/count consistency before planning.

2. Add explicit role-prompt requirement:
   - Require using `THE_PROMPT_v5_{orchestrator,research,implementation,review,test,debug}.md` as base for each spawned run.

3. Add IntelliJ MCP Steroid requirement explicitly:
   - State it is primary for code review/tests/build validation, with CLI fallback only when needed.

4. Resolve message bus protocol conflict in prompt text:
   - Define one canonical workflow for this repo now (direct append fallback vs bus tooling/REST).

5. Add missing quality checks:
   - `go fmt ./...`
   - `golangci-lint run`

6. Bound the autonomous scope:
   - Example: “Complete all CRITICAL issues first; then stop and produce a status summary + next batch plan.”

7. Add a guard for malformed `ISSUES.md`:
   - Instruct agent to ignore non-issue trailing lines or require cleanup before prioritization.

## Overall Assessment

**NEEDS CHANGES**

The prompt is close to operational, but it currently has critical priority inconsistencies and methodology gaps that can misdirect a fully autonomous agent.
