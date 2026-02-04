# Environment & Invocation Contract - Questions

- Q: Confirm the exact JRUN_* variable names and full list (project/task/run/parent/backend metadata).
  Proposed default: JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID, JRUN_PARENT_ID.
  A: Correct. Plus, we update PATH to include the run-agent tool path. run-agent tool is responsible to add paths to task and run folders to the beginning of the prompt for agent.

- Q: Which injected variables are read-only vs writable for agents?
  Proposed default: JRUN_* and RUN_FOLDER are read-only; no agent-writable env vars.
  A: Agent should not use environment variables, only use prompt and CWD to work and know the paths.

- Q: Should RUN_FOLDER also be set for the root agent (pointing at task folder) or only for sub-agents?
  Proposed default: Only sub-agents receive RUN_FOLDER; root relies on CWD and task paths.
  A: No, RUN_FOLDER is only included in the paths and prompts, we do not use environment variables for agents internally.
