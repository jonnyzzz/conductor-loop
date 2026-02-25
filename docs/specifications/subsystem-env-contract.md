# Environment & Invocation Contract Subsystem

## Overview
Defines the invocation contract between run-agent (Go binary; including task and job) and spawned agents. This includes internal environment variables used for run tracking, informational path variables, the prompt/context injection that agents use to discover paths, and safety rules that prevent agents from mutating runner-owned environment variables across process boundaries.

## Goals
- Document internal JRUN_* variables for run tracking.
- Document informational path variables injected into agent environment.
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
- Specify informational path variables and their meanings.
- Define prompt and context injection rules.
- Define read-only vs writable classification for any injected values.
- Describe failure behavior when required variables are missing.
- Define reserved environment variable handling and override rules.

## Internal Environment Variables (Runner-Owned)
These are set by run-agent for internal run tracking. They are overwritten on each spawn, even if present in the parent environment. Agents must not reference them for workflow; they are consumed by `run-agent` sub-command invocations within the agent.

| Variable | Description |
|----------|-------------|
| `JRUN_PROJECT_ID` | Project identifier for the current run |
| `JRUN_TASK_ID` | Task identifier for the current run |
| `JRUN_ID` | Run identifier (format: `YYYYMMDD-HHMMSSffff-PID-SEQ`) |
| `JRUN_PARENT_ID` | Parent run identifier for lineage tracking (empty for root runs) |

## Informational Environment Variables (Agent-Visible)
These are injected as convenience variables. Agents MAY use them but MUST NOT depend on them being static — sub-agents may override them when redirecting to different task/run contexts.

| Variable | Description |
|----------|-------------|
| `JRUN_RUNS_DIR` | Absolute path to the runs directory for the current task |
| `JRUN_MESSAGE_BUS` | Absolute path to the task-level message bus file (`TASK-MESSAGE-BUS.md`) |
| `JRUN_TASK_FOLDER` | Absolute path to the task directory |
| `JRUN_RUN_FOLDER` | Absolute path to the current run directory |
| `JRUN_CONDUCTOR_URL` | URL of the conductor API server (injected if configured; may be absent) |

All path variables are normalized using `filepath.Clean` (OS-native separators).

## Reserved Environment Variables and Safety
- `JRUN_*` variables are runner-owned; overwritten on every spawn regardless of parent environment or caller-provided env.
- `CONDUCTOR_*` prefix is reserved for future runner internals.
- Callers cannot override reserved variables — the runner enforces correct values before spawning.
- Informational variables (JRUN_RUNS_DIR, JRUN_MESSAGE_BUS, JRUN_TASK_FOLDER, JRUN_RUN_FOLDER, JRUN_CONDUCTOR_URL) are NOT blocked from override — agents may redirect them for sub-tasks.

## Prompt/Context Injection (Agent-Visible)
- run-agent prepends the prompt with absolute paths for the task folder and run folder.
- The preamble also includes an instruction to write final output to `output.md` in the run folder.
- Paths are normalized using OS-native conventions (Go filepath.Clean).
- No current date or time is injected into the prompt preamble; agents access system time themselves.
- On restart attempts > 0, the runner prepends "Continue working on the following:" before the task prompt. The preamble is always included, even on restarts.

Example preamble:
```text
JRUN_TASK_FOLDER=/absolute/path/to/task
JRUN_RUN_FOLDER=/absolute/path/to/run
Write output.md to /absolute/path/to/run/output.md
```

## Injection Rules
- run-agent always sets JRUN_* and informational path variables before spawning agents.
- run-agent prepends its own binary location to PATH for child processes, avoiding duplicate entries when possible.
- Backend tokens are injected into the agent process environment via config (backend-specific); see subsystem-runner-orchestration-config-schema.md for mappings.
- Agents inherit the full parent environment (no sandbox restrictions in MVP).
- `CLAUDECODE` env var (set by Claude CLI) passes through automatically via inherited environment; no special handling needed.

## Error Messaging
- Missing required JRUN_* when consumed by sub-command: fail fast.
- Error messages must not instruct agents to set env vars manually.

## Signal Handling
- run-agent forwards SIGTERM to the agent process group.
- Grace period: 30 seconds wait after SIGTERM before sending SIGKILL.
- Termination events (STOP, CRASH) are logged to the message bus by run-agent.

## Security Notes
- Tokens and credentials are injected into agent processes from config; naming is backend-specific (see subsystem-runner-orchestration-config-schema.md).

## Related Files
- subsystem-runner-orchestration.md
- subsystem-agent-protocol.md
- subsystem-storage-layout.md
