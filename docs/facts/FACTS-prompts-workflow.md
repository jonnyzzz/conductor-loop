# FACTS: Prompts, Workflow & Methodology

Extracted from all revisions of THE_PROMPT_v5.md, role prompt files, AGENTS.md, CLAUDE.md, Instructions.md, and rlm-orchestration.md.

---

## Agent Roles & Definitions

[2026-02-04 23:16:56] [tags: workflow, prompt, orchestrator, roles]
Six canonical agent roles are defined in the THE_PROMPT_v5 system: Orchestrator, Research, Implementation, Review, Test, Debug. A seventh role, Monitor, is defined for status watching.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, cwd]
Orchestrator Agent CWD is the task folder (`~/run-agent/<project>/task-<id>/`). It has full tool access and read/write across the project. It must NOT do implementation work directly — spawn implementation agents instead.

[2026-02-04 23:16:56] [tags: workflow, research, cwd, constraint]
Research Agent is read-only. CWD is task folder (default) or project root (if specified by parent). Tools available: Read, Grep, Glob, WebFetch, WebSearch, Bash (read-only commands). No Edit/Write tools. No builds/tests.

[2026-02-04 23:16:56] [tags: workflow, implementation, cwd, constraint]
Implementation Agent CWD is project source root. Tools: Read, Edit, Write, Bash (for builds). Must use IntelliJ MCP Steroid for quality checks, build verification, and code inspections before commit.

[2026-02-04 23:16:56] [tags: workflow, review, constraint, quorum]
Review Agent is read-only (no Edit/Write). CWD is task folder or project root. Quorum of 2+ independent review agents required for non-trivial or multi-line changes. Trivial changes may be handled by single root-review agent.

[2026-02-04 23:16:56] [tags: workflow, review, quorum-decision]
Review quorum decision rule: 2+ approvals = APPROVED; 2+ rejections = CHANGES REQUESTED; mixed = NEEDS MORE REVIEW (spawn additional reviewer). Review agents must review independently (without reading other reviews first) then compare.

[2026-02-04 23:16:56] [tags: workflow, test, cwd]
Test Agent CWD is project source root. Tools: Read, Bash (for test execution). IntelliJ MCP Steroid preferred for running tests. Must report: total tests, passed, failed, coverage, execution time.

[2026-02-04 23:16:56] [tags: workflow, debug, cwd]
Debug Agent CWD is project source root. Tools: Read, Grep, Bash, Edit, Write (for fixes), IntelliJ MCP Steroid (for breakpoints and step-through). Must add regression test before committing fix.

[2026-02-04 23:16:56] [tags: workflow, monitor, cwd, constraint]
Monitor Agent CWD is project root or runs directory. No Edit/Write for source files (read-only). No process control — report issues only, do not kill/restart agents. May write monitoring output files.

---

## Workflow Stages (THE_PROMPT_v5.md — Required Development Flow)

[2026-02-04 23:03:05] [tags: workflow, stages, orchestrator]
The Required Development Flow defines 13 agent stages (Stage 0–12). Any failure must be logged to MESSAGE-BUS (and ISSUES.md if blocking). Flow restarts from beginning or from a root-selected stage as appropriate.

[2026-02-04 23:03:05] [tags: workflow, stage-0, cleanup]
Stage 0 (Cleanup): Read MESSAGE-BUS.md, AGENTS.md, Instructions.md, FACTS.md, ISSUES.md. Summarize/append; do not edit MESSAGE-BUS history. Ensure orchestration files are accessible.

[2026-02-04 23:03:05] [tags: workflow, stage-1, docs]
Stage 1 (Read local docs): Read AGENTS.md and all relevant .md files using absolute paths.

[2026-02-04 23:03:05] [tags: workflow, stage-2, research, parallelism]
Stage 2 (Research task with multi-agent context): Run at least two agents in parallel (research and implementation) to scope the task.

[2026-02-04 23:03:05] [tags: workflow, stage-3, task-selection]
Stage 3 (Select tasks): Choose actionable tasks based on IntelliJ MCP Steroid exploration; prefer low-hanging fruit first.

[2026-02-04 23:03:05] [tags: workflow, stage-4, quality-gate, tests]
Stage 4 (Select and validate tests/build): Pick relevant unit/integration tests and verify they pass in IntelliJ MCP Steroid; verify project builds. If no tests/builds apply, log N/A to MESSAGE-BUS.

[2026-02-04 23:03:05] [tags: workflow, stage-5, implementation]
Stage 5 (Implement changes and tests): Make code changes and add/update tests.

[2026-02-04 23:03:05] [tags: workflow, stage-6, quality-gate, intellij]
Stage 6 (IntelliJ MCP quality gate): Verify no new warnings/errors/suggestions in IntelliJ MCP Steroid.

[2026-02-04 23:03:05] [tags: workflow, stage-7, tests, quality-gate]
Stage 7 (Re-run tests in IntelliJ MCP): Re-run relevant tests in IntelliJ MCP Steroid.

[2026-02-04 23:03:05] [tags: workflow, stage-8, authorship]
Stage 8 (Research authorship and patterns): Use git annotate/blame and project review tools to identify maintainers and align with existing patterns.

[2026-02-04 23:03:05] [tags: workflow, stage-9, commit-review]
Stage 9 (Commit guideline review and cross-agent code review): Validate commit rules. Require quorum of at least two independent agents for non-trivial/multi-line changes.

