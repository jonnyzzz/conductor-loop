# Task: Add JRUN_* Variables to Prompt Preamble

## Required Reading (absolute paths)
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md
- /Users/jonnyzzz/Work/conductor-loop/QUESTIONS.md (see Q6)
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-QUESTIONS.md

## Context
Per human answer (Q6/QUESTIONS.md): "the runner should set the JRUN_* variables correctly to the started agent process, agent process will start run-agent binary again for sub-agents, that is why the variables should be maintained carefully. Make sure to assert and validate consistency."

The DECISION says: Add JRUN_* values to the prompt preamble for visibility. Document them in the agent protocol spec.

Currently `job.go` sets `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID` as environment variables. The prompt preamble (built in `buildPrompt()`) only includes `TASK_FOLDER` and `RUN_FOLDER`.

## Task
1. Read the prompt building code in `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` (especially `buildPrompt()`)
2. Add the following to the prompt preamble:
   - `JRUN_PROJECT_ID=<value>`
   - `JRUN_TASK_ID=<value>`
   - `JRUN_ID=<value>`
   - `JRUN_PARENT_ID=<value>` (when non-empty)
3. Add validation that JRUN_* env vars are consistent with the values passed to the job:
   - Assert JRUN_PROJECT_ID matches --project flag
   - Assert JRUN_TASK_ID matches --task flag
   - Log a warning (not error) if there's a mismatch for now
4. Add tests for the updated prompt preamble
5. Verify: `go build ./...` passes
6. Verify: `go test ./...` passes

## Constraints
- Follow existing code patterns from AGENTS.md
- Keep changes minimal and focused
- Do NOT modify MESSAGE-BUS.md or ISSUES.md
