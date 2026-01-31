# Runner & Orchestration - Questions

- Q: What is the exact completion signal in TASK_STATE.md that ends the "ralph" restart loop?
  Proposed default: A line "status: done" or "done: true" at the top of TASK_STATE.md.
  A: We prompt the root agent to return "COMPLETED" message when the job is done. It is possible to restart the agent and let it move on further. Sometimes agent can complete the work (stop), but the work is not complete, to handle that we by default start the root agent again until it returns the clear message. It can be different agent each run.

- Q: How should "lucky" agent selection work (round-robin, random, weighted)?
  Proposed default: Round-robin across codex/claude/gemini per root restart.
  A: Round-robin + handling of errors and re-runs

- Q: Should run-task enforce a maximum number of restarts or time budget?
  Proposed default: Max restarts = 20, or max wall time = 8 hours.
  A: Yes, do that. Feature to control run-agent.sh behaviour too (e.g. park and say "Waiting for other agents to complete" instead of direct run)
  AA: Should be configured in the app settings under user home

- Q: Where must the "COMPLETED" signal be emitted (stdout, TASK_STATE.md, MESSAGE-BUS)?
  Proposed default: Root agent writes "COMPLETED" to TASK-MESSAGE-BUS.md and updates TASK_STATE.md.
  A: AGREED

- Q: How does run-agent.sh ensure JRUN_ID uniqueness if a run directory already exists?
  Proposed default: Generate run id with timestamp+pid and retry with random suffix on collision.
  A: run-agent and run-task sub commands manage JRUN_ID, it is never set by an agent itself. Just use how it's done today -- use date time + PID.

- Q: What happens if run-task is invoked without --project or --task?
  Proposed default: Interactive prompt to select existing project/task or create new.
  A: It fails. This feature must not be visible if started under run-task (where environment variables are set)

- Q: Should run-agent.sh enforce a per-run timeout (separate from run-task global budget)?
  Proposed default: No timeout by default; allow JRUN_AGENT_TIMEOUT to opt-in.
  A: No timeout for now. But it should track if an agent is fully idle for long time. Create an option for idle-timeout in the settings, set it to 1 minute as default.

- Q: How should "lucky" agent selection handle failures (rate limits/auth errors)?
  Proposed default: Mark failed agent as degraded for N minutes, log to message bus, and try next agent.
  A: AGREED. Also we need to detect codex and any other agent sandbox which can break it and want.

- Q: What extra run metadata should be captured for observability (duration, exit code, etc.)?
  Proposed default: Persist start/end time, duration, exit code, agent type, cwd, command line.
  A: Agree.

- Q: How should message-bus poller crashes be handled?
  Proposed default: Exponential backoff restarts with a max retry window, and alert in monitoring UI.
  A: Just crash, show the error in output, let agent handle it.

- Q: Should run-task validate TASK.md is non-empty before starting an agent?
  Proposed default: Yes, require a minimum prompt length and fail fast otherwise.
  A: yes, it must validate that. 
