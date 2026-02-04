# Environment & Invocation Contract - Questions

- Q: Confirm the exact JRUN_* variable names and full list (project/task/run/parent/backend metadata).
  Proposed default: JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID, JRUN_PARENT_ID.
  A: TBD.

- Q: Which injected variables are read-only vs writable for agents?
  Proposed default: JRUN_* and RUN_FOLDER are read-only; no agent-writable env vars.
  A: TBD.

- Q: Should RUN_FOLDER also be set for the root agent (pointing at task folder) or only for sub-agents?
  Proposed default: Only sub-agents receive RUN_FOLDER; root relies on CWD and task paths.
  A: TBD.
