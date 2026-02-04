# Agent Backend: Codex - Questions

- Q: What environment variable name does the Codex CLI expect for API credentials (e.g., OPENAI_API_KEY), and how should run-agent map config keys to it? 
  Proposed default: Map OPENAI_API_KEY from config.hcl. 
  A: Map OPENAI_API_KEY from config.hcl. All keys should allow passing the key file instead of the actual value. look ../run-agent.sh for details and conduct necessary experiments.

