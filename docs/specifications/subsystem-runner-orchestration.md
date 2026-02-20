# Runner & Orchestration Subsystem

## Overview
This subsystem owns how agents are started, restarted, and coordinated. It defines the run-agent CLI and its task/job flows, implements the Ralph restart loop, and ensures runs are persisted to the storage layout. It also enforces safe environment propagation, process detachment, and start/stop observability.

The runner is the only component that may spawn agent processes. All inter-agent communication happens via the message bus and on-disk artifacts.

## Goals
- Start agents with a consistent run layout and metadata (see Storage & Data Layout).
- Track parent-child relationships and restart chains (previous_run_id).
- Keep the root orchestrator running until the task is done (DONE marker).
- Provide a stable CLI for starting tasks and jobs.
- Rotate or auto-select agent types to avoid stalling.
- Enforce delegation depth limits to keep tasks small.
- Preserve runner-owned environment variables and PATH injection.

## Non-Goals
- Implementing the message bus itself (handled by Message Bus Tooling).
- Implementing the monitoring UI (handled by Monitoring UI).
- Enforcing strict sandboxing or resource limits (not required yet).
- Long-lived agent sessions; agents should complete a scoped task and exit.

## Responsibilities
- Validate required environment variables for run tracking (internal to runner).
- Create run directories and record run metadata.
- Launch agents detached from the parent process (new process group/session).
- Manage the Ralph restart loop for the root agent.
- Ensure message bus handling agents are started per incoming message (typically orchestrated by the root agent; no dedicated poller/heartbeat in MVP).
- Record START/STOP/CRASH events for auditability.
- Ensure the run-agent binary is available on PATH for spawned agents.
- Guard runner-owned environment variables from being overridden by caller-provided env.

## Components
### run-agent job (subcommand)
- Starts one agent run and blocks until completion.
- CLI agents are started as detached processes but the runner still waits on the PID for exit status.
- REST-backed agents (perplexity, xai) execute in-process; run-info.yaml still records start/stop times and exit codes, and PID refers to the runner process.
- Generates run_id (timestamp + PID) and creates the run folder.
- Writes run metadata (run-info.yaml) and output files.
- Output logic: If output.md does not exist after the agent process terminates, create output.md using agent-stdout.txt as fallback.
- Posts run START/STOP events to the message bus.
- Prepends its own binary location to PATH for child processes.

### run-agent task (subcommand)
- Creates or locates project and task folders.
- Validates TASK.md is non-empty.
- Starts root agent and enforces the Ralph restart loop.
- Task is complete only when DONE exists AND all child runs have exited.

Ralph loop termination logic:
1. Check for DONE marker before starting or restarting the root agent.
1. If DONE exists and there are no active children, record completion and exit.
1. If DONE exists and children are still running, log waiting state and poll until they exit or timeout.
1. If timeout expires, log WARNING and complete the task without restarting the root agent.
1. If DONE is missing, start or restart the root agent (subject to max_restarts), with a small delay between attempts.

Between restart attempts, pause restart_delay to avoid tight loops.

### run-agent serve (subcommand)
- Serves the Monitoring UI and message bus API endpoints (REST + SSE).
- Planned for post-MVP if not yet implemented.

### run-agent bus (subcommand)
- Provides CLI access for posting, polling, and streaming message bus entries.
- Planned for post-MVP if not yet implemented.

### run-agent stop (subcommand)
- Stops a run by run_id.
- Sends SIGTERM to the agent process group, then SIGKILL after a grace period.
- Records STOP event in run metadata and message bus.
- Planned for post-MVP if not yet implemented.

## Prompt Construction & Restart Prefix
The runner prepends a prompt preamble, for example:

```text
TASK_FOLDER=/absolute/path/to/task
RUN_FOLDER=/absolute/path/to/run
Write output.md to /absolute/path/to/run/output.md
```

On restart attempts greater than 0, the runner should prefix the task prompt with:

```text
Continue working on the following:
```

The preamble is always included, even on restarts.

## Configuration
Config file format: HCL (Hashicorp).
Stored under user home at ~/run-agent/config.hcl (global config only). Per-project/task config is not required yet.
Precedence (when/if per-project/task config is added): CLI > env vars > task config > project config > global config.

