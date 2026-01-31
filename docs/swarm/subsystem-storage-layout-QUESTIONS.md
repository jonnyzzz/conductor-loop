# Storage & Data Layout - Questions

- Q: Should TASK_STATE.md include a machine-readable schema (YAML/JSON) instead of free text?
  Proposed default: Keep YAML-like key/value lines for the first block, then free-text bullets.
  A: Just text message from agent to itself. Agent should update it time after time.

- Q: Should FACT files be per-agent-run or per logical fact group?
  Proposed default: Per logical fact group (one FACT file per batch of facts).
  A: Accepted. Plus propagation to the project level. Review all facts of a task and propagate up.

- Q: Is there a maximum size for message bus files (rotation)?
  Proposed default: Rotate at 5MB and archive with suffix .old.
  A: Keep it limited, yes, yes, move to old and remove messages which are unrelevant. Better approach use -N.md ending for archives; keep TASK-MESSAGE-BUS.md as the active file.

- Q: ideas.md mentions home-folders.md for tracking source paths. Should it be part of the layout?
  Proposed default: Yes, add home-folders.md at project root.
  A: Accepted. Already included in ideas.md layout; add it explicitly to this spec.

- Q: What is the canonical naming scheme for rotated message bus files?
  Proposed default: Keep active TASK-MESSAGE-BUS.md and rotate to TASK-MESSAGE-BUS-<N>.md (N increments).
  A: Use TASK-MESSAGE-BUS-N.md as the active file; when rotating, increase N. Always write to the maximal N. 
  A++: Since we manage it with the go binary, it's much more easy to support and propose more options.
- A+++: Consider a format where each message bus message is clearly separated from the other messages. I do still prefer text output to the agent. And keep the file hunam readable.
- A++++: MESSAGE-BUS is now a folder, each message is message-<date>-<pid>.md, use --- --- with YAML as header. 

- Q: The run metadata file is named "cwd" in the spec. Should this be renamed to run-info/run-meta?
  Proposed default: Rename to run-info.env (optionally keep cwd.txt for backward compatibility).
  A: It makes sense to put many parameters to a run-info.json/yaml file. We need to keep parent/child run id, task id, project id, work-dir of agent, task folder, and so on
  A++: The layout is designed to allow post-run reviews

- Q: Should the task state file be named TASK_STATE.md (spec) or STATE.md (ideas)?
  Proposed default: Use TASK_STATE.md as canonical; omit STATE.md unless backward-compat is required.
  A: Use TASK_STATE.md.

- Q: What is the canonical task-level facts filename (TASK-FACTS-<timestamp>.md vs TASK FACT FILES-<date-time>.md)?
  Proposed default: Standardize on TASK-FACTS-<timestamp>.md.
  A: Use TASK-FACTS-<timestamp>.md.

- Q: What is the intended split between output.md and agent-stdout.txt/agent-stderr.txt?
  Proposed default: output.md is the final agent response; stdout/stderr are raw logs and optional.
  A: TBD.

- Q: Should run metadata include PROJECT_ID and TASK_ID fields (in addition to RUN_ID)?
  Proposed default: Yes, include PROJECT_ID and TASK_ID in run metadata.
  A: Yes, include project/task IDs.

- Q: What timestamp format should be used for <timestamp>/<date-time> in names?
  Proposed default: YYYYMMDD-HHMMSS (zero-padded) for lexical sort.
  A: TBD.

- Q: ideas.md mentions a start/stop log from run-agent.sh; where should that live in the layout?
  Proposed default: Global ~/run-agent/run-agent.log (optionally per-project logs).
  A: TBD.

- Q: In run-info (cwd) metadata, are PROMPT/STDOUT/STDERR fields paths or inline content?
  Proposed default: Store relative file paths to avoid duplication.
  A: TBD.

- Q: Is MESSAGE-BUS stored as a single file or a folder of per-message files?
  Proposed default: Single file for MVP; folder-based only if tooling requires it.
  A: TBD.

- Q: Rotation semantics conflict: is TASK-MESSAGE-BUS.md always active, or do we write to TASK-MESSAGE-BUS-N.md?
  Proposed default: TASK-MESSAGE-BUS.md is active; TASK-MESSAGE-BUS-<N>.md are archives.
  A: TBD.

- Q: What is the canonical format for run metadata (run-info): .env, JSON, or YAML?
  Proposed default: .env-style key/value for simplicity; consider JSON for richer metadata.
  A: TBD.

- Q: If MESSAGE-BUS format changes (single file vs folder), what is the migration path for existing runs?
  Proposed default: Keep backward-compatible readers and migrate on demand.
  A: TBD.
