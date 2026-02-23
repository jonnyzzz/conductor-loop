# FACTS: Prompts, Workflow & Methodology

Extracted from all revisions of THE_PROMPT_v5.md, role prompt files, AGENTS.md, CLAUDE.md, Instructions.md, contributing.md, testing.md, rlm-orchestration.md, and swarm legacy docs.

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
Stage 0 (Cleanup): Read MESSAGE-BUS.md, AGENTS.md, Instructions.md, FACTS.md, ISSUES.md. Summarize/append; do not edit MESSAGE-BUS history. Ensure orchestration files are accessible (THE_PROMPT_v5.md, role prompts, run-agent.sh).

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
Stage 11 (Push, preflight, and code review): Push to feature branch, run project's preflight gate (e.g. safe push), create code review, log all links to MESSAGE-BUS.md.

[2026-02-04 23:03:05] [tags: workflow, stage-12, monitor]
Stage 12 (Monitor and apply fixes): Monitor preflight and review results, apply required fixes, log failures and restart flow as needed.

---

## Orchestrator Workflow Stages (THE_PROMPT_v5_orchestrator.md)

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-0-init]
Orchestrator Stage 0 (Initialize): Read THE_PROMPT_v5.md, AGENTS.md, Instructions.md, THE_PLAN_v5.md, TASK.md, TASK_STATE.md, TASK-MESSAGE-BUS.md, PROJECT-MESSAGE-BUS.md. Assess situation (done/remaining/blockers). Update TASK_STATE.md and post PROGRESS.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-1-plan]
Orchestrator Stage 1 (Plan Execution Strategy): Identify independent vs. dependent subtasks. Write focused prompt files with absolute paths, expected outputs, success criteria, and appropriate CWD for each agent.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-2-spawn, parallelism]
Orchestrator Stage 2 (Spawn Sub-Agents): Launch independent agents in parallel (max 16 concurrent). Record PIDs and run_ids. Post PROGRESS per spawn. For sequential dependencies, wait for prerequisites before spawning.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-3-monitor]
Orchestrator Stage 3 (Monitor Progress): Poll message bus every 10–30 seconds. React to message types: PROGRESS → update state; FACT → check completion; DECISION → review/override; QUESTION → answer or escalate; ERROR → investigate/spawn debug agent; REVIEW → collect feedback.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-4-converge]
Orchestrator Stage 4 (Convergence): Wait for all spawned agents to exit. Aggregate findings, verify implementations, collect reviews, check test results. Iterate if blockers remain.

[2026-02-04 23:16:56] [tags: workflow, orchestrator, stage-5-finalize]
Orchestrator Stage 5 (Finalize): Quality checks (artifacts, tests, builds, IntelliJ gate). Write output.md, create DONE file, post final FACT, exit code 0.

---

## RLM Pattern (Recursive Language Model Decomposition)

[2026-02-21 11:03:53] [tags: workflow, rlm, methodology]
RLM (Recursive Language Model) solves "context rot" — the accuracy drop when a single agent must reason over a large context window. The fix is to treat context as a variable in an external environment and route each partition to a focused sub-agent.

[2026-02-21 11:03:53] [tags: workflow, rlm, activation-threshold]
Activate RLM when: Context > 50K tokens; OR context > 16K AND multi-hop reasoning needed; OR files > 5; OR task spans more than ~3 subsystems or requires >500 lines of changes.

[2026-02-21 11:03:53] [tags: workflow, rlm, six-step-protocol]
RLM Six-Step Protocol: 1. ASSESS — peek at file sizes/counts before reading; 2. DECIDE — match context size to strategy; 3. DECOMPOSE — split at natural boundaries; 4. EXECUTE — spawn sub-agents in parallel; 5. SYNTHESIZE — collect sub-agent outputs, merge and resolve conflicts; 6. VERIFY — run go test ./..., go build ./..., spot-check 2–3 claims.

