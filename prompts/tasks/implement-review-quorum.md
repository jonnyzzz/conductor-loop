# Task: Implement `run-agent review quorum`

## Context

The CLI currently has no review command group:
- `./bin/run-agent review quorum --help` returns `unknown command "review"`.
- Message bus data already contains the primitives needed for quorum checks (`type`, `run_id`, `body` in `internal/messagebus/messagebus.go`).

A lightweight quorum gate is needed so orchestrators can programmatically decide whether enough independent reviewers approved a change.

## Scope

Add a new command group:
- `run-agent review quorum`

## Requirements

1. Add CLI wiring:
- Register top-level `review` command and `quorum` subcommand in `cmd/run-agent/main.go`.
- Implement command code in `cmd/run-agent/review.go` (new file).

2. Input and filtering:
- Required flags:
  - `--project`
  - `--runs <run-id[,run-id...]>`
- Optional flags:
  - `--root` (default `./runs` or `JRUN_RUNS_DIR`)
  - `--required` (default `2`)
  - `--task` (scope to one task bus when known)
- Read relevant task/project message bus entries and consider only messages tied to provided run IDs.

3. Quorum semantics (deterministic, explicit):
- Count approvals from messages where:
  - type in `{DECISION, REVIEW}`
  - body contains case-insensitive approval tokens: `APPROVED`, `LGTM`, or `+1`
- Count rejections where body contains `REJECTED`, `BLOCKED`, or `CHANGES_REQUESTED`.
- Exit rules:
  - success (exit 0): approvals >= `--required` and rejections == 0
  - failure (exit 1): otherwise
- Print a concise summary:
  - considered runs
  - approvals, rejections, required
  - matched message IDs

4. Test coverage:
- Add `cmd/run-agent/review_quorum_test.go` for:
  - quorum met
  - quorum not met
  - rejection veto
  - mixed-case body matching
  - missing/unknown run IDs

## Acceptance Criteria

- `run-agent review quorum --help` is available.
- Command exits 0 only when quorum is reached and no rejection is present.
- Output includes machine-readable counts and matched run IDs.

## Verification

```bash
cd /Users/jonnyzzz/Work/conductor-loop
go build ./cmd/run-agent

# Unit tests
go test ./cmd/run-agent -run 'TestReviewQuorum' -count=1

# CLI surface
./bin/run-agent review quorum --help

# Example bus setup + check
run-agent bus post --project p1 --task t1 --type DECISION --body "APPROVED" --run r1
run-agent bus post --project p1 --task t1 --type REVIEW --body "LGTM" --run r2
./bin/run-agent review quorum --root /Users/jonnyzzz/run-agent --project p1 --task t1 --runs r1,r2 --required 2
```

## Key Files

- `cmd/run-agent/main.go`
- `cmd/run-agent/review.go` (new)
- `internal/messagebus/messagebus.go`
- `cmd/run-agent/bus.go`
