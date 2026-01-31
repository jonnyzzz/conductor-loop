# Runner & Orchestration - Questions

- Q: What is the exact completion signal in TASK_STATE.md that ends the "ralph" restart loop?
  Proposed default: A line "status: done" or "done: true" at the top of TASK_STATE.md.
AAA: We prompt the root agent to return "COMPLETED" message when the job is done. It is possible to restart the agent and let it move on further. Soemtimes agent can complete the work (stop), but the work is not complete, to handle that we by default start the root agent again untill it returns the clear message. It can be different agent each run.

- Q: How should "lucky" agent selection work (round-robin, random, weighted)?
  Proposed default: Round-robin across codex/claude/gemini per root restart.
AAA: Round-robin + handling of errors and re-runs

- Q: Should run-task enforce a maximum number of restarts or time budget?
  Proposed default: Max restarts = 20, or max wall time = 8 hours.
AAA: yes, do that. Feature to control run-agent.sh behaviour too (e.g. park and say "Waiting for other agents to complete" instead of direct run)
