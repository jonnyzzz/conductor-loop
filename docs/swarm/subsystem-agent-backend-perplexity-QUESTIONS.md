# Agent Backend: Perplexity - Questions

- Q: Which Perplexity model should be the default for coding tasks (e.g., sonar-pro vs sonar-reasoning), and should this be configurable per task? 
  Proposed default: Choose a reasoning-capable model by default; allow override in config.hcl. 
  A: Yes, introduce perplexity section in the config, use the most smart model by default.
 
- Q: What timeout and retry policy should the Perplexity adapter use for long prompts? 
  Proposed default: Align with runner transient backoff (1s/2s/4s) and a 60s request timeout.
  A: We tend to run the tool with progress updates, we measure if the tool is not quiet for some time, the same as for other agent tools
