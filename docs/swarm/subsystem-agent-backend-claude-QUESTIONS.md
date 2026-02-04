# Agent Backend: Claude Code - Questions

- Q: Which Claude model should be the default for coding tasks (e.g., Sonnet vs Opus), and should this be configurable per task? 
  Proposed default: Use CLI defaults; allow optional override in config.hcl. 
  A: Right now we do not specify a model to Claude Code at all. Let's assume it's configured on the host system the correct way.

- Q: Should tool access be restricted beyond `--tools default` for certain tasks? 
  Proposed default: Keep `--tools default` for now; revisit if security constraints are added. 
  A: All tools should be accessible and allowed. Review how ../run-agent.sh is done, it has to be the same for all tasks.
