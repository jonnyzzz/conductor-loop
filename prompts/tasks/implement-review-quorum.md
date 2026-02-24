# Task: Implement run-agent review quorum

## Context
The command `run-agent review quorum` is documented but missing. It is intended to check if N reviewers approved a change by analyzing the outputs or status of specific runs.

## Requirements
- Register `review` command with `quorum` subcommand in CLI.
- Implement logic to check for a "quorum" (e.g., at least N approvals).
- Flags: `--project`, `--runs` (list of runs), `--required` (N, default 1).
- Should look for a "FACT" or "DECISION" in the message bus or `output.md` indicating approval.

## Acceptance Criteria
- `run-agent review quorum --help` appears in help.
- Returns success if quorum is met, error/failure if not.
- Functional implementation in `internal/runner` or `internal/cmd`.

## Verification
```bash
# Mock some review outputs
run-agent bus post --type DECISION --body "APPROVED" --project p1 --run r1
run-agent bus post --type DECISION --body "APPROVED" --project p1 --run r2

# Check quorum
run-agent review quorum --project p1 --runs r1,r2 --required 2
```
