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
  A: Use TASK-MESSAGE-BUS.md as the active file; when rotating, move it to TASK-MESSAGE-BUS-<N>.md and continue writing to TASK-MESSAGE-BUS.md. The largest N is the most recent archive.

- Q: The run metadata file is named "cwd" in the spec. Should this be renamed to run-info/run-meta?
  Proposed default: Rename to run-info.env (optionally keep cwd.txt for backward compatibility).
  A: TBD
