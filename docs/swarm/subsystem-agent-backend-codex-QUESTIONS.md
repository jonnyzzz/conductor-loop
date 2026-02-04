# Agent Backend: Codex - Questions

- Q: Should run-agent allow overriding the Codex model and reasoning settings per task? | Proposed default: Use CLI defaults; allow optional overrides via config.hcl. | A: TBD.
- Q: Should the Codex adapter always run with `--dangerously-bypass-approvals-and-sandbox`, or allow a safer mode? | Proposed default: Keep full-access mode for now to match project constraints. | A: TBD.
