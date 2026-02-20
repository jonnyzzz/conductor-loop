# Environment & Invocation Contract - Questions

## Open Questions

TODO: Research about CLAUDECODE environment variable. 

TODO2: We need integration tests with all agents to log environment variables difference between the agent environment and the sub-task environment. Just create test-task.sh that captures enviroments, ask an agent (in Docker) to start it and let's see how env changes. For all supported agents.


### Q1: RUNS_DIR and MESSAGE_BUS Environment Variables
**Issue**: The current contract explicitly excludes MESSAGE_BUS/TASK_MESSAGE_BUS and does not define RUNS_DIR, but older notes referenced RUNS_DIR/MESSAGE_BUS as read-only if present. The runner also does not set or enforce these variables.

**Code Evidence**:
- `internal/runner/job.go` merges env overrides without filtering.
- No references to RUNS_DIR or MESSAGE_BUS exist in the runner.

**Question**: Should run-agent keep RUNS_DIR/MESSAGE_BUS out of the contract (current spec), or start injecting them as read-only and block overrides from CLI/env maps?

**Answer**: Inject RUNS_DIR and MESSAGE_BUS as informational env vars. Do NOT block overrides — agents may need to redirect these for sub-tasks. These are "available if you need them" additions to the contract, not enforced constraints.

**Implementation (2026-02-20)**: RUNS_DIR and MESSAGE_BUS env vars are now injected into agent subprocess environment via envOverrides in internal/runner/job.go. They are set alongside JRUN_* variables. Validated by 6 integration tests in `internal/runner/env_contract_test.go`.

## Resolved Questions

- Q1: RUNS_DIR and MESSAGE_BUS — inject as informational, don't block overrides (resolved 2026-02-20)
