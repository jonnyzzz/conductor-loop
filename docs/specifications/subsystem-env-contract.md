# Environment & Invocation Contract Subsystem

## Overview
Defines the invocation contract between run-agent (Go binary; including task and job) and spawned agents. This includes internal environment variables used for run tracking, the prompt/context injection that agents use to discover paths, and safety rules that prevent agents from mutating runner-owned environment variables across process boundaries.

## Goals
- Document internal JRUN_* variables for run tracking.
- Document prompt/context injection for run and task paths.
- Clarify that agents should not rely on environment variables for workflow.
- Ensure child processes can find run-agent via PATH.
- Prevent runner-owned variables from being overridden by caller-provided environment.

## Non-Goals
- Standardizing backend token names beyond config injection.
- Enforcing env vars at the OS or sandbox level.
- Providing a full process supervisor API (handled by Runner & Orchestration).

## Responsibilities
- Specify required internal variables and their meanings.
- Define prompt and context injection rules.
- Define read-only vs writable classification for any injected values.
- Describe failure behavior when required variables are missing.
- Define reserved environment variable handling and override rules.

## Internal Environment Variables (Runner-Only)
These are set by run-agent for internal bookkeeping. Agents must not reference them.
- JRUN_PROJECT_ID (required; internal): project identifier for the current run.
- JRUN_TASK_ID (required; internal): task identifier for the current run.
- JRUN_ID (required; internal): run identifier (timestamp + PID format).
- JRUN_PARENT_ID (required; internal): parent run identifier for lineage tracking.

## Reserved Environment Variables and Safety
- Runner-owned variables are reserved and must be overwritten on spawn, even if present in the parent environment or provided by a caller.
- Reserved prefixes include JRUN_ and any future CONDUCTOR_ runner internals.
- No MESSAGE_BUS or TASK_MESSAGE_BUS environment variables are part of this contract; message bus paths are discovered from storage layout and prompt preamble.
- The runner must drop or overwrite any caller-provided values for reserved keys before spawning agents.

## Prompt/Context Injection (Agent-Visible)
- run-agent prepends the prompt with absolute paths for the task folder and run folder.
- The run folder is provided as a prompt label (e.g., RUN_FOLDER=/path/to/run), not as an environment variable.
- Paths are normalized using OS-native conventions (Go filepath.Clean).
- Root agents rely on CWD (task folder) and prompt preamble; sub-agents rely on the prompt preamble.
- The prompt preamble includes explicit instructions to write final output to output.md in the run folder.
- No current date or time is injected into the prompt preamble; agents access system time themselves.

## Injection Rules
- run-agent always sets JRUN_* internally before spawning agents.
- run-agent prepends its own binary location to PATH for child processes, and should avoid duplicate entries when possible.
- Backend tokens are injected into the agent process environment via config (backend-specific); agents must not rely on them for workflow.
- No agent-writable environment variables are defined.
- Agents inherit the full parent environment (no sandbox restrictions in MVP).

## Error Messaging
- Missing required JRUN_*: fail fast.
- Error messages must not instruct agents to set env vars manually.

## Signal Handling
- run-agent forwards SIGTERM to the agent process group.
- Grace period: 30 seconds wait after SIGTERM before sending SIGKILL.
- Termination events (STOP, CRASH) are logged to the message bus by run-agent.

## Security Notes
- Tokens and credentials are injected into agent processes from config; the naming is backend-specific and not standardized here.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-agent-protocol.md
- subsystem-storage-layout.md
