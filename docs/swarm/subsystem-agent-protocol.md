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
- Agents MUST write updates to TASK_STATE.md each cycle (root only).
- Agents MUST write final results to output.md in their run folder.
- Agents MUST use run-agent bus tooling; direct file appends are disallowed.
- Agents SHOULD log progress to stderr during long operations.

## Run Folder Ownership
- No OWNERSHIP.md file.
- run-agent injects a RUN_FOLDER path into sub-agent prompts.
- Prompt preamble example: `RUN_FOLDER=<path>, use this folder for prompts, task outputs, and other temp files. Put the results to $RUN_FOLDER/output.md file`.
- Agents write prompts/outputs/temporary files only inside RUN_FOLDER.
- Ownership is conceptual (prompt-guided), not enforced by files.
- Parents may read child output/TASK_STATE for monitoring; policy does not restrict this.

## Message Bus Protocol
- Agents do not emit START/STOP (runner-only).
- Use TASK-MESSAGE-BUS for task-scoped updates/questions.
- Use PROJECT-MESSAGE-BUS for cross-task facts (typically via root agent).
- Thread replies and corrections with parents[].
- Poll for new messages as often as possible; read only new content.

## Task State
- TASK_STATE.md is free text written by root agent.
- No strict schema; keep it short and current.

## Facts
- FACT files are Markdown with YAML front matter.
- Naming: FACT-<timestamp>-<name>.md (project), TASK-FACTS-<timestamp>.md (task).
- Promotion to project-level facts is decided by the root agent; task agents can propose via message bus.

## Delegation & Depth
- Max delegation depth: 16 (configurable in global settings).
- run-task should fail new spawn attempts beyond the limit.

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

## Cancellation Protocol
- Runner sends SIGTERM to agent pgid, waits 30s, then SIGKILL.
- Agent should flush message bus + TASK_STATE on SIGTERM when possible.

## Environment Variables
- JRUN_* variables are implementation details (agents should not reference them).
- Error messages must not instruct agents to set env vars.
- RUN_FOLDER is injected for sub-agents; treat it as read-only.

## Protocol Versioning
- No version negotiation or compatibility checks yet; assume backward compatibility.

## Exit Codes
- 0 = completed, 1 = failed.
- Other statuses are conveyed via stdout/stderr and message bus.
