# Agent Backend: xAI - Questions

- Q: Which xAI model/agent should be targeted first for coding tasks, and what is the invocation method (API vs CLI)? 
  Proposed default: Research and select a coding-focused model; use REST API if available. 
  A: We use OpenCode agent, which needs to be configured to use xAI. Only if the token is provided (same for all agents), use the best model by default. You need to pass parameters to the opencode to make it use xai model. Park this to TODOs, we address the feature later after the MVP.

- Q: Does the xAI backend require additional tool-calling or sandbox capabilities that need to be surfaced in run-agent? 
  Proposed default: Start with plain text completion; add tool-calling support later if needed. 
  A: We run all tools without sandboxing or restrictions.
