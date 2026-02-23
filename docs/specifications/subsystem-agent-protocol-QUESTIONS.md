# Agent Protocol & Governance - Questions

## Open

1. Should protocol docs require capturing provider session IDs for each agent call when backend supports it?
Answer: unresolved in current implementation.
Notes: no general session-id field is currently persisted in run metadata.

## Resolved

1. Who guarantees `output.md` exists?
Answer: runner guarantees it.
Evidence: `agent.CreateOutputMD(runDir, "")` is always used as fallback when run completes.

2. Is restart prefix enforcement implemented?
Answer: yes.
Evidence: `RestartPrefix = "Continue working on the following:\n\n"` is applied on Ralph-loop retries (`attempt > 0`) and in resume mode on first attempt.

3. Is delegation-depth enforcement implemented in runtime?
Answer: no.
Evidence: current runner/task execution has no enforced max-depth guard; depth remains governance-level policy.

4. Are JRUN/TASK/RUN variables real env vars or prompt-only hints?
Answer: real env vars (and also shown in prompt preamble).
Evidence: runner sets `JRUN_*`, `MESSAGE_BUS`, `TASK_FOLDER`, `RUN_FOLDER`, `RUNS_DIR`, optional `CONDUCTOR_URL` in process environment.

5. Are lifecycle bus events standardized as `RUN_START`/`RUN_STOP`/`RUN_CRASH`?
Answer: yes.
Evidence: runner posts these constants from `messagebus` during start/finish/failure paths.
