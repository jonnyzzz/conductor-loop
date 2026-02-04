# Agent Protocol & Governance - Questions

- Q: Confirm the exact RUN_FOLDER prompt preamble format (variable name and wording) that run-agent injects for sub-agents.
  A: We keep consistent set of environment variables that are set and ensuted along the call tree. All run folders are always created under the task folder. Each run usees a dedicated sub-folder under the runs' folder. Each task has it's own dedicated run folder.
