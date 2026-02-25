# Agent Protocol & Governance - Questions

## Open

None.

## Resolved

1. Should protocol docs require capturing provider session IDs for each agent call when backend supports it?
**Answer (2026-02-24):** No, the current implementation does not capture session IDs. The `RunInfo` schema (`internal/storage/runinfo.go`) does not include a field for it, and agent parsers (e.g. Claude) do not extract it.

2. Who guarantees `output.md` exists?
Answer: runner guarantees it.
Evidence: `agent.CreateOutputMD(runDir, "")` is always used as fallback when run completes.

3. Is restart prefix enforcement implemented?
Answer: yes.
Evidence: `RestartPrefix = "Continue working on the following:\n\n"` is applied on Ralph-loop retries (`attempt > 0`) and in resume mode on first attempt.

4. Is delegation-depth enforcement implemented in runtime?
Answer: no.
Evidence: current runner/task execution has no enforced max-depth guard; depth remains governance-level policy.

5. Are JRUN/TASK/RUN variables real env vars or prompt-only hints?
Answer: real env vars (and also shown in prompt preamble).
Evidence: runner sets `JRUN_*`, `JRUN_MESSAGE_BUS`, `JRUN_TASK_FOLDER`, `JRUN_RUN_FOLDER`, `JRUN_RUNS_DIR`, optional `JRUN_CONDUCTOR_URL` in process environment.

6. Are lifecycle bus events standardized as `RUN_START`/`RUN_STOP`/`RUN_CRASH`?
Answer: yes.
Evidence: runner posts these constants from `messagebus` during start/finish/failure paths.
