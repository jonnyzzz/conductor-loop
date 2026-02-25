# FACTS: Swarm Ideas & Legacy Design

Extracted from: docs/swarm/docs/legacy/ (migrated from jonnyzzz-ai-coder)
Earliest ideas.md commit: 729df20 (2026-01-29) — "docs(message-bus-mcp): Add final review and swarm ideas"
Legacy files imported to conductor-loop: 283157b (2026-02-21 17:36:06)

---

## Core Vision & Manifesto

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, vision]
Agents are better at smaller, focused tasks. The longer an agent works and the more prompts it receives, the harder it is to maintain original attention and goals. The swarm design builds work as a chunk of recursive tasks to address this.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, manifesto]
Swarm manifesto: Agent should do a selected task and exit. Task can be of any kind (coding, managing). Work is built as recursive tasks. On each level an agent must decide if it can work on the task or delegate smaller tasks down the hierarchy.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, vision]
Root idea of the SWARM: Agents control everything. Each agent run is persisted and tracked. Agents use MESSAGE-BUS to communicate. Each agent must decide if the task is small enough to work on, otherwise it delegates down recursively. Parent-child relationships are tracked in runs.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, vision]
Original constraint: There should be no direct communication with agents. Instead, one can only write to the message bus. The message bus is the only interface for user interaction with agents.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, vision]
State persistence requirement: The current state of the task is persisted by the agent in a STATE.md file in the task folder, so each new started agent picks up from there. (Later renamed TASK_STATE.md.)

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, monitoring]
Monitoring app should be react/web based to see the tree: 1/3 screen shows tree progress, left 1/5 is message bus view, down below all agents output colored per agent, all done in JetBrains Mono font.

---

## Core Components (Original Design)

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, components]
Three components of the system: (1) run-agent binary to start agents, (2) start-task.sh to ask for a task and start it, (3) monitoring tool using disk layout and message-bus only (web-ui).

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, architecture]
Assumptions: Message bus MCP is added to all agents. All agent CLIs are configured to run out of the box with tokens provided. The entire system runs on the same machine.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, architecture]
System inputs: Generic prompts to start and initiate the work. Each task goes to a specific project/task folder with an initial TASK.md prompt. TASK prompt includes recommended improvement iterations.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, agent-management]
Agent rotation principle: Run different agent types each time to keep up the work. The system should be resilient, regularly checking if the root agent is running and restarting if needed.

---

## The Ralph Loop (Original)

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, ralph]
Ralph-like restarts: Agent may quit before whole work is done; this is where Ralph-like restarts are used. The task system (run-agent task / start-task.sh) uses a "while true, basically ralph" loop to restart the root agent.

[2026-02-01 14:47:16] [tags: swarm, idea, legacy, ralph]
Ralph loop design: The essence of run-task.sh is while true—basically Ralph. The main prompt starts the agent to manage the swarm. The system regularly checks if the root agent is running and restarts it if needed.

---

## Storage Layout (Original Vision)

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, storage]
Original layout rooted at ~/run-agent: project folder with PROJECT-MESSAGE-BUS.md, FACT files per date-time, home-folders.md for project info, task-<date-time>-<name> subfolders each with TASK-MESSAGE-BUS.md, TASK_STATE.md, and a runs/ subdirectory.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, storage]
Run directory structure: Each run under <runId>-<date-time> contains: parent-run-id file, prompt.md, output.md, agent-type, cwd, agent process pid and commandline.

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, storage]
FACT files: Agent should write facts per file with YAML front matter. Facts can be promoted from task level to project level, and from project to global location.

---

## Message Bus Design

[2026-01-29 11:29:30] [tags: swarm, idea, legacy, message-bus]
MESSAGE-BUS files must be per-task and per-project to avoid mixing. Use MESSAGE-BUS as the way to interact with agents. Users can post comments to message bus. All agent start/stop events should be included in message bus updates.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, message-bus, decisions]
Message bus format decision: Append-only YAML front-matter entries separated by "---" (no legacy single-line compatibility). Required headers: msg_id, ts (ISO-8601 UTC), type, project; optional: task/run_id/parents/attachment_path.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, message-bus, decisions]
Message types defined: FACT, QUESTION, ANSWER, USER, START, STOP, ERROR, INFO, WARNING, OBSERVATION, ISSUE.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, message-bus, decisions]
Message routing: Separate PROJECT-MESSAGE-BUS.md and TASK-MESSAGE-BUS.md. Task messages stay in task scope. Project messages are project-wide. UI aggregates at read time.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, message-bus, decisions]
Message size limit: Soft 64KB body limit; larger payloads stored as attachments in the task folder with timestamp + short description naming and attachment_path metadata.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, message-bus, decisions]
Atomic writes: Writes via run-agent bus use temp + atomic rename swap. Direct writes disallowed. O_APPEND + flock (POSIX) chosen after all 3 agent reviewers agreed on this approach.

