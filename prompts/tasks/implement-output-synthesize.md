# Task: Implement run-agent output synthesize

## Context
The command `run-agent output synthesize` is documented as a way to aggregate outputs from multiple sub-agent runs (e.g., concatenate or summarize for synthesis), but it is currently missing from `cmd/run-agent/main.go` and `cmd/run-agent/output.go`.

## Requirements
- Register `synthesize` as a subcommand of `output` in `cmd/run-agent/output.go`.
- Implement logic to aggregate `output.md` files from a set of specified run directories.
- Support a `--runs` flag to accept a list of run IDs or paths.
- Default behavior: concatenate outputs with clear separators.
- Logic should reside in `internal/runner/synthesize.go` (new file) or `internal/cmd/synthesize.go`.

## Acceptance Criteria
- `run-agent output synthesize --help` shows usage and flags.
- Command successfully aggregates content from multiple runs.
- Gracefully handles missing `output.md` files in specified runs.

## Verification
```bash
# Run multiple sub-jobs
run-agent job --project p1 --task t1 --prompt "output 1"
run-agent job --project p1 --task t2 --prompt "output 2"

# Synthesize results
run-agent output synthesize --project p1 --runs run-id-1,run-id-2
```
