# Runner & Orchestration Subsystem

## Overview
This subsystem owns how agents are started, restarted, and coordinated. It defines the `run-agent` binary (implemented in Go, including the `task` and `job` subcommands) responsible for spawning agent runs, tracking lineage, and enforcing the root "Ralph" restart loop until completion.

## Goals
- Start agents with a consistent run layout and metadata (see Storage & Data Layout).
- Track parent-child relationships and restart chains (previous_run_id).
- Keep the root orchestrator running until the task is done (DONE marker).
- Provide a stable CLI for starting tasks and agents.
- Rotate or auto-select agent types to avoid stalling.

## Non-Goals
- Implementing the message bus itself (handled by Message Bus Tooling).
- Implementing the monitoring UI (handled by Monitoring UI).
- Enforcing strict sandboxing or resource limits (not required yet).

## Responsibilities
- Validate required environment variables for run tracking (internal to runner).
- Create run directories and record run metadata.
- Launch agents detached from the parent process.
- Manage the Ralph restart loop for the root agent.
- Ensure message-bus handling agents are started per incoming message (typically orchestrated by the root agent; no dedicated poller/heartbeat in MVP).
- Record START/STOP/CRASH events for auditability.
- Ensure the run-agent binary is available on PATH for spawned agents.

## Components
### run-agent job (subcommand)
- Starts one agent run and blocks until completion.
- Detaches from the controlling terminal (so the agent survives parent exit) but still waits on the agent PID for exit status.
- Maintains process group ID for stop/kill.
- Generates run_id (timestamp + PID) and creates the run folder.
- Writes run metadata (run-info.yaml) and output files.
- **Output Logic**: If `output.md` does not exist after the agent process terminates, creates `output.md` using the content of `agent-stdout.txt`.
- Posts run START/STOP events to message bus.
- Prepends its own binary location to PATH for child processes.

### run-agent task (subcommand)
- Creates/locates project and task folders.
- Validates TASK.md is non-empty.
- Starts root agent and enforces the Ralph restart loop.
- Task is complete only when DONE exists AND all child runs have exited.
- **Ralph Loop Termination Logic**:
  - Check for DONE marker BEFORE starting/restarting root agent.
  - If **DONE exists**:
    - Enumerate all active children (runs with empty end_time and non-empty parent_run_id).
    - If **No Active Children**: Task is complete. Exit successfully.
    - If **Active Children Running**:
      - Log "Waiting for N children to complete: [run_ids...]"
      - Post INFO message to message bus with child run IDs.
      - Poll children every 1s using `kill(-pgid, 0)` checks.
      - Timeout after 300s (configurable via config.child_wait_timeout).
      - On timeout: Log WARNING to message bus, proceed to completion (orphan children).
      - Once all children exit: Task is complete. Exit successfully.
      - **Do NOT restart root agent** (root already declared completion via DONE).
  - If **DONE missing**: Start/restart Root Agent (subject to max_restarts).
  - Between restart attempts, pause 1s to avoid tight loops.
- `run-task`/`run-task.sh` are thin wrappers (if present) that call `run-agent task`.

### run-agent serve (subcommand)
- Serves the Monitoring UI (TypeScript + React, embedded static assets) and message bus API endpoints (REST + SSE).

### run-agent bus (subcommand)
- Provides CLI access for posting, polling, and streaming message bus entries.

### run-agent stop (subcommand)
- Stops a run by run_id.
- Sends SIGTERM to the agent process group, then SIGKILL after a grace period.
- Records STOP event in run metadata + message bus.

## Configuration
- Config file format: HCL (Hashicorp).
- Stored under user home at ~/run-agent/config.hcl (global config only). Per-project/task config is not required yet.
- Precedence (when/if per-project/task config is added): CLI > env vars > task config > project config > global config.
- Configurable items:
  - max restarts / time budget for Ralph loop
  - child_wait_timeout (default 300s, used when DONE exists but children running)
  - child_poll_interval (default 1s, used when waiting for children)
  - idle/stuck thresholds
  - agent selection weights and ordering
  - delegation depth (max 16 by default)
  - supported agent backends/providers list
  - projects_root (override for ~/run-agent)
  - deploy_ssh_key (optional; used when backing storage with a git repo)
