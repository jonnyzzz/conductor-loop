# Agent Protocol & Governance Subsystem

## Overview
This document describes the current agent execution protocol as implemented by the runner.
It separates:
- Runner-enforced behavior.
- Prompt-level guidance (soft governance).

Primary implementation sources:
- `internal/runner/orchestrator.go`
- `internal/runner/task.go`
- `internal/runner/job.go`
- `internal/runner/wrap.go`
- `internal/agent/executor.go`

## Runner-Enforced Contract

### Run Layout and Core Files
For each run, the runner allocates a run directory and manages:
- `prompt.md`
- `run-info.yaml`
- `agent-stdout.txt`
- `agent-stderr.txt`
- `output.md` (guaranteed; see Output section)

### Prompt Preamble Injection
Before agent execution, the runner prepends context, including:
- `TASK_FOLDER=<abs path>`
- `RUN_FOLDER=<abs path>`
- `JRUN_PROJECT_ID=<project>`
- `JRUN_TASK_ID=<task>`
- `JRUN_ID=<run>`
- `JRUN_PARENT_ID=<parent run>` when present
- `MESSAGE_BUS=<abs task bus path>`
- `CONDUCTOR_URL=<url>` when available
- `Write output.md to <run dir>/output.md`

It also injects message-bus usage instructions and DONE-file completion instructions.

### Environment Variables Injected into Agent Process
The runner sets process env vars (not prompt-only):
- `JRUN_PROJECT_ID`
- `JRUN_TASK_ID`
- `JRUN_ID`
- `JRUN_PARENT_ID` (when set)
- `RUNS_DIR`
- `MESSAGE_BUS`
- `TASK_FOLDER`
- `RUN_FOLDER`
- `CONDUCTOR_URL` (when known)

Agent-token mappings:
- `codex` -> `OPENAI_API_KEY`
- `claude` -> `ANTHROPIC_API_KEY`
- `gemini` -> `GEMINI_API_KEY`
- `perplexity` -> `PERPLEXITY_API_KEY`
- `xai` -> `XAI_API_KEY`

### Lifecycle Event Emission
Runner posts lifecycle events to task bus:
- `RUN_START` when run starts.
- `RUN_STOP` on successful completion.
- `RUN_CRASH` on failure/non-zero execution.

These are emitted by both normal job execution and wrapped command execution.

### Restart Prefix Behavior
Task restarts prepend:
`Continue working on the following:\n\n`

Behavior:
- Applied on attempts after the first Ralph-loop attempt.
- Also applied on first attempt when task is started in resume mode.

### Output Guarantees
`output.md` is guaranteed after run completion:
1. Agent may create it directly (best effort).
2. CLI stream-json backends may synthesize parsed output.
3. Fallback always runs: `agent.CreateOutputMD(runDir, "")` copies from `agent-stdout.txt` if `output.md` is missing.

## Execution Model by Backend Type

### CLI Backends
Handled by `executeCLI` with process capture to stdout/stderr files.
Current command patterns:
- Codex: `codex exec --dangerously-bypass-approvals-and-sandbox --json -`
- Claude: `claude -p --input-format text --output-format stream-json --verbose --tools default --permission-mode bypassPermissions`
- Gemini: `gemini --screen-reader true --approval-mode yolo --output-format stream-json`

### REST Backends
Handled by `executeREST`:
- Perplexity
- xAI

They implement `agent.Agent` and write outputs via run context paths.

## Message Bus Participation
- Runner emits lifecycle events.
- Prompt instructs agents to post progress/facts/errors/decisions via `run-agent bus post`.
- Task dependency waiting also emits bus updates (`PROGRESS`, `FACT`, `ERROR`).

## Prompt-Level Governance (Soft)
The following are guidance-level, not hard runtime barriers:
- Delegation quality and scope discipline.
- Using message bus for coordination.
- Writing concise task state summaries.
- Avoiding unrelated file changes.

## Not Enforced in Current Runtime
- Hard delegation-depth limit enforcement in runner.
- Hard file ownership boundaries between sibling runs.
- Sandboxed filesystem boundaries per agent role.

## Related Specs
- `docs/specifications/subsystem-message-bus-object-model.md`
- `docs/specifications/subsystem-message-bus-tools.md`
- `docs/specifications/subsystem-frontend-backend-api.md`
