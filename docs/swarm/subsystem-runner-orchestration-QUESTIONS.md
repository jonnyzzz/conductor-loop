# Runner & Orchestration - Questions

- Q: How should Perplexity and xAI backends be integrated (CLI wrapper vs API client) and exposed via config?
  Proposed default: Add provider types in config.hcl and implement API-based clients first; allow optional CLI wrapper later.
  A: First of all, we specify api keys in the config (or we speficy a token file). Second, we add perplexity as yet another agent type to the run-agent command. Make Perpelxity as native agent to fit, use their REST API to deal with the API. We need a dedicated design document per agent type (codex, claude code, gemini). For xAI, we need to conduct a sub-agent driven research/review to figure out what coding agent is the best to work with xAI. Next we discuss xAI here in questions.
