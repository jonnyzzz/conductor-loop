# Agent Backend: Codex - Questions

- Q: Should run-agent allow overriding the Codex model and reasoning settings per task?
  Proposed default: Use CLI defaults; allow optional overrides via config.hcl. 
  A: See Claude answers

- Q: Should the Codex adapter always run with `--dangerously-bypass-approvals-and-sandbox`, or allow a safer mode? 
  Proposed default: Keep full-access mode for now to match project constraints. A: TBD.
  A: We start all agents with `--dangerously-bypass-approvals-and-sandbox` for now, make sure we pass agent-specific flags for all agents. See ../run-agent.sh as the example