[2026-02-04 23:03:05] [tags: workflow, stage-10, rebase, quality-gate]
Stage 10 (Rebase, rebuild, and tests): Squash or split into logical commits, rebase on latest main/master once workspace is clean, verify compilation in IntelliJ MCP Steroid, re-run related tests.

[2026-02-04 23:03:05] [tags: workflow, stage-11, push, code-review]
Stage 11 (Push, preflight, and code review): Push to feature branch, run project's preflight gate, create code review, log all links to MESSAGE-BUS.md.

[2026-02-04 23:03:05] [tags: workflow, stage-12, monitor]
Stage 12 (Monitor and apply fixes): Monitor preflight and review results, apply required fixes, log failures and restart flow as needed.

---

## Orchestrator Workflow Stages (THE_PROMPT_v5_orchestrator.md)

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-0-init]
Orchestrator Stage 0 (Initialize): Read context (THE_PROMPT_v5.md, AGENTS.md, etc.), assess situation (DONE/blockers), update TASK_STATE.md, post PROGRESS.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-1-plan]
Orchestrator Stage 1 (Plan Execution Strategy): Identify independent vs. dependent subtasks. Write focused prompt files with absolute paths, expected outputs, success criteria, and appropriate CWD.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-2-spawn, parallelism]
Orchestrator Stage 2 (Spawn Sub-Agents): Launch independent agents in parallel (max 16 concurrent). Record PIDs/run_ids. Post PROGRESS. Wait for prerequisites before spawning dependents.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-3-monitor]
Orchestrator Stage 3 (Monitor Progress): Poll message bus (10–30s). React to message types (PROGRESS, FACT, DECISION, QUESTION, ERROR, REVIEW).

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-4-converge]
Orchestrator Stage 4 (Convergence): Wait for all spawned agents to exit. Aggregate findings, verify implementations, collect reviews, check test results. Iterate if blockers remain.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-5-finalize]
Orchestrator Stage 5 (Finalize): Quality checks (artifacts, tests, builds, IntelliJ gate). Write output.md, create DONE file, post final FACT, exit code 0.

---

## Parallelism Limits

[2026-02-04 23:03:05] [tags: workflow, parallelism, limit]
Max parallel agents: 16. This limit applies at every delegation level. If the thread limit is reached, close completed agents and retry spawns.

[2026-02-04 23:16:56] [tags: workflow, parallelism, orchestrator]
Orchestrator should rotate agent types (claude, codex, gemini) for variety and track success rates.

---

## Quality Gates (Pre-Commit)

[2026-02-04 23:16:56] [tags: workflow, quality-gate, commit]
Before commit: 1. Run `go fmt` on all changed files; 2. Run `golangci-lint run` (zero warnings); 3. Run unit tests: `go test ./...` (all pass); 4. Run IntelliJ MCP Steroid quality check (no new warnings); 5. Verify builds: `go build ./...` (success).

[2026-02-04 23:16:56] [tags: workflow, quality-gate, pr]
Before PR: All quality gates passed; integration tests passed; multi-agent review approved (2+ agents); documentation updated; commit messages follow format; rebased on latest main.

---

## Validation Round 2 (gemini)

[2026-02-21 11:03:53] [tags: workflow, rlm, methodology]
RLM (Recursive Language Model) solves "context rot". Activation threshold: Context > 50K tokens; OR context > 16K AND multi-hop reasoning needed; OR files > 5; OR task spans > 3 subsystems.

[2026-02-21 11:03:53] [tags: workflow, rlm, six-step-protocol]
RLM Six-Step Protocol: 1. ASSESS (peek at scope); 2. DECIDE (choose strategy); 3. DECOMPOSE (split at boundaries); 4. EXECUTE (parallel sub-agents with `&` + `wait`); 5. SYNTHESIZE (merge outputs); 6. VERIFY (test/build).

[2026-02-21 11:03:53] [tags: workflow, rlm, environment]
RLM execution requires `--parent-run-id $JRUN_ID` for hierarchy tracking. Sub-agents run in parallel using background processes.

[2026-02-21 11:18:13] [tags: workflow, quality-gate, test-integrity]
Tests must be REAL. Fake tests (assert true, empty body, unjustified skip) are strictly forbidden. `make test-coverage` enforces minimum coverage (default 60%).

[2026-02-21 11:03:33] [tags: workflow, conductor-loop, constraints]
Conductor-loop specific: Read AGENTS.md before starting implementation. Never hard-code ports (use `:0`). Post PROGRESS at start, FACT on completion.

[2026-02-21 11:03:53] [tags: workflow, completion, done-file]
The `DONE` file (`touch "$TASK_FOLDER/DONE"`) signals the Ralph Loop to STOP restarting the task. Do NOT create if you want to be restarted (e.g. for context refresh). Never commit `DONE`.

[2026-02-21 11:03:53] [tags: workflow, messagebus, posting]
Message bus posting via `run-agent bus post`. Types: PROGRESS, FACT, DECISION, ERROR, QUESTION. `MESSAGE_BUS` env var is pre-configured.

[2026-02-21 11:03:53] [tags: workflow, environment, variables]
Env vars injected: `TASK_FOLDER`, `RUN_FOLDER`, `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID`, `MESSAGE_BUS`, `CONDUCTOR_URL`.

[2026-02-21 11:03:53] [tags: workflow, output-md, location]
Write `output.md` to `$RUN_FOLDER/output.md`. This is required for UI visibility and synthesis.

[2026-02-23 07:12:05] [tags: workflow, evolution, core-update]
Latest core update (2026-02-23) shipped task orchestration, self-update, and web UI reliability improvements, reinforcing the RLM and Conductor-specific workflows.
