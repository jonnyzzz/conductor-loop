# Task: Implement run-agent iterate

## Context
The command `run-agent iterate` is documented but missing. This command should implement the iteration loop: run -> review -> fix -> repeat until success or max iterations.

## Requirements
- Register `iterate` command in CLI.
- Implement the loop logic:
  1. Execute a job.
  2. Run a review/validation step.
  3. If review fails, trigger a "fix" job (passing previous output and failure context).
  4. Repeat until review passes or `--max-iterations` reached.
- Flags: `--max-iterations` (default 3), `--project`, `--prompt`, `--review-prompt`.

## Acceptance Criteria
- `run-agent iterate` appears in `--help`.
- Successfully loops through the run/review/fix cycle.
- Terminates correctly on success or max iterations.

## Verification
```bash
# Run iteration loop
run-agent iterate --project p1 --prompt "Fix bug X" --review-prompt "Check if bug X is fixed" --max-iterations 2
```