[2026-02-21 11:03:53] [tags: workflow, rlm, assess]
ASSESS step: peek at scope BEFORE reading anything in full. Check file counts, directory sizes, LOC. Post a FACT with scope assessment to message bus.

[2026-02-21 11:03:53] [tags: workflow, rlm, decide, strategy]
RLM DECIDE strategy table: < 4K tokens → Single agent, read directly; 4K–50K tokens → Grep first, read matches only; > 50K tokens → Partition + parallel sub-agents; > 5 independent files → One sub-agent per file/group.

[2026-02-21 11:03:53] [tags: workflow, rlm, decompose, boundaries]
DECOMPOSE at semantic seams — by package, by subsystem, by concern. Good Go boundaries: one sub-agent per top-level package (`internal/runner`, `internal/api`, `pkg/storage`); or per feature concern; or per concern type (implementation, tests, docs). Target 4K–10K tokens per sub-task.

[2026-02-21 11:03:53] [tags: workflow, rlm, execute, parallel]
EXECUTE with background spawns using `&` then `wait`. Always pass `--parent-run-id $JRUN_ID` for hierarchy tracking in run-info.yaml. Use `--prompt-file` for complex instructions.

[2026-02-21 11:03:53] [tags: workflow, rlm, synthesize, strategies]
SYNTHESIZE strategies: Concatenate (independent findings, order matters); Merge+deduplicate (overlapping code paths); Vote (classification, e.g. "is this a bug?"); Reduce (counting: error counts, coverage numbers).

[2026-02-21 11:03:53] [tags: workflow, rlm, verify, checklist]
VERIFY checklist: original task fully addressed? All partitions contributed output? No coverage gaps (spot-check 2–3 claims)? Tests pass (go test ./...)? Build clean (go build ./...)?

[2026-02-21 11:03:53] [tags: workflow, rlm, anti-patterns]
RLM Anti-patterns: Reading all files before deciding; spawning > 10 sub-agents at once (group into 3–6 partitions); giving sub-agents no shared context; splitting at arbitrary byte offsets; omitting --parent-run-id; skipping `wait`; merging without deduplication.

---

## Parallelism Limits

[2026-02-04 23:03:05] [tags: workflow, parallelism, limit]
Max parallel agents: 16. This limit applies at every delegation level. If the thread limit is reached, close completed agents and retry spawns.

[2026-02-04 23:16:56] [tags: workflow, parallelism, orchestrator]
Orchestrator should rotate agent types (claude, codex, gemini) for variety and track success rates to adjust distribution. Load-balance across agent types.

[2026-02-04 23:16:56] [tags: workflow, parallelism, independence]
Use parallelism to separate research, implementation, review, and testing. Do not parallelize dependent subtasks — wait for prerequisites before spawning dependent agents.

---

## Quality Gates (Pre-Commit)

[2026-02-04 23:16:56] [tags: workflow, quality-gate, commit]
Before commit: 1. Run `go fmt` on all changed files; 2. Run `golangci-lint run` (zero warnings); 3. Run unit tests: `go test ./...` (all pass); 4. Run IntelliJ MCP Steroid quality check (no new warnings); 5. Verify builds: `go build ./...` (success).

[2026-02-04 23:16:56] [tags: workflow, quality-gate, pr]
Before PR: All quality gates passed; integration tests passed; multi-agent review approved (2+ agents); documentation updated; commit messages follow format; rebased on latest main.

[2026-02-21 11:18:13] [tags: workflow, quality-gate, test-integrity]
Tests must be REAL. A test is fake if it: asserts trivially-true statements; mocks away the unit under test so only the mock is exercised; has an empty test body; uses t.Skip() without documented legitimate reason; catches an expected error and discards it silently.

