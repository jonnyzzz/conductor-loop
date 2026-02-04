# Agent Backend: Claude Code - Questions

- Q: What is the correct CLI flag to enable all tools (e.g., `--tools default` vs `--allowedTools`), and should run-agent mirror `claude --help` output exactly? 
  Proposed default: Use the flag that enables all tools on the installed CLI (verify with `claude --help` during implementation). 
  A: Run experiments to figure out the flags. Look at ../run-agent.sh for details. 
