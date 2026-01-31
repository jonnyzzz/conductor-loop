# Runner & Orchestration - Questions

- Q: What is the exact completion signal in TASK_STATE.md that ends the "ralph" restart loop?
  Proposed default: A line "status: done" or "done: true" at the top of TASK_STATE.md.

- Q: How should "lucky" agent selection work (round-robin, random, weighted)?
  Proposed default: Round-robin across codex/claude/gemini per root restart.

- Q: Should run-task enforce a maximum number of restarts or time budget?
  Proposed default: Max restarts = 20, or max wall time = 8 hours.