[2026-02-21 11:18:13] [tags: workflow, quality-gate, test-integrity]
If a scenario is hard to test, make the code more testable. Do not fake the test. Reviewers must reject PRs with fake tests. `make test-coverage` enforces minimum 60% threshold (configurable via COVERAGE_THRESHOLD env).

[2026-02-04 23:16:56] [tags: workflow, quality-gate, implementation]
Implementation agent must: run tests with race detector (`go test -race ./...`); verify no new IntelliJ warnings; check builds succeed; review changes for completeness. Must NOT commit with failing tests, lint warnings, or build errors.

---

## Commit Format Requirements

[2026-02-04 23:16:56] [tags: workflow, commit-format, conventions]
Commit format: `<type>(<scope>): <subject>` with optional body and footer. Subject: max 50 chars, imperative mood, lowercase first letter, no period.

[2026-02-04 23:16:56] [tags: workflow, commit-format, types]
Valid commit types: feat (new feature), fix (bug fix), refactor (no behavior change), test (adding/updating tests), docs (documentation), chore (maintenance), perf (performance improvements), style (formatting only), ci (CI/CD changes).

[2026-02-04 23:16:56] [tags: workflow, commit-format, scopes]
Valid commit scopes (subsystem names): agent, runner, storage, messagebus, config, api, ui, test, ci.

[2026-02-04 23:16:56] [tags: workflow, commit-format, guidelines]
Commit guidelines: Keep commits atomic (one logical change per commit). Use present tense. Include ticket/issue references when applicable. Squash WIP commits before PR. Rebase on main before pushing.

[2026-02-21 11:03:53] [tags: workflow, commit-format, conductor-loop]
In conductor-loop: Post a FACT message for every commit with the commit hash. Use git commits as information handoff between agents in addition to message bus.

---

## Message Bus Protocol

[2026-02-04 23:16:56] [tags: workflow, messagebus, types]
Message types: FACT (concrete results: tests, commits, links), PROGRESS (in-flight status), DECISION (choices and policy updates), REVIEW (structured feedback), ERROR (failures blocking progress), QUESTION (questions requiring human or agent response), INFO, WARNING, OBSERVATION, ISSUE.

[2026-02-04 23:16:56] [tags: workflow, messagebus, protocol]
Agents MUST post progress updates to MESSAGE-BUS. Post at minimum: PROGRESS at start of each major step; FACT for every concrete outcome (commits, test results, key file paths); ERROR when blocked (include error text and attempted remediation).

[2026-02-04 23:16:56] [tags: workflow, messagebus, scopes]
Two message bus scopes: TASK-MESSAGE-BUS.md for task-scoped updates; PROJECT-MESSAGE-BUS.md for cross-task facts. Task messages stay in task scope; project messages are project-wide; UI aggregates at read time.

[2026-02-04 23:16:56] [tags: workflow, messagebus, mechanics]
Message bus: append-only file per scope. Atomic writes via run-agent bus (temp + swap). Direct writes disallowed. Body soft limit: 64KB; larger payloads stored as attachments in task folder with timestamp + short description naming.

[2026-02-04 23:16:56] [tags: workflow, messagebus, threading]
Message bus threading: parents[] links supported. Format: YAML front-matter entries separated by `---`. Required headers: msg_id, ts (ISO-8601 UTC), type, project. Optional: task/run_id/parents/attachment_path.

[2026-02-21 11:03:53] [tags: workflow, messagebus, posting]
Post message using `run-agent bus post`. MESSAGE_BUS env var is pre-configured; no --bus flags needed. For project-aware posting, specify --project and --root. Fallback: POST via HTTP API to CONDUCTOR_URL.

---

## Agent Execution & Traceability

[2026-02-04 23:03:05] [tags: workflow, traceability, runner]
All agent runs must use the unified runner script (./run-agent.sh or `run-agent job`). Runner creates a new run folder, enforces consistent file names, writes prompt/logs/artifacts under the same run folder.

