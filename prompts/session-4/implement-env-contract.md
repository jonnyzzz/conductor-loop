# Implementation Task: Environment Contract & Token Passthrough

## Context
You are an implementation agent working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

## Required Reading
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-env-contract-QUESTIONS.md
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go — token injection logic
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go — tokenEnvVar, mergeEnv

## Task
Improve the environment contract for agent processes:

1. **Token file loading**: The config supports `token_file` but the job.go code only uses `selection.Config.Token`. If `token_file` is specified, load the token from that file and inject it. Check if this is already handled in config loading (internal/config/) or if it needs to be added to the job runner.

2. **Environment variable passthrough**: Verify that when `GEMINI_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` etc. are set in the user's environment, they are correctly passed through to agent subprocesses (the `mergeEnv` function should handle this).

3. **Add a test**: Add a test that verifies:
   - When config has a token, it's injected as the correct env var
   - When config has no token but the env var is already set, it's passed through
   - When config has token_file, the file is read and injected

4. **RUNS_DIR and MESSAGE_BUS env vars**: Per the env contract spec, these should be set for agent subprocesses. Check if they're being set in job.go's envOverrides. If not, add them:
   - `RUNS_DIR` → the runs directory path
   - `MESSAGE_BUS` → the message bus file path

## Constraints
- Follow code conventions in AGENTS.md
- Must pass: `go build ./...` and `go test ./...`
- Do NOT add new dependencies
- Minimal changes — only add what's missing

## Output
When complete:
1. Verify `go build ./...` passes
2. Verify `go test ./internal/runner/...` passes
3. Write a summary to agent-stdout.txt