---

## run-agent Binary Design

[2026-01-31 22:41:02] [tags: swarm, idea, legacy, run-agent]
Idea to merge all tools into one Go binary: "run-agent serve" to start web UI, "run-agent task" to start a task, "run-agent job" to run an agentic job, "run-agent bus" to deal with message bus. All commands prefer user home location for data storage.

[2026-01-31 22:41:02] [tags: swarm, idea, legacy, run-agent]
run-agent binary must put itself to the PATH for its sub-processes, putting itself at the front. Must ensure it's not already included from the parent process.

[2026-01-31 22:41:02] [tags: swarm, idea, legacy, config]
Configuration: ~/run-agent/config.hcl (HCL format) to configure: projects folder location (default ~/run-agent/), deployment ssh key, other sensible parameters, list of supported agents. Per-project overrides are optional future work.

[2026-01-31 22:41:02] [tags: swarm, idea, legacy, run-agent]
Environment variables contract: run-agent asserts it has JRUN_TASK_ID, JRUN_PROJECT_ID, JRUN_ID environment vars set. Tracks parent-child relation between runs (creating the tree).

[2026-02-04 10:53:50] [tags: swarm, idea, legacy, run-agent]
run-agent bus post commands include type, message, task, project. run-agent bus poll blocks and waits for new messages as the file grows, with --wait option and integration of project and task level messages.

---

## Agent Environment Contract

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, env-contract, decisions]
Environment variable contract: Runner sets JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID, JRUN_PARENT_ID internally. Agents must not rely on them being settable from outside. Error messages must not instruct agents to set env vars. Agents should not manipulate JRUN_* vars.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, env-contract, decisions]
Task/run paths injected via prompt preamble (JRUN_RUN_FOLDER in prompt text, not env var). Path normalization uses OS-native Go filepath.Clean.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, env-contract, decisions]
Signal handling decided: SIGTERM → 30 second grace period → SIGKILL. Environment inheritance: full inheritance in MVP (no sandbox). Date/time: NOT injected; agents access system time themselves.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, env-contract, decisions]
Security note (original): Agent should not be able to manipulate environment variables. Example from Codex changing MESSAGE-BUS variable caused mixed messages between tasks. Env var injection protection is critical.

---

## Agent Protocol & Governance

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, governance, decisions]
Max delegation depth = 16 (configurable). Each folder is owned by its own agent. Always delegate, never touch another agent's scope directly.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, governance, decisions]
CWD guidance: Root in task folder; code-change agents in project root; research/review agents default to task folder. No enforced sandbox, sensitive-path guardrails, or resource limits in MVP. Scripts are allowed; cross-project access is not blocked.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, governance, decisions]
Git safety is guidance only (touch only selected files). Agents seem to be clumsy with Git; "Git Pro" behavior is expected. Agents should commit only selected files without touching other files.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, governance, decisions]
Root agent polls and processes message bus updates in MVP. No dedicated poller/heartbeat process in MVP. A dedicated process can be started explicitly to analyze and promote data from task to project.

---

## Technology Stack Decisions

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, stack, decisions]
Implementation language: Go (single binary distribution) for backend and all utilities. Monitoring UI: TypeScript + React; built via npm/package.json + webpack; assets embedded in Go binary for MVP (go:embed).

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, stack, decisions]
UI framework decision: TypeScript + React SPA using JetBrains Ring UI and JetBrains Mono font. Installation: "npm install @jetbrains/ring-ui --save-exact". 50+ React controls available.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, stack, decisions]
Config format: HCL (HashiCorp Configuration Language) at ~/run-agent/config.hcl. Library: github.com/hashicorp/hcl/v2 with hclsimple for direct struct loading, gohcl for struct tags.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, stack, decisions]
Streaming technology: SSE (Server-Sent Events) chosen over WebSockets for run-agent serve. Reason: server-to-client only (monitoring UI reads from backend), automatic reconnection with Last-Event-ID, simpler implementation, HTTP/2 compatible. 2s polling fallback for browsers without SSE.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, stack, decisions]
Web UI runs read-only for MVP, localhost only. Layout: tree ~1/3, message bus ~1/5, output pane bottom. Projects are roots; order by last activity. State management: React Context + hooks (no Redux/Zustand in MVP).