[2026-02-04 23:03:05] [tags: workflow, traceability, artifacts]
Required run folder artifacts: prompt.md, agent-stdout.txt, agent-stderr.txt, cwd.txt (with EXIT_CODE= after completion), pid.txt (while running), run-info.yaml (runner-managed, agents must NOT modify).

[2026-02-04 23:03:05] [tags: workflow, traceability, status-check]
Status check: while pid.txt present → agent running (verify with `ps -p <pid>`). After completion → pid.txt removed; check EXIT_CODE= in cwd.txt. If background start produces no logs within 30s → re-run in foreground.

[2026-02-21 11:03:53] [tags: workflow, traceability, parent-tracking]
Always pass `--parent-run-id $JRUN_ID` when spawning sub-agents so parent-child hierarchy is tracked in run-info.yaml and visible in web UI.

---

## Environment Variables

[2026-02-04 23:16:56] [tags: workflow, environment, variables]
Environment variables injected into agent processes: TASK_FOLDER (absolute path to task dir), RUN_FOLDER (absolute path to current run dir), JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID, JRUN_PARENT_ID (if sub-agent), MESSAGE_BUS (absolute path to TASK-MESSAGE-BUS.md), RUNS_DIR, CONDUCTOR_URL (conductor REST API URL).

[2026-02-04 23:16:56] [tags: workflow, environment, constraint]
Agents must NOT manipulate JRUN_* environment variables. Error messages must not instruct agents to set env vars. run-agent binary prepends its own location to PATH for child processes.

[2026-02-04 23:16:56] [tags: workflow, environment, paths]
All file references must use absolute paths (not `~`). Path normalization uses OS-native Go filepath.Clean. Agents should use $JRUN_PROJECT_ID, $JRUN_TASK_ID, etc. in shell commands.

---

## Completion Protocol

[2026-02-21 11:03:53] [tags: workflow, completion, protocol]
Completion sequence: 1. Write summary to $RUN_FOLDER/output.md; 2. Run all tests (go build ./... and go test ./...); 3. Commit changes following AGENTS.md format; 4. Create DONE file: `touch "$TASK_FOLDER/DONE"`.

[2026-02-21 11:03:53] [tags: workflow, completion, done-file]
DONE file signals Ralph loop NOT to restart the task. Do NOT create DONE if you want to be restarted (e.g. ran out of context mid-task). Never commit the DONE file — it is a runtime signal, not a source file.

[2026-02-21 11:03:53] [tags: workflow, completion, ralph-loop]
Ralph Loop: when DONE is absent, Ralph restarts root agent. When DONE exists but children are running, Ralph waits for children (PGID tracking + 300s timeout), then restarts root agent one final time for result aggregation. Run chain tracked via previous_run_id on each restart.

---

## Task ID Format

[2026-02-21 11:03:53] [tags: workflow, task-id, format]
Task ID format: `task-<YYYYMMDD>-<HHMMSS>-<slug>`. Example: `task-20260220-190000-my-feature`. Omit --task flag to get an auto-generated valid ID. Invalid formats cause immediate exit with code 1.

---

## Sub-Agent Spawning

[2026-02-21 11:03:53] [tags: workflow, spawning, command]
Sub-agent spawn command:
```bash
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID" \
  --parent-run-id "$JRUN_ID" \
  --agent claude \
  --cwd /path/to/working/dir \
  --prompt "Sub-task description"
```

[2026-02-21 11:03:53] [tags: workflow, spawning, parallel-pattern]
Parallel spawn pattern: use `&` after each spawn command, then `wait` to collect all results before synthesizing. Never synthesize before calling `wait`.

[2026-02-21 11:03:53] [tags: workflow, spawning, timeout]
Use `--timeout <duration>` to limit sub-agent runtime (e.g. `--timeout 30m`). Prevents runaway agents from blocking the parent.

