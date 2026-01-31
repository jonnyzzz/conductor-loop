# Storage & Data Layout - Questions

- Q: Should TASK_STATE.md include a machine-readable schema (YAML/JSON) instead of free text?
  Proposed default: Keep YAML-like key/value lines for the first block, then free-text bullets.
AAA: Just text message from agent to itself. Agent should update it time after time.

- Q: Should FACT files be per-agent-run or per logical fact group?
  Proposed default: Per logical fact group (one FACT file per batch of facts).
AAA: Accepted. Plus propagation to the project level. Review all facts of a task and propagate up.

- Q: Is there a maximum size for message bus files (rotation)?
  Proposed default: Rotate at 5MB and archive with suffix .old.
AAA: Keep it limited, yes, yes, move to old and remove messages which are unrelevant. Better approach use -N.md ending, so agent must always use the maximal N file

