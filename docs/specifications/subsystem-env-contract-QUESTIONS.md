# Environment & Invocation Contract - Questions

## Open Questions

None. All previously open questions have been resolved.

---

## Resolved Questions

### CLAUDECODE Environment Variable
**Issue**: Research needed on whether the CLAUDECODE environment variable needed special handling.

**Resolution (2026-02-20)**: CLAUDECODE is set by the Claude CLI when it runs as an agent. Sub-agents launched via `run-agent job` inherit all parent environment variables including CLAUDECODE. No special handling is needed in run-agent — the variable passes through automatically via the inherited environment.

---

### Cross-Agent Env Comparison Tests
**Issue**: Integration tests needed to compare environment variables between agent and sub-task environments for all supported agents.

**Resolution**: Docker-based cross-agent env comparison tests are deferred. The JRUN_* and JRUN_RUNS_DIR/JRUN_MESSAGE_BUS injection is already validated by integration tests in `internal/runner/env_contract_test.go`. Full Docker multi-agent env diff testing can be added as a separate issue if needed.

---

### Q1: JRUN_RUNS_DIR, JRUN_MESSAGE_BUS, JRUN_TASK_FOLDER, JRUN_RUN_FOLDER, JRUN_CONDUCTOR_URL Environment Variables
**Issue**: The original contract explicitly excluded JRUN_MESSAGE_BUS/TASK_MESSAGE_BUS and did not define JRUN_RUNS_DIR, but older notes referenced these as read-only if present. The runner did not set or enforce these variables. The spec claimed that JRUN_TASK_FOLDER and JRUN_RUN_FOLDER were provided only via the prompt preamble, not as env vars.

**Answer (2026-02-20, confirmed 2026-02-23)**: Inject JRUN_RUNS_DIR, JRUN_MESSAGE_BUS, JRUN_TASK_FOLDER, JRUN_RUN_FOLDER, and JRUN_CONDUCTOR_URL as informational env vars. Do NOT block overrides — agents may need to redirect these for sub-tasks. These are "available if you need them" additions to the contract, not enforced constraints.

**Implementation**: All five variables are injected into agent subprocess environment via `envOverrides` in `internal/runner/job.go`:
- `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID` — runner-owned, always overwritten
- `JRUN_RUNS_DIR`, `JRUN_MESSAGE_BUS`, `JRUN_TASK_FOLDER`, `JRUN_RUN_FOLDER` — informational, always injected
- `JRUN_CONDUCTOR_URL` — informational, injected when configured

Validated by integration tests in `internal/runner/env_contract_test.go`.