[2026-02-21 11:03:53] [tags: workflow, spawning, prompt-file]
Use `--prompt-file <path>` for complex prompts that don't fit inline. Always use absolute paths for all .md file references inside prompts so sub-agents do not need to search.

---

## Root Agent Constraints

[2026-02-04 23:03:05] [tags: workflow, root-agent, constraint]
Root agent must not modify target codebase directly if it is outside PROJECT_ROOT. Use sub-agents with CWD set to the target repo. Root agent orchestrates; sub-agents do codebase work.

[2026-02-04 23:03:05] [tags: workflow, root-agent, planning]
For planning: run 5–10 iterations to converge on the task plan; log each iteration outcome to MESSAGE-BUS.md. For review planning: run 5–10 iterations to converge on the review plan.

[2026-02-04 23:03:05] [tags: workflow, root-agent, conflict-resolution]
If contradiction found across docs, the Required Development Flow overrides. Start a research agent, decide based on project specifics, log options to MESSAGE-BUS.md.

---

## File Access Policies

[2026-02-04 23:16:56] [tags: workflow, file-access, agents]
All agents: Full read access to project files. Read TASK_STATE.md and message bus on startup. Use absolute paths when referencing files.

[2026-02-04 23:16:56] [tags: workflow, file-access, write-restrictions]
Write access: Implementation/Test/Debug agents → source files, test files. All agents → message bus, output.md in own run folder. Root agents only → TASK_STATE.md (direct write). No agents → run-info.yaml (runner-managed only), .git/ directory (use git commands only).

[2026-02-04 23:16:56] [tags: workflow, file-access, restricted]
Agents must NOT modify: other agents' run folders; run-info.yaml files; .git/ directory; global config (~/.run-agent/config.hcl).

---

## TASK_STATE.md Protocol

[2026-02-04 23:16:56] [tags: workflow, task-state, format]
TASK_STATE.md format: # Task State: <Name>, Last Updated timestamp, Status (In Progress/Blocked/Complete), Current Phase, Completed items, In Progress items (with agent run_id and PID), Pending items, Blockers, Next Steps.

[2026-02-04 23:16:56] [tags: workflow, task-state, update-frequency]
Update TASK_STATE.md: after spawning agents; after receiving significant messages; after agents complete; before exiting. This enables resumed agents to pick up where the previous left off.

---

## Testing Standards

[2026-02-04 23:16:56] [tags: workflow, testing, coverage]
Coverage targets: Overall > 80% line coverage. Per package: internal/storage/ > 90%, internal/messagebus/ > 90%, internal/config/ > 85%, internal/agent/ > 80%, internal/runner/ > 85%, internal/api/ > 80%.

[2026-02-04 23:16:56] [tags: workflow, testing, mandatory-policy]
NEVER skip tests. All tests must pass before committing. No exceptions. If a test fails, fix the code — do not skip or exclude the test. Tests marked t.Skip() require explicit documented justification.

[2026-02-04 23:16:56] [tags: workflow, testing, port-policy]
Never hard-code ports. Use `:0` for test listeners. For HTTP test servers: `net.Listen("tcp", ":0")` then read assigned port. Prefer `httptest.NewServer`/`httptest.NewUnstartedServer` for HTTP test servers.

[2026-02-04 23:16:56] [tags: workflow, testing, patterns]
Test patterns: Table-driven tests preferred. Race detector required for concurrency tests (`go test -race`). Test types: unit (fast < 1s total, isolated), integration (1–10s, real filesystem, no network), E2E (10–60s, full stack), performance (benchmarks).

[2026-02-04 23:16:56] [tags: workflow, testing, naming]
Test naming convention: `Test{FunctionName}_{Scenario}`. Examples: TestCreateRun_ValidInputs, TestAppendMessage_ConcurrentWrites.

---

## Code Style (Go)

[2026-02-04 23:16:56] [tags: workflow, code-style, go]
Go style: Language Go 1.21+. Formatting: gofmt exclusively. Linting: golangci-lint recommendations. Package naming: lowercase, single-word. Error handling: always check errors, use errors.Wrap() for context. Files: max 500 lines, split by functionality.