---

## Agent Backend Integrations

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, backends, decisions]
Agent backend environment variable mappings (hardcoded in runner): Codex → OPENAI_API_KEY, Claude → ANTHROPIC_API_KEY, Gemini → GEMINI_API_KEY, Perplexity → PERPLEXITY_API_KEY. All support @file reference for token file paths.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, backends, decisions]
Claude CLI flags: "claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions". Codex: "codex exec --dangerously-bypass-approvals-and-sandbox -C $CWD -". Gemini: "gemini --screen-reader true --approval-mode yolo".

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, backends, decisions]
Perplexity is a native REST-backed agent (not CLI). Supports SSE streaming via stream=True parameter. All Perplexity models support streaming (sonar-pro, sonar-reasoning, sonar-deep-research, r1-1776). Citations arrive at the end of stream.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, backends, decisions]
xAI backend integration: Deferred to post-MVP. Tracked in docs/dev/issues.md backlog.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, backends, decisions]
Agent selection: Round-robin by default. "I'm lucky" random selection with weights. Selection may consult message bus history.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, backends, decisions]
Backend failure handling: Transient errors use exponential backoff (1s, 2s, 4s; max 3 tries). Auth/quota fail fast. No proactive credential validation.

---

## Storage & Data Layout

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, storage, decisions]
Layout rooted at ~/run-agent/<project>/task-<timestamp>-<slug>/runs/<run_id>/. config.hcl at ~/run-agent/. Run ID format: YYYYMMDD-HHMMSSMMMM-PID (millisecond precision).

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, storage, decisions]
run-info.yaml is canonical run metadata (lowercase keys). Schema version 1 with 21 required fields: identity, lineage, agent, timing, paths. Optional fields: backend metadata, command line.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, storage, decisions]
TASK_STATE.md is free text; DONE is completion marker (empty file). TASK_STATE updates use temp+rename (root only). output.md is the final agent response; stdout/stderr are raw streams.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, storage, decisions]
FACT files require YAML front matter. home-folders.md is YAML with explanations. Message bus stored as single append-only file per scope. No symlinks/hardlinks; no size limits enforced in MVP.

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, storage, decisions]
UTF-8 encoding: Strict UTF-8 without BOM for all text files.

---

## Problem Decisions (Critical Blockers Resolved)

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-1, message-bus, decisions]
Problem #1: Message Bus Race Condition. Resolved with O_APPEND + flock (POSIX atomic append). All 3 agent reviewers (Claude, Codex, Gemini) unanimously agreed on this approach. msg_id format: MSG-YYYYMMDD-HHMMSS-NNNNNNNNN-PIDXXXXX-SSSS (timestamp + nanoseconds + PID + sequence counter).

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-2, ralph, decisions]
Problem #2: Ralph Loop DONE + Children Running. Decision: "Wait Without Restart" approach. When DONE exists and children are running: (1) Wait for all children to exit with 300s timeout, (2) Do NOT restart root agent, (3) Declare task complete once all children exit. DONE means root declared completion; restarting would be semantically incorrect.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-2, ralph, decisions]
Ralph loop algorithm: Check DONE before starting root agent. If DONE + no children → task complete. If DONE + active children → wait (poll PGID with kill(-pgid, 0)), timeout after 300s, then complete. If no DONE → start/restart root agent (subject to max_restarts). Between restart attempts, pause 1s to avoid tight loops.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-3, storage, decisions]
Problem #3: run-info.yaml Update Race Condition. Resolved: Updates use atomic rewrite pattern (temp write + atomic rename). UI must tolerate partial reads during transition.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-4, message-bus, decisions]
Problem #4: msg_id Collision. Resolved by Problem #1's comprehensive msg_id solution. Format: MSG-YYYYMMDD-HHMMSS-NNNNNNNNN-PIDXXXXX-SSSS guarantees uniqueness through: atomic counter for same-process same-nanosecond, PID for cross-process on same machine, nanosecond precision for cross-time.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-5, output, decisions]
Problem #5: output.md Creation Responsibility. Decision: Approach A (Runner Fallback). Agent responsibility: agents *should* write output.md if possible (best-effort). Runner responsibility: If output.md does not exist after agent terminates, Runner MUST create it using content of agent-stdout.txt.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-6, perplexity, decisions]
Problem #6: Perplexity Output Double-Write. Resolved: Perplexity adapter writes both stdout and output.md sequentially. All backends now follow same pattern: agent tries, runner fallback.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-7, process, decisions]
Problem #7: Process Detachment vs Wait Contradiction. Clarification: "Detach" means setsid() — creating new process group/session, NOT daemonization. Effect: child is detached from parent's controlling terminal (no CTRL-C signals), but parent can still waitpid() and collect exit code. Implementation: cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, problem-8, sse, decisions]
Problem #8: SSE Stream Run Discovery. Resolved: Backend discovers new run folders via polling (filesystem scan on interval). New runs automatically included in log stream without explicit notification.

