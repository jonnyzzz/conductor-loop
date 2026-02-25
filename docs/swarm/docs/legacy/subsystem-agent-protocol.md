# Agent Protocol & Governance Subsystem

## Overview
Defines behavioral rules for all agents in the swarm, including delegation, communication, state/fact updates, and git safety. Enforcement is primarily via prompts and runner tooling, not sandboxing.

## Goals
- Ensure small, focused agent runs that exit on completion.
- Force all communication through message bus tooling.
- Maintain a reliable state/facts trail for handoff.
- Reduce cross-agent conflicts via delegation.

## Non-Goals
- Hard sandbox enforcement or resource limits.
- Strict path boundaries across projects.

## Behavioral Rules
- Agents MUST work on a scoped task and exit when done.
- Agents MUST delegate if a task is too large or outside their folder context.
- Agents SHOULD scope work to a single module/folder and delegate other folders to sub-agents.
- Agents MUST read TASK_STATE.md and message bus on start.
- Root agents SHOULD read PROJECT-MESSAGE-BUS.md and project FACT files on start.
- Agents MUST write updates to TASK_STATE.md each cycle (root only).
- Agents SHOULD write final results to output.md in their run folder (best-effort; runner creates output.md from stdout if missing).
- Agents MUST use run-agent bus tooling; direct file appends are disallowed.
- Agents SHOULD log progress to stderr during long operations.

## Run Folder Ownership
- No OWNERSHIP.md file.
- run-agent injects a JRUN_RUN_FOLDER path into sub-agent prompts (prompt text, not an env var).
- Prompt preamble includes full path instruction: `Write output.md to /full/path/to/<run_id>/output.md`
- This is a best-effort instruction; agents should attempt to create output.md, but if missing, the runner creates it from stdout.
- Agents write prompts/outputs/temporary files only inside JRUN_RUN_FOLDER.
- Ownership is conceptual (prompt-guided), not enforced by files.
- Parents may read child output/TASK_STATE for monitoring; policy does not restrict this.

## Output Files & I/O Capture
- **output.md**: The final result file
  - Runner prepends instruction to agent prompt: `Write output.md to /full/path/to/<run_id>/output.md`
  - Agent should attempt to write final results to this file
  - **Unified Rule**: If `output.md` does not exist after the agent exits, the runner MUST create it using the content of `agent-stdout.txt`
- **agent-stdout.txt**: Runner-captured stdout stream (always created)
  - Contains all stdout output from the agent process
  - Captured independently of agent behavior
  - Source for `output.md` creation if the agent fails to write the file
- **agent-stderr.txt**: Runner-captured stderr stream (always created)
  - Contains progress logs, debugging info, errors
  - Captured independently of agent behavior
- Parent agents can always rely on `output.md` existing (either written by agent or created by runner from stdout)

## Message Bus Protocol
- Agents do not emit START/STOP (runner-only).
- Use TASK-MESSAGE-BUS for task-scoped updates/questions.
- Use PROJECT-MESSAGE-BUS for cross-task facts (typically via root agent).
- Thread replies and corrections with parents[].
- Use absolute paths when referencing files in message bus entries or outputs.
- Root agent is responsible for polling and processing message bus updates in MVP (no dedicated poller service).
- Poll for new messages as often as possible; read only new content.

## Task State
- TASK_STATE.md is free text written by root agent.
- No strict schema; keep it short and current.

## Facts
- FACT files are Markdown with YAML front matter.
- Naming: FACT-<timestamp>-<name>.md (project), TASK-FACTS-<timestamp>.md (task).
- Promotion to project-level facts is decided by the root agent; task agents can propose via message bus.
- Root agents SHOULD promote stable task facts to project-level FACT files.

## Delegation & Depth
- Max delegation depth: 16 (configurable in global settings).
- run-agent task should fail new spawn attempts beyond the limit.

## CWD Guidance
- Root agent runs in task folder.
- Code-change sub-agents (Implementation/Test/Debug) run in project source root.
- Research/Review sub-agents default to the task folder unless the parent overrides.
- CWD is recorded in run-info.yaml for audit.

## Permissions & Safety
- No enforced read/write boundaries or sensitive-path guardrails; cross-project access is not blocked.
- Agents may execute repository scripts (no restrictions yet).
- No resource limits enforced.

## Git Safety (Guidance)
- Stage only modified files.
- Avoid destructive commands unless explicitly required.
- Provide a clear list of touched files in the final response.
- Do not touch unrelated files; commit only selected files when asked.
- Use "Git Pro" behavior: precise staging and no incidental file changes.
- Git Pro guidance is provided via THE_PROMPT_v5.md; extend that prompt with small-logical-commit instructions.

## Cancellation Protocol
- Runner sends SIGTERM to agent pgid, waits 30s, then SIGKILL.
- Agent should flush message bus + TASK_STATE on SIGTERM when possible.

## Environment Variables
- JRUN_* variables are implementation details (agents should not reference them).
- Error messages must not instruct agents to set env vars.
- JRUN_RUN_FOLDER is provided in the prompt preamble (not as an env var); treat it as read-only.

## Protocol Versioning
- No version negotiation or compatibility checks yet; assume backward compatibility.

## Exit Codes
- 0 = completed, 1 = failed.
- Other statuses are conveyed via stdout/stderr and message bus.