[2026-02-04 23:16:56] [tags: workflow, code-style, naming]
Naming conventions: Interfaces → noun or noun phrase (Agent, MessageBus, Storage). Functions → verb or verb phrase (StartAgent, WriteMessage). Constants → CamelCase exported, camelCase unexported. Files → lowercase_with_underscores.go.

[2026-02-04 23:16:56] [tags: workflow, code-style, imports]
Import organization: three groups: 1. Standard library, 2. Third-party packages, 3. Project packages. Empty lines between groups.

[2026-02-04 23:16:56] [tags: workflow, code-style, patterns]
Implementation patterns: Atomic file write via temp+rename. File locking via flock() with timeout. Error wrapping with `fmt.Errorf("context: %w", err)`. Process spawning with Setsid: true in SysProcAttr.

---

## Monitoring Strategy

[2026-02-04 23:03:05] [tags: workflow, monitoring, scripts]
Monitoring options: watch-agents.sh (60-second polling), monitor-agents.sh (10-minute polling, background via nohup), monitor-agents.py (live console with [run_xxx] prefixes and colored compact header). All read <RUNS_DIR>.

[2026-02-04 23:16:56] [tags: workflow, monitoring, stall-detection]
Stall detection: agent running > timeout without log activity > 5 minutes = stalled. Check last modification time of agent-stdout.txt. SIGTERM → 30s grace → SIGKILL for stuck agents.

[2026-02-04 23:16:56] [tags: workflow, monitoring, issue-types]
Monitor detects: stalled runs (running > timeout, no progress), failed runs (non-zero exit codes), no output (no logs after expected time), orphaned processes (pid.txt exists but process dead), missing artifacts (expected files not created).

---

## Swarm Design Principles (Legacy Ideas)

[2026-02-04 23:16:51] [tags: workflow, swarm, principles]
Core swarm manifesto: agents do a selected task and exit; tasks are recursive; each agent decides if it can work directly or must delegate smaller tasks down the hierarchy; all agent runs are persisted and tracked; agents communicate via MESSAGE-BUS; parent-child relationships tracked in runs.

[2026-02-04 23:16:51] [tags: workflow, swarm, architecture]
Swarm architecture principle: monitoring web UI is only for observing, never for controlling or depending on. The system must work entirely without the monitoring process. Monitoring is fully optional.

[2026-02-04 23:16:51] [tags: workflow, swarm, delegation]
Each folder is owned by its own agent; always delegate, never touch another agent's folder directly. Agent processes are detached from parent terminal (sub-agents survive parent agent death). Explicit stop command required to kill sub-agents.

[2026-02-04 23:16:51] [tags: workflow, swarm, restart]
When task is restarted, prepend "Continue working on the following:" to the prompt. Agent state is maintained in TASK_STATE.md so each new agent starts from there.

[2026-02-04 23:16:51] [tags: workflow, swarm, max-depth]
Limit of 16 agents at any delegation depth. Recursion is allowed; each agent may decide to delegate down to sub-agents for their sub-task.

---

## Conductor-Loop Specific Constraints

[2026-02-21 11:03:33] [tags: workflow, conductor-loop, constraints]
Conductor-loop constraints: Read AGENTS.md before starting any implementation work. Run `git log --oneline -- <file>` before editing any file. Never hard-code ports. All tests must pass — do not skip tests. Post PROGRESS at start, FACT on completion. Keep commits atomic (one logical change per commit).

[2026-02-21 11:03:33] [tags: workflow, conductor-loop, conductor-url]
When CONDUCTOR_URL is set, agents can query the REST API: list projects, get task status, read Prometheus metrics at /metrics endpoint.

