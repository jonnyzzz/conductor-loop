# Environment & Invocation Contract - Questions

## Open Questions

### Q1: RUNS_DIR and MESSAGE_BUS Environment Variables
**Issue**: The current contract explicitly excludes MESSAGE_BUS/TASK_MESSAGE_BUS and does not define RUNS_DIR, but older notes referenced RUNS_DIR/MESSAGE_BUS as read-only if present. The runner also does not set or enforce these variables.

**Code Evidence**:
- `internal/runner/job.go` merges env overrides without filtering.
- No references to RUNS_DIR or MESSAGE_BUS exist in the runner.

**Question**: Should run-agent keep RUNS_DIR/MESSAGE_BUS out of the contract (current spec), or start injecting them as read-only and block overrides from CLI/env maps?

**Answer**: (Pending - user)

## Resolved Questions

No resolved questions at this time.