Configurable items include:
- max_restarts / time budget for Ralph loop
- child_wait_timeout (default 300s, used when DONE exists but children running)
- child_poll_interval (default 1s, used when waiting for children)
- restart_delay (default 1s)
- idle/stuck thresholds
- agent selection strategy and weights
- delegation depth limit (default 16)
- supported agent backends/providers list
- projects_root (override for ~/run-agent)
- deploy_ssh_key (optional; used when backing storage with a git repo)

Token values may be provided inline or via token_file references.
token_file is read, trimmed, and used as the token value.

Schema validation:
- Embed schema definition in Go binary (see subsystem-runner-orchestration-config-schema.md).
- Validate config.hcl on startup with clean, explanatory error messages.
- Provide run-agent config schema command to extract/display the schema.
- Provide run-agent config init command to create or update config.hcl with comments and defaults.

Note: The current implementation still loads YAML config files. HCL is the target format. See subsystem-runner-orchestration-QUESTIONS.md for the migration decision.

## Run ID / Timestamp Rules
- Canonical timestamp format: UTC YYYYMMDD-HHMMSSMMMM-PID (lexically sortable).
- run_id is generated by run-agent using the same format.
- Each Ralph restart creates a new run; run-info records previous_run_id to form a chain.
- No lockfile coordination is required; uniqueness is ensured by timestamp + PID and retry on collisions.

## Agent Selection
- Default: round-robin across available agent types.
- I'm lucky mode: random selection with configurable weights.
- On failure, mark the backend as degraded for a cooldown and try the next.
- Selection decisions may consult recent message bus events.
- Each run records its chosen agent/backend in run-info.yaml and message bus entries.

## Agent Backends
- CLI agents: codex, claude, gemini.
- REST agents: perplexity, xai.
- Each agent type has a dedicated design document (see subsystem-agent-backend-*.md).

## Idle / Stuck / Waiting Detection
- Use last stdout or stderr timestamp plus message bus activity.
- Idle: no stdout or stderr AND all children idle for N seconds (default 5m).
- Stuck: no stdout or stderr for a longer threshold (default 15m).
- Waiting: last message bus entry is QUESTION.
- On idle timeout: kill the process, log to message bus, and let parent recover.

## Root Orchestrator Prompt Contract
The root agent prompt must include:
- Read TASK_STATE.md and TASK-MESSAGE-BUS.md on start.
- Use message bus only for communication.
- Regularly poll message bus for new messages.
- Write facts as FACT-*.md (YAML front matter required).
- Update TASK_STATE.md with a short free-text status.
- If delegating work to children, wait for children to complete and post results before writing DONE.
- Monitor message bus for CHILD_DONE or result messages from children if aggregation is needed.
- Write final results to output.md in the run folder.
- Create DONE file only when the task is complete (all work finished, including children if applicable).
- Post an INFO message to the bus when writing DONE.
- Delegate sub-tasks by starting sub agents via run-agent.
- Respect the delegation depth limit and split work by subsystem when possible.

## Error Handling
- Missing env vars: fail fast (error messages must not instruct agents to set env).
- Transient backend errors: exponential backoff (1s, 2s, 4s; max 3 tries).
- Auth or quota errors: fail fast.
- Message bus handler crashes: log and let root agent decide recovery.
- No proactive credential validation or refresh in MVP; failures are handled at spawn time.

## Concurrency & Coordination
- No global coordination across tasks; best-effort heuristics may inspect message bus state for backpressure.
- On supervisor restart, re-discover running child PIDs via run-info and post a SUPERVISOR_RESTART event to the message bus (best-effort).

## Observability
- run-info.yaml is the canonical per-run audit record (start or end times, pid or pgid, paths, and exit codes).
- RUN_START, RUN_STOP, and RUN_CRASH events are posted to the task message bus for UI reconstruction.
- No separate start/stop log file in MVP; the combination of run-info.yaml and message bus is the source of truth.

## Security / Permissions
- Tokens are read from config and injected via environment variables.
- Runner-owned env vars (JRUN_*) are overwritten on spawn; callers cannot override them.
- Message bus paths are not exposed via environment variables; agents discover paths via prompt preamble and the storage layout.
- No sandboxing or resource limits enforced yet.
