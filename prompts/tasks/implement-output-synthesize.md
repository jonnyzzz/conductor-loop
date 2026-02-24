# Task: Implement `run-agent output synthesize`

## Context

`run-agent output` exists today (`cmd/run-agent/output.go`) but only prints a single run output file. The command surface has no synthesis mode:
- `./bin/run-agent output --help` lists only file/tail/follow flags.
- There is no `output synthesize` subcommand in `cmd/run-agent/main.go` or `cmd/run-agent/output.go`.

This leaves a gap for multi-agent workflows where several run outputs must be aggregated into one artifact for review/handoff.

## Scope

Implement `run-agent output synthesize` as a subcommand under `output`.

## Requirements

1. Add command wiring:
- Register `synthesize` under `newOutputCmd()` in `cmd/run-agent/output.go`.
- Keep existing `run-agent output` behavior unchanged.

2. Implement synthesis inputs:
- Accept run selectors via `--runs` (comma-separated run IDs and/or absolute run paths).
- Support both:
  - explicit `--run-dir` style absolute paths
  - run IDs resolved from `--root`, `--project`, `--task`
- Reject empty input with a clear validation error.

3. Implement synthesis output behavior:
- Default mode: concatenate each run's output with deterministic section headers:
  - `## run: <run-id>`
  - source path
  - content block
- For each run, prefer `output.md`; fallback to `agent-stdout.txt` if `output.md` missing (same fallback semantics as existing `output` command).
- If a run has no output artifact:
  - default: emit a warning and continue
  - `--strict`: fail the command

4. Output destination:
- Default write to stdout.
- Optional `--out <path>` writes synthesized markdown to file.

5. Tests:
- Add unit tests in `cmd/run-agent/output_synthesize_test.go` for:
  - successful concat of multiple runs
  - mixed run IDs + absolute run paths
  - fallback to `agent-stdout.txt`
  - missing output behavior (`default` vs `--strict`)
  - deterministic ordering of synthesized sections
- Keep existing `cmd/run-agent/output_test.go` and `output_follow_test.go` passing.

## Acceptance Criteria

- `run-agent output synthesize --help` is available and documented.
- Aggregation works for at least 2 runs in one invocation.
- Missing artifacts are non-fatal by default and fatal with `--strict`.
- Existing `run-agent output` behavior is unchanged.

## Verification

```bash
# Build
cd /Users/jonnyzzz/Work/conductor-loop
go build ./cmd/run-agent

# Tests for output command surface
go test ./cmd/run-agent -run 'Test(Output|RunOutput|OutputSynthesize)' -count=1

# Help surface
./bin/run-agent output synthesize --help

# Manual smoke (example)
./bin/run-agent output synthesize \
  --root /Users/jonnyzzz/run-agent \
  --project conductor-loop \
  --task task-20260224-093009-evo-r2-nexttasks \
  --runs run-a,run-b
```

## Key Files

- `cmd/run-agent/output.go`
- `cmd/run-agent/main.go`
- `cmd/run-agent/output_test.go`
- `cmd/run-agent/output_follow_test.go`
