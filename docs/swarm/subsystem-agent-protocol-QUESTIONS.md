# Agent Protocol & Governance - Questions

- Q: How strict should agent exit timing be (max runtime)?
  Proposed default: Soft limit 15 minutes per run; require delegation after that.
  A: No time limit right now, nice feature for UI to restart an agent, since we persist prompt and commandline and cwd. And it must be logged to the message bus. Plus each agent should be instructed to re-read message bus as often as possible (monitor only new content).

- Q: Should the protocol enforce a standard format for FACT files (YAML/Markdown template)?
  Proposed default: Markdown with a short header: date, author, scope.
  A: Markdown should be used.

- Q: Do we need an explicit "git allowlist" per task?
  Proposed default: Optional file TASK_GIT_ALLOWLIST.md listing writable paths.
  A: Need more information to cover that question. Normally the project folder != the root agent run folder. The root agent must start sub agent processes in the target project folder. This information is persisted in the project directory for project. See home-folders.md.

- Q: Should we strictly separate PROJECT-MESSAGE-BUS.md and TASK-MESSAGE-BUS.md instead of a single MESSAGE-BUS.md?
  Proposed default: Yes, keep distinct project vs task scopes.
  A: Yes, keep distinct project vs task scopes (already in ideas.md layout).

- Q: What standard JRUN_* environment variables must be injected into every agent run?
  Proposed default: JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID, JRUN_PARENT_ID, JRUN_ROOT.
  A: In this document we name it run_id, let's use RUN_ID not JOB. JRUN_ROOT is unknown, the CWD is just a parameter. Tasks can have various CWDs, it's advised and necessary

- Q: Who is responsible for restarting the root agent when it exits?
  Proposed default: An external supervisor (run-task) handles restarts; agents do not loop.
  A: run-task aka supervisor should do that. It follows the Ralph Wignim approach (start agent to research details and get summary)

- Q: What is the naming convention for FACT files?
  Proposed default: FACT-<YYYYMMDD-HHMMSS>-<topic>.md.
  A: Agreed

- Q: What is the maximum delegation depth to prevent infinite recursion?
  Proposed default: 16 levels (from ideas.md).
  A: 16 levels.

- Q: Should agents prefer message-bus tooling (post-message.sh/poll-message.sh) or raw file appends?
  Proposed default: Use message-bus tooling when available; fallback to file appends.
  A: TBD.

- Q: How should agents choose between TASK-MESSAGE-BUS.md and PROJECT-MESSAGE-BUS.md?
  Proposed default: Task-scoped updates/questions go to TASK-MESSAGE-BUS; cross-task facts go to PROJECT-MESSAGE-BUS.
  A: Use TASK-MESSAGE-BUS for task updates; PROJECT-MESSAGE-BUS for cross-task facts.

- Q: What is the mechanism for rotating agent types across restarts?
  Proposed default: Supervisor picks next agent from config.json list (round-robin or random).
  A: TBD.

- Q: When should an agent promote a FACT file from task level to project level?
  Proposed default: When a fact is reusable across multiple tasks (architecture, module structure).
  A: TBD.

- Q: Should TASK_STATE.md follow a standard schema (status/next_steps/blockers)?
  Proposed default: Minimal schema with Status/Next Steps/Blockers sections.
  A: TBD.

- Q: What happens if MESSAGE-BUS is missing or corrupted during a run?
  Proposed default: Recreate file, log warning to ISSUES.md, post recovery notice.
  A: Create it and retry; corruption handling TBD.

- Q: Should agents invoke run-agent.sh directly to spawn sub-agents, or delegate via MESSAGE-BUS?
  Proposed default: Agents invoke run-agent.sh directly for sub-tasks; MESSAGE-BUS for coordination.
  A: Agents invoke run-agent.sh directly.

- Q: How should agents handle CWD (task folder vs project folder)?
  Proposed default: Root agent runs in task folder; code-change sub-agents run in project folder.
  A: Root agent runs in task folder; code-change sub-agents run in project folder.

- Q: What is the canonical message-bus entry format?
  Proposed default: Markdown with timestamp, run_id, type, and message body (per bus tooling spec).
  A: TBD.

- Q: How often should agents poll for new MESSAGE-BUS entries?
  Proposed default: Every 30-60s during long operations (or as often as possible).
  A: Poll as often as possible; monitor only new content.

- Q: Is there a standard exit code convention for agent runs?
  Proposed default: 0=done, 1=blocked, 2=error, 3=delegated.
  A: TBD.

- Q: Are agents allowed to modify files outside the project folder?
  Proposed default: Only task artifacts (TASK_STATE, FACT, MESSAGE-BUS, ISSUES) within task folder.
  A: Yes, only task artifacts within their task folder.

- Q: How should agents handle concurrent MESSAGE-BUS appends?
  Proposed default: Use file locking or message-bus tooling with retry.
  A: TBD.

- Q: Should agents log tool invocations to a TRACE.md file?
  Proposed default: Optional, enabled via config flag.
  A: TBD.

- Q: What should agents do if TASK_STATE.md is stale (no updates for 24h)?
  Proposed default: Post warning to MESSAGE-BUS and ask supervisor whether to restart.
  A: TBD.

- Q: Should agents validate JRUN_* environment variables on startup, and what happens if validation fails?
  Proposed default: Validate and exit with a clear error; post to MESSAGE-BUS if possible.
  A: TBD.

- Q: JRUN_* vs RUN_* naming is inconsistent across docs. Which prefix is canonical?
  Proposed default: Standardize on JRUN_* for all agent runs.
  A: TBD.
