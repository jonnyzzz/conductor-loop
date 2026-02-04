# Agent Backend: Claude Code - Questions

- Q: Which Claude model should be the default for coding tasks (e.g., Sonnet vs Opus), and should this be configurable per task? | Proposed default: Use CLI defaults; allow optional override in config.hcl. | A: TBD.
- Q: Should tool access be restricted beyond `--tools default` for certain tasks? | Proposed default: Keep `--tools default` for now; revisit if security constraints are added. | A: TBD.