[2026-02-21 11:03:33] [tags: workflow, conductor-loop, root-server]
Conductor server root directory: use `--root ./runs` (NOT the project root). Required for correct stats/message-bus API behavior.

---

## Output.md Format

[2026-02-04 23:16:56] [tags: workflow, output-md, format]
output.md required sections: Status (Complete/Failed/Partial), Duration, Agents Spawned, Summary (brief), Results (key outcomes), Agents Run (list with run IDs and descriptions), Artifacts (files modified/created/docs updated), Issues, Next Steps.

[2026-02-21 11:03:53] [tags: workflow, output-md, location]
Write output.md to `$RUN_FOLDER/output.md`. The exact path is given in the prompt preamble ("Write output.md to ..."). This file is shown in web UI OUTPUT tab in real time.

---

## References Hierarchy (Boot Order)

[2026-02-04 23:03:05] [tags: workflow, references, boot-order]
New agents should begin by reading in this order: 1. THE_PROMPT_v5.md (primary entry point), 2. AGENTS.md, 3. Instructions.md, 4. THE_PLAN.md or THE_PLAN_v5.md, 5. Project development guide (e.g. DEVELOPMENT-GUIDE.md).

[2026-02-04 23:03:05] [tags: workflow, references, precedence]
Precedence (high to low): Required Development Flow > project-specific AGENTS/Instructions (only when explicitly conflicting) > THE_PROMPT_v5.md > Standard Workflow template. Unresolved conflicts: research + logged DECISION.

---

## Prompt Creation Rules

[2026-02-04 23:03:05] [tags: workflow, prompt, creation]
When creating a run prompt.md: copy the relevant role file verbatim and append task-specific instructions. Always use absolute paths for all .md file references inside prompts. If role prompt files don't exist, create them from base template.

[2026-02-04 23:16:56] [tags: workflow, prompt, delegation]
Delegation prompt writing rules: be specific (clear objective, scope, expected output); provide context with absolute paths; set CWD appropriate for agent type; define concrete success criteria; include references to relevant specs, decisions, prior work.

---

## Agent Selection Strategy

[2026-02-04 23:03:05] [tags: workflow, agent-selection, strategy]
Root agent selects agent type (Codex/Claude/Gemini) at random unless statistics strongly indicate a better choice. Codex is preferred backend for Implementation agents. Agent selection may consult message bus history.

[2026-02-04 23:16:56] [tags: workflow, agent-selection, rotation]
Agent selection: round-robin by default; "I'm lucky" random with weights; rotate agent types for variety across tasks. Track success rates and adjust distribution. Backend failures: transient → exponential backoff (1s, 2s, 4s; max 3 tries); auth/quota → fail fast.

---

## Document Evolution Notes

[2026-02-04 23:03:05] [tags: workflow, evolution, v5-origin]
THE_PROMPT_v5.md has a single commit (2026-02-04 23:03:05). All subsequent role-specific prompt files (orchestrator, research, implementation, review, test, debug, monitor) were added in the same commit (2026-02-04 23:16:56) under "docs(bootstrap): Add role-specific prompt files".

[2026-02-21 11:03:53] [tags: workflow, evolution, conductor-edition]
THE_PROMPT_v5_conductor.md was introduced in 2026-02-21 as the conductor-loop-specific orchestration edition — a streamlined overlay for agents running inside the Ralph loop, referencing THE_PROMPT_v5.md for the general methodology.

[2026-02-21 11:03:53] [tags: workflow, evolution, rlm-guide]
docs/user/rlm-orchestration.md was introduced in commit 45cee97c (2026-02-21 20:19:09) as a concrete operational guide translating the RLM methodology into specific run-agent job and run-agent bus commands.

[2026-02-21 11:18:13] [tags: workflow, evolution, test-integrity]
Test integrity rules were added to CLAUDE.md in commit 350e2993 (2026-02-21 11:18:13) with the specific definition of "fake test" and the make test-coverage enforcement mechanism.
