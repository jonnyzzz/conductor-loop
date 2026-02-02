# Storage & Data Layout - Questions

- Q: What is the canonical run metadata file name and format (run-info.env vs run-info.yaml/json), and what fields must it include?
  Proposed default: run-info.yaml with RUN_ID, PROJECT_ID, TASK_ID, PARENT_RUN_ID, AGENT, CWD, START_TIME, END_TIME, EXIT_CODE, PID, PROMPT_PATH, OUTPUT_PATH, STDOUT_PATH, STDERR_PATH.
  A: as proposed, but all keys lowercase. It's yaml format, and we can include more info later.

- Q: Is MESSAGE-BUS stored as a single append-only file or as a folder of per-message files?
  Proposed default: Keep a single logical bus; implementation may be single-file or folder-based, owned by the Go tooling.
  A: Single append only file. With careful and atomic writes. One file for each task yet another one for project.

- Q: If MESSAGE-BUS is folder-based, what is the per-message file naming and directory layout?
  Proposed default: message-<YYYYMMDD-HHMMSS>-<pid>.md with YAML header + body; grouped under TASK-MESSAGE-BUS/ or PROJECT-MESSAGE-BUS/ folders.
  A: Not folder based.

- Q: How should home-folders.md be structured (format and required fields)?
  Proposed default: YAML with project_root, source_folders[], additional_folders[] (one per line, absolute paths).
  A: Correct, use as proposed. Include text explanation for each folder.

- Q: What triggers promotion of task facts to project-level FACT files, and how is scope indicated?
  Proposed default: Promotion runs on task completion; task fact files may include a short header with scope=project.
  A: A message in message bus, agent should know if can do that. Otherwise, nothing. All agents has read only access to the project-level message bus and all sub tasks. That means project level message bus announces new tasks are started with information to read about the task details and how.

- Q: Where should poller offsets/cursors be persisted on disk for project/task message buses?
  Proposed default: .offsets/<consumer-id>.txt under the bus directory storing last seen msg_id.
  A: Offests are not needed, it's up to an AI Agent to deal with offestes. The run-agent CLI should be designed to support that and AI Agent should easily fetch necessary pages one by one. Promote to start sub-agent to read all messages.

- Q: What cleanup/archival policy applies to old run directories?
  Proposed default: Keep all runs by default; optional archive runs older than N days to runs-archive/ (configurable).
  A: No cleanup is needed so far.

- Q: Should the Go binary maintain an index file for fast UI lookup of projects/tasks/runs?
  Proposed default: Optional index.yaml under ~/run-agent/.index.yaml maintained by the runner.
  A: No, let's avoid indexes and caches. We can introduce them later if needed.

- Q: Should run metadata capture backend details (provider, model, endpoint, credential alias) for audits?
  Proposed default: Add backend fields to run-info.yaml; never store secrets.
  A: Yes, only if that is easy to capture. We can show some of that info in UI.