---

## Agent Design Patterns (Documented)

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, patterns]
Pattern A (Recommended): Parallel Delegation with Aggregation. Root spawns N children → monitors message bus for CHILD_DONE messages → waits for all children → aggregates results → writes output.md → writes DONE → exits. Key: root writes DONE only after children complete.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, patterns]
Pattern B: Fire-and-Forget Delegation. Root spawns N children for independent subtasks → writes DONE immediately (root's work is done) → exits. Ralph loop waits for children to exit before completing task. Used when root does not need children's results.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, patterns]
Anti-pattern (DO NOT USE): Root spawns children, writes DONE immediately, expects to be restarted to aggregate results. This is incorrect — root should wait for children BEFORE writing DONE if aggregation is needed.

---

## Subsystems Registry (Final Design, 8 Total)

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 1 — Runner & Orchestration: run-agent binary (Go) + task/job/serve/bus/stop commands, Ralph restart loop, run linking, idle/stuck handling, agent selection/rotation (round-robin/weighted), config.hcl schema/validation, stop/kill with SIGTERM/SIGKILL.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 2 — Storage & Data Layout: ~/run-agent layout, run-info.yaml schema (versioned), TASK_STATE/DONE, FACT files, timestamp/run_id format, home-folders.md, UTF-8 encoding, retention.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 3 — Message Bus Tooling & Object Model: run-agent bus CLI/REST, YAML front-matter format, message types/threading, atomic appends, streaming/polling, relationship schema, cross-scope references.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 4 — Monitoring & Control UI: React UI (TypeScript + Ring UI + JetBrains Mono) served by run-agent serve, project/task/run tree, threaded message bus view, live output streaming via SSE, task creation UI, webpack dev workflow, REST/JSON API with integration tests.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 5 — Agent Protocol & Governance: Agent behavior rules, run folder usage, delegation depth (max 16), message-bus-only comms, git safety guidance, no sandbox.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 6 — Environment & Invocation Contract: JRUN_* internal vars, prompt preamble path injection (OS-native normalization), PATH prepending, full env inheritance, SIGTERM/SIGKILL signal handling (30s grace period), error-message rules.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 7 — Agent Backend Integrations: Per-agent adapter specs (codex, claude, gemini, perplexity, xAI), CLI/REST invocation and I/O contracts, token management (@file support), env var mapping, output conventions, streaming behavior. All 4 active backends verified ready.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, subsystems]
Subsystem 8 — Frontend-Backend API Contract: REST/JSON + SSE API endpoints for monitoring UI, request/response schemas, TypeScript type generation, error handling, security (path validation), log/message streaming.

---

## Planning Process Facts

[2026-02-04 11:00:29] [tags: swarm, idea, legacy, planning]
Planning conducted in rounds (1-7+) across approximately 3 days (2026-02-01 to 2026-02-04). Round 6 consolidated all Q&A from *-QUESTIONS.md files and created schema specifications. Round 7 verified agent backend CLI flags and researched Perplexity streaming. Round 7+ simplified config schema per user feedback.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, planning]
6 agents independently reviewed all 8 subsystems by mentally modeling execution flows: 3x Claude Sonnet 4.5, 3x Gemini (6 successful runs; 3 Codex failures produced no output — connectivity issue). All 6 agents identified similar critical issues with remarkable consensus on top problems.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, planning]
Review consensus: Architecture universally praised for: excellent separation of concerns, solid auditability with versioned schemas, strong backend abstraction (CLI + REST), elegant Ralph loop restart design, path safety, streaming-first architecture, no premature optimization.

[2026-02-04 22:50:12] [tags: swarm, idea, legacy, planning]
Planning phase declared COMPLETE on 2026-02-04. All 8 subsystems fully implementation-ready. 30+ design questions resolved (100% resolution rate). ~2,500+ lines of specifications across 15 subsystem specs + 6 supporting documents.

