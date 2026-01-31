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
