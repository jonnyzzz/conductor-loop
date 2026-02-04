# Environment & Invocation Contract Subsystem

## Overview
Defines the environment variables and invocation contract between run-agent, run-task, and spawned agents. This subsystem standardizes which variables are injected, their meanings, and how agents are expected to treat them.

## Goals
- Document required environment variables for run tracking.
- Clarify which variables are internal-only vs agent-visible.
- Standardize RUN_FOLDER injection for sub-agents.
- Ensure error messages do not instruct agents to set env vars manually.

## Non-Goals
- Defining backend-specific token naming beyond config injection.
- Enforcing env vars at the OS/sandbox level.
- Providing a full process supervisor API (handled by Runner & Orchestration).

## Responsibilities
- Specify required variables and their meanings.
- Define when variables are set (root vs sub-agent).
- Define read-only vs writable classification.
- Describe failure behavior when required variables are missing.

## Variable Contract (MVP)
- JRUN_PROJECT_ID (required; internal): project identifier for the current run.
- JRUN_TASK_ID (required; internal): task identifier for the current run.
- JRUN_ID (required; internal): run identifier (timestamp + PID format).
- RUN_FOLDER (required for sub-agents; agent-visible): absolute path to the run folder to use for prompts, outputs, and temp files.
- JRUN_PARENT_ID (required; internal): parent run identifier for lineage tracking. Name is TBD if it differs from this label.

## Injection Rules
- run-task/run-agent set JRUN_* variables before launching any agent process.
- Sub-agents inherit JRUN_* variables and receive a RUN_FOLDER pointing at their run directory (under task/runs/).
- Agents treat RUN_FOLDER as read-only and must place output.md in that folder.

## Error Messaging
- Missing required env vars -> fail fast.
- Error messages must not instruct agents to set env vars manually.

## Security Notes
- Tokens and credentials are injected into agent processes from config; the naming is backend-specific and not standardized here.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-agent-protocol.md
- subsystem-storage-layout.md
