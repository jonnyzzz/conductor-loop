# Storage & Data Layout - Questions

- Q: Should TASK_STATE.md include a machine-readable schema (YAML/JSON) instead of free text?
  Proposed default: Keep YAML-like key/value lines for the first block, then free-text bullets.

- Q: Should FACT files be per-agent-run or per logical fact group?
  Proposed default: Per logical fact group (one FACT file per batch of facts).

- Q: Is there a maximum size for message bus files (rotation)?
  Proposed default: Rotate at 5MB and archive with suffix .old.