---

## Project Naming Decision

[2026-02-05 11:44:55] [tags: swarm, idea, legacy, naming]
Naming task: The temporary project name "jonnyzzz-swarm" was too generic. Requirements: professional, memorable, descriptive of orchestration/agents/coordination, unique, short (2-3 words hyphenated), technology-neutral (no "ai-" or "llm-" prefixes).

[2026-02-05 11:44:55] [tags: swarm, idea, legacy, naming]
Naming candidates considered: orchestration, conductor, maestro, swarm, hive, nexus, relay, chorus, ensemble. "conductor-loop" was chosen, combining the conductor metaphor (orchestrating multiple agents) with loop (the Ralph restart loop at the heart of the system).

---

## Additional Ideas (Later Additions to ideas.md)

[2026-02-11 10:59:19] [tags: swarm, idea, legacy, ideas]
Idea: Allow message dependencies; introduce "issue" type for message bus. Allow issues to relate to each other. Research https://github.com/steveyegge/beads approach (message dependency chains).

[2026-02-11 10:59:19] [tags: swarm, idea, legacy, ideas]
Idea: Add support for Perplexity as a native REST-backed agent in run-agent. Support xAI. Research best coding agent for xAI.

[2026-02-11 10:59:19] [tags: swarm, idea, legacy, ideas]
Idea: When task is restarted, prepend to prompt "Continue working on the following:".

[2026-02-11 10:59:19] [tags: swarm, idea, legacy, ideas]
Idea: Need dedicated document for environment variable specifications between all tool calls.

[2026-02-11 11:02:34] [tags: swarm, idea, legacy, ideas]
Idea (2026-02-11): Make sure we are reporting progress output from each agent. Need to see the output to determine if an agent is alive or not. Review how ~/Work/mcp-steroid/test-helper does for Docker*Session classes.

[2026-02-21 10:58:16] [tags: swarm, idea, legacy, architecture]
Architecture goal (2026-02-21): Keep things as separated as possible. Allow as many run-agent processes as possible. Each process is independent and works fully with disk as main storage. The web UI process is only for monitoring; it should never be a dependency. The monitoring process is fully optional from the standpoint of the system. No dependency on the monitoring process should exist.

---

## Key Design Principles (Summary)

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, principles]
MAIN principle: Prompts are done recursively. Any run-agent agent must be able to review and decide if it just works or executes deeper. Limit of 16 agents deep.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, principles]
MAIN2 principle: Let agents split work by subsystems too. Start one agent to dig a selected project module/folder — this consolidates context and reduces waste.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, principles]
Process group management: Use process groups (PGID) for managing agent hierarchies. Send signals to process groups (negative PID), not individual PIDs. Find and terminate all child PIDs recursively. Log START/STOP/CRASH events to message bus for auditability.

[2026-02-04 17:54:24] [tags: swarm, idea, legacy, principles]
The web UI should be ready to maintain multiple backends/hosts (future goal). For MVP: localhost only, read-only interface.

---

## Validation Round 2 (codex)

[2026-02-23 19:15:06] [tags: swarm, idea, legacy, validation]
Feature Status: "Beads-inspired" message dependency model (planned 2026-02-11) was NOT implemented. Grep for "beads", "blocking", "kind" in internal/ yielded no results.

[2026-02-23 19:15:06] [tags: swarm, idea, legacy, validation]
Feature Status: Global facts storage and promotion (planned 2026-02-11) was NOT implemented. Grep for "global fact", "promote" in internal/ yielded no results.

[2026-02-23 19:15:06] [tags: swarm, idea, legacy, validation]
Feature Status: Multi-host support (planned 2026-01-29) is partially stubbed (RemoteAddr in audit logs) but no complex logic exists in codebase. Remains a future goal.

[2026-02-23 19:15:06] [tags: swarm, idea, legacy, validation]
Evolution: Tooling transitioned from shell scripts (`run-agent.sh`, `start-task.sh` in 729df20) to a unified Go binary (`run-agent` in 2d156d41). The `run-agent` binary successfully merged task, job, bus, and serve commands as planned.

[2026-02-23 19:15:06] [tags: swarm, idea, legacy, validation]
Evolution: Naming evolved from "jonnyzzz-swarm" (temporary) to "conductor-loop" (final choice on 2026-02-05) to reflect the "conductor" (orchestrator) and "loop" (Ralph restart loop) metaphors.
