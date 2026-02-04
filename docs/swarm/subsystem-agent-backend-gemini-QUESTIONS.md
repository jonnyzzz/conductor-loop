# Agent Backend: Gemini - Questions

- Q: Should `--approval-mode yolo` remain the default, or should a safer approval mode be supported? 
  Proposed default: Keep yolo for now to match full-access policy. 
  A: We start all agents with full permissions and without sandbox. See ../run-agent.sh for details.

- Q: Which Gemini model should be the default for coding tasks, and should this be configurable per task? 
  Proposed default: Use CLI defaults; allow optional override in config.hcl. 
  A: See the answer for Claude.