- Token values may be provided inline or via @/path/to/token file references.
- @file references are resolved at config load; missing files are treated as configuration errors.
- Schema validation:
  - Embed schema definition in Go binary (see subsystem-runner-orchestration-config-schema.md).
  - Validate config.hcl on startup with clean, explanatory error messages.
  - Provide `run-agent config schema` command to extract/display the schema.
  - Provide `run-agent config init` command to create/update config.hcl with comments and defaults.

## Run ID / Timestamp Rules
- Canonical timestamp format: UTC `YYYYMMDD-HHMMSSMMMM-PID` (lexically sortable).
- run_id is generated by run-agent using the same format.
- Each Ralph restart creates a new run; run-info records previous_run_id to form a chain.
- No lockfile coordination is required; uniqueness is ensured by timestamp+PID and retry on collisions.

## Agent Selection
- Default: round-robin across available agent types.
- "I'm lucky" mode: random selection with uniform weighting (weights configurable).
- On failure, mark the backend as degraded temporarily and try the next agent.
- Selection decisions may consult recent message bus events; each run should record its chosen agent/backend in run-info and/or message bus entries.

## Agent Backends
- Native agent types include: codex, claude, gemini, perplexity.
- xAI integration is deferred post-MVP (see subsystem-agent-backend-xai.md).
- Each agent type has a dedicated design document (see subsystem-agent-backend-*.md).

## Idle / Stuck / Waiting Detection
- Use last stdout/stderr timestamp + message bus activity.
- Idle: no stdout/stderr AND all children idle for N seconds (default 5m).
- Stuck: no stdout/stderr for a longer threshold (default 15m).
- Waiting: last message bus entry is QUESTION.
- On idle timeout: kill the process, log to message bus, and let parent recover.

## Root Orchestrator Prompt Contract
The root agent prompt must include:
- Read TASK_STATE.md and TASK-MESSAGE-BUS.md on start.
- Use message bus only for communication.
- Regularly poll message bus for new messages.
- Write facts as FACT-*.md (YAML front matter required).
- Update TASK_STATE.md with a short free-text status.
- **IMPORTANT**: If delegating work to children, wait for children to complete and post results BEFORE writing DONE.
- Monitor message bus for CHILD_DONE or result messages from children if aggregation is needed.
- Write final results to output.md in the run folder.
- Create DONE file ONLY when the task is complete (all work finished, including children if applicable).
- Post an INFO/OBSERVATION message to the bus when writing DONE.
- Delegate sub-tasks by starting sub agents via run-agent.

## Error Handling
- Missing env vars -> fail fast (error messages must not instruct agents to set env).
- Transient backend errors -> exponential backoff (1s, 2s, 4s; max 3 tries).
- Auth/quota errors -> fail fast.
- Message-bus handler crashes -> log and let root agent decide recovery.
- No proactive credential validation/refresh in MVP; failures are handled at spawn time.

## Concurrency & Coordination
- No global coordination across tasks; best-effort heuristics may inspect message bus state for backpressure.
- On supervisor restart, re-discover running child PIDs via run-info and post a SUPERVISOR_RESTART event to the message bus (best-effort).

## Observability
- run-info.yaml contains:
  - run_id, project_id, task_id, parent_run_id, previous_run_id
  - agent type, pid/pgid, start/end time, exit code
  - paths to prompt/output/stdout/stderr
  - commandline (optional; full command used to start the agent)
- START/STOP/CRASH events are posted to message bus for UI reconstruction.

## Agent Design Patterns

Root agents should follow one of these patterns when delegating work to child agents:

### Pattern A: Parallel Delegation with Aggregation (Recommended)
```
Root: Spawn N children for parallel subtasks
Root: Monitor message bus for CHILD_DONE messages
Root: Wait for all children to report completion
Root: Aggregate results from children's outputs
Root: Write final output.md
Root: Write DONE
Root: Exit
```

**Key principle:** Root writes DONE only after children complete and results are aggregated.

### Pattern B: Fire-and-Forget Delegation
```
Root: Spawn N children for independent subtasks
Root: Write DONE immediately (root's work is done)
Root: Exit (children continue independently)
```

**Key principle:** Root does not need children's results. Ralph loop will wait for children to exit before completing task.

### Anti-pattern (DO NOT USE)
```
Root: Spawn children
Root: Write DONE immediately
Root: Expect to be restarted to aggregate results
```

This is incorrect - the Ralph loop will NOT restart root after DONE is written. If root needs to aggregate results, it must wait for children BEFORE writing DONE (use Pattern A).

## Security / Permissions
- Tokens are read from config and injected via environment variables.
- No sandboxing or resource limits enforced yet.
