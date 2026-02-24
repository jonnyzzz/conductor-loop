# Orchestrator: Create Next Tasks for Project Evolution (Element 3)

## FIRST: Read your operating manuals

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PROMPT_v5.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PROMPT_v5_orchestrator.md
cat /Users/jonnyzzz/Work/jonnyzzz.com-src/RLM.md
```

## Your Mission

Run **5 iterations** of a 3-agent workgroup to:
1. Synthesize the full project state from all facts/docs
2. Identify the most valuable next steps for project evolution
3. Create well-structured task files ready for execution by run-agent
4. Prioritize ruthlessly — P0 first, only actionable concrete tasks

## Output locations

- Updated: `docs/facts/FACTS-suggested-tasks.md`
- New: `docs/roadmap/` folder with detailed evolution plans
- New: individual task prompt files in `prompts/tasks/` ready for `run-agent job --prompt-file`

## Inputs to synthesize

```bash
# Current state
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-suggested-tasks.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/dev/todos.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md

# Architecture vision
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-swarm-ideas.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-prompts-workflow.md

# What exists
ls /Users/jonnyzzz/Work/conductor-loop/docs/architecture/ 2>/dev/null
cat /Users/jonnyzzz/Work/conductor-loop/docs/architecture/README.md 2>/dev/null
```

## Iteration Plan

### Iteration 1 — Full State Analysis (3 agents in parallel)

**Agent A (claude) — Gap Analysis:**
- What features does README.md describe that have no test coverage?
- What features are in FACTS-suggested-tasks.md P0 but still open?
- What open ISSUES have no assigned task?
- Output: `docs/roadmap/gap-analysis.md`
- Commit

**Agent B (gemini) — Evolution Opportunities:**
- From FACTS-swarm-ideas.md: which unimplemented ideas are now feasible?
- From FACTS-prompts-workflow.md: what workflow improvements would help most?
- From FACTS-runs-conductor.md (125 runs): what patterns keep appearing as friction?
- Output: `docs/roadmap/evolution-opportunities.md`
- Commit

**Agent C (codex) — Technical Debt Map:**
- From FACTS-issues-decisions.md: what deferred issues need resolution?
- What is the binary port mismatch status? (`conductor` binary still showing 8080)
- What specs vs code drift remains?
- What test coverage gaps exist in `internal/`?
- Output: `docs/roadmap/technical-debt.md`
- Commit

### Iteration 2 — Task Creation: Critical Reliability

**Agent A (codex):** Create detailed task prompts for P0 items:
- `prompts/tasks/fix-conductor-binary-port.md` — reconcile port 8080/14355
- `prompts/tasks/fix-sse-cpu-hotspot.md` — SSE 100ms polling fix
- `prompts/tasks/fix-monitor-process-cap.md` — process proliferation cap
- Each prompt: full context, acceptance criteria, verification steps

**Agent B (gemini):** Create task prompts for P1 correctness:
- `prompts/tasks/implement-output-synthesize.md` — `run-agent output synthesize`
- `prompts/tasks/implement-review-quorum.md` — `run-agent review quorum`
- `prompts/tasks/implement-iterate.md` — `run-agent iterate`
- Include: current binary state, expected API design, test requirements

**Agent C (claude):** Create task prompts for P1 security/release:
- `prompts/tasks/token-leak-audit.md` — full repo token scan
- `prompts/tasks/release-readiness-gate.md` — CI green + integration tests
- `prompts/tasks/unified-bootstrap.md` — merge install.sh and run-agent.cmd
- Include: step-by-step verification, success criteria

All commit after creating prompts.

### Iteration 3 — Task Creation: Architecture Evolution

**Agent A (claude):** Create task prompts for architecture improvements:
- `prompts/tasks/merge-conductor-run-agent.md` — merge two binaries or clarify separation
- `prompts/tasks/hcl-config-deprecation.md` — formally deprecate HCL, YAML-only
- `prompts/tasks/env-sanitization.md` — inject only required agent API keys
- `prompts/tasks/global-fact-storage.md` — promote facts across scopes

**Agent B (codex):** Create task prompts for Windows support:
- `prompts/tasks/windows-file-locking.md` — ISSUE-002: shared-lock readers
- `prompts/tasks/windows-process-groups.md` — ISSUE-003: Job Objects
- Include: current stub locations in code, implementation approach

**Agent C (gemini):** Create task prompts for UX improvements:
- `prompts/tasks/ui-latency-fix.md` — multi-second update lag
- `prompts/tasks/ui-task-tree-guardrails.md` — regression test suite
- `prompts/tasks/gemini-stream-json-fallback.md` — CLI version compatibility

All commit.

### Iteration 4 — Roadmap Synthesis + Prioritization

**Agent A (gemini):** Create `docs/roadmap/ROADMAP.md`:
- Quarters view: Q1 2026 (now), Q2, Q3, long-term
- P0 → Q1, P1 → Q2, Architecture → Q3, Long-term ideas
- Each item: task prompt file reference, estimated complexity (S/M/L/XL)
- Commit

**Agent B (claude):** Update `docs/facts/FACTS-suggested-tasks.md`:
- Add all new tasks created in iterations 2-3
- Remove any now superseded by new task prompts
- Mark P0 items with `prompt-file:` references
- Commit

**Agent C (codex):** Create `docs/roadmap/quick-wins.md`:
- Tasks completable in a single run-agent session (<30min)
- Tasks with clear acceptance criteria and no dependencies
- Ordered by value/effort ratio
- Commit

### Iteration 5 — Review, Validate, Final Commit

All 3 agents review the full `docs/roadmap/` and `prompts/tasks/` output:

**Agent A:** Verify every task prompt is self-contained (includes enough context to run standalone)
**Agent B:** Verify roadmap priorities are consistent with facts (no P2 above P0)
**Agent C:** Create `docs/roadmap/README.md` — index of all roadmap docs

Final commit:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
git add docs/roadmap/ prompts/tasks/
git commit -m "docs(roadmap): complete project evolution roadmap and task prompts"
git push origin main
```

## Completion signal

```bash
echo "DONE" > /Users/jonnyzzz/run-agent/conductor-loop/TASK_NEXTTASKS_DONE
```

Write final summary to `output.md`.
