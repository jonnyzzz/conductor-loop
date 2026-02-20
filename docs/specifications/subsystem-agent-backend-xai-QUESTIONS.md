# Agent Backend: xAI - Questions

- Q: Which xAI model(s) should be the default for coding tasks? Current implementation defaults to `grok-4`, but the idea backlog mentions researching the best coding agent.
  Answer: The latest and the most powerful.

- Q: Config format/token syntax mismatch: specs reference config.hcl with inline or `@file` token values, but code currently loads YAML with `token`/`token_file` fields and no `@file` shorthand. Which format is authoritative, and should `@file` be supported by the runner?
  Answer: Same as for other agents -- token/token_file. 


TODO: Consider the coding agent, which we are going to use with xAI.


