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
