# Task: Implement `run-agent iterate`

## Context

`run-agent iterate` is referenced in workflow docs/backlog but not available in the binary:
- `./bin/run-agent iterate --help` returns `unknown command "iterate"`.
- Existing building blocks already exist:
  - `run-agent job` execution path (`cmd/run-agent/main.go` / runner)
  - message bus posting and status tracking
  - `workflow run` command for staged orchestration (`cmd/run-agent/workflow.go`)

The missing piece is a direct iterative loop command for run-review-fix cycles.

## Scope

Implement a first usable `run-agent iterate` command for autonomous iteration loops.

## Requirements

1. Add CLI command:
- Register `iterate` in `cmd/run-agent/main.go`.
- Implement in `cmd/run-agent/iterate.go` (new file).

2. Core loop behavior:
- Inputs:
  - `--project` (required)
  - `--task` (required)
  - `--prompt` or `--prompt-file` for initial implementation run
  - `--review-prompt` or `--review-prompt-file` for review pass
  - `--max-iterations` (default 3)
- Per iteration:
  1. Execute implementation run.
  2. Execute review run against latest implementation output/context.
  3. Evaluate review result using explicit tokens:
     - pass tokens: `APPROVED`, `PASS`
     - fail tokens: `REJECTED`, `FAIL`, `CHANGES_REQUESTED`
  4. If failed and iteration budget remains, construct next prompt by appending review feedback and retry.

3. Output and state:
- Print per-iteration summary (impl run ID, review run ID, result).
- Exit codes:
  - `0`: review passed within iteration budget
  - `1`: exhausted iterations without pass
  - `2`: execution/configuration error
- Persist iteration summary file under task folder (for example `ITERATE-SUMMARY.md` or JSON companion).

4. Bus integration:
- Post PROGRESS/FACT updates for each iteration boundary.
- Include iteration number and run IDs in body text.

5. Tests:
- Add `cmd/run-agent/iterate_test.go` covering:
  - pass on first iteration
  - fail then pass on subsequent iteration
  - max-iterations exhausted
  - invalid flag combinations

## Acceptance Criteria

- `run-agent iterate --help` is available.
- Command performs actual multi-step loop with deterministic termination.
- Iteration summary and exit code accurately reflect final result.

## Verification

```bash
cd /Users/jonnyzzz/Work/conductor-loop
go build ./cmd/run-agent

# Unit tests
go test ./cmd/run-agent -run 'TestIterate' -count=1

# CLI surface
./bin/run-agent iterate --help

# Smoke invocation
./bin/run-agent iterate \
  --root /Users/jonnyzzz/run-agent \
  --project conductor-loop \
  --task task-20260224-093009-evo-r2-nexttasks \
  --prompt "Implement X" \
  --review-prompt "Return APPROVED or REJECTED with reasons" \
  --max-iterations 2
```

## Key Files

- `cmd/run-agent/main.go`
- `cmd/run-agent/iterate.go` (new)
- `cmd/run-agent/workflow.go`
- `internal/runner/` (job execution helpers)
