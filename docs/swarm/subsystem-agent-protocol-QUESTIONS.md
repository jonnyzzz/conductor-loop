# Agent Protocol & Governance - Questions

- Q: How strict should agent exit timing be (max runtime)?
  Proposed default: Soft limit 15 minutes per run; require delegation after that.
==  A: No time limit right now, nice feature for UI to restart an agent, since we persist promt and commandline and cwd. And it must be logged to the message bus. Plus each agent should be instructed to re read message bus as often as possible (monior only new content) 

- Q: Should the protocol enforce a standard format for FACT files (YAML/Markdown template)?
  Proposed default: Markdown with a short header: date, author, scope.
=== A: Markdown should be used

- Q: Do we need an explicit "git allowlist" per task?
  Proposed default: Optional file TASK_GIT_ALLOWLIST.md listing writable paths.
=== A: Need more information to cover that question. Normally the project folder != the root agent run folder. The root agent must start sub agent processes in the target project folder. This information is perisited inthe project directory for project. See home-folders.md.