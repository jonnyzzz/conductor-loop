# Runner & Orchestration - Questions

- Q: What is the authoritative completion signal and status taxonomy for a task (TASK_STATE.md vs MESSAGE-BUS vs exit code)?
  Proposed default: TASK_STATE.md is authoritative with status: completed|blocked|failed; MESSAGE-BUS mirrors status; exit code only influences backoff.
  A: We ask the root agent in the prompt to create the DONE file, as the marker of completion. The file is removed if the user wants it to restart once again.

- Q: Should run-task perform compaction/fact propagation between root-agent iterations (Ralph loop)?
  Proposed default: Optional compaction between iterations, enabled by default and configurable in settings.
  A: We have git repositories and message bus as propagation. Research if more approaches are needed.

- Q: When the root agent marks COMPLETED but sub-agents are still running, should run-task wait, detach, or terminate them?
  Proposed default: Leave sub-agents running by default; allow config to wait or terminate; always log the decision to MESSAGE-BUS.
  A: Yes, root task wait for all sub-tree to complete (it can be that some agents stuck). If the root agent exits sooner, we need to restart it to catch up and potentially update the outcomes.

- Q: Should run-agent detach agent processes from the parent terminal so they survive parent exit?
  Proposed default: Yes; run-agent tracks detached PIDs and exposes a stop command.
  A: Yes, run-agent tool should start all agent processes detached from itself, so they survive parent exit, and they survive agent kills.

- Q: What is the canonical stop/kill sequence for an agent run, and where is it recorded?
  Proposed default: Graceful stop (SIGTERM) -> wait N seconds -> SIGKILL; record in run metadata and MESSAGE-BUS.
  A: there is agent-run stop command with runid parameter. Agent can call it to clearly stop the run. Killing the run-agent process should not be enough.

- Q: How should idle-timeout detect true inactivity vs waiting on sub-agents?
  A: a run idle only if no stdout/stderr AND no active child runs for N seconds.

- Q: What is the policy when run-task is invoked for a task that already has an active root run?
  A: Use a per-task lock; second invocation exits with a clear message (optional attach/read-only mode).

- Q: Should runner enforce max-concurrent agents and support "park and wait" instead of failing immediately?
  Proposed default: 
  A: Yes; park and poll for free slots when limit is reached. It can still yield to starvation if all agents are waiting. We can fix that later.

- Q: What safety mechanism prevents infinite restart loops on repeated fast failures?
  A: Exponential backoff plus a circuit breaker after N quick failures (<2 min each). Raise WARN message to the message bus.

- Q: Should run-agent post START/STOP events to MESSAGE-BUS in addition to run metadata?
  Proposed default: Yes; include run_id, parent_run_id, pid, prompt/output paths, and exit_code on STOP.
  A: We keep run metadata in the message bus for audit purposes. So just put that message onces. We need that task graph for UI, so it should be cheap to build.

- Q: Where should sandbox detection/agent health marking live (run-agent.sh vs runner binary)?
  Proposed default: Runner binary detects sandbox issues and marks an agent backend as degraded before spawning.
  A: We move that as closer to agent process as possible. So each run-agent process should take care of it's agent. It should work wihtout consolidation or orchestration.

- Q: What is the canonical config format and migration strategy for ~/run-agent/config.json?
  Proposed default: JSON with // comments (JSON5-style) and a version field; load merges defaults, logs deprecated keys, and never deletes user values.
  A: We use HCL format from hashicorp/hcl. Each setting comes with optional value and included default value and comment. We update default values as part of the migrations, when new values are implemented. For now, just use HCP and create a stub file if there is no file under the user home.

- Q: Where are backend tokens stored and how are they injected into agent processes without leaking to logs?
  Proposed default: Store in config.json via @file references or OS keychain; pass via env vars and redact from logs.
  A: Agent tokens are loaded and set as environment variables. We do not care if they are leaked to logs.

- Q: How is multi-backend selection configured per project/task (priority, fallback, health checks)?
  Proposed default: Global agents.order in config.json with per-project overrides; failures mark backend degraded temporarily.
  A: Non goal for now. Let's assume there is 1 machine and all processes are local.

- Q: Should run-agent enforce CPU/memory limits per agent process?
  Proposed default: Optional ulimit/cgroup constraints configured in config.json.
  A: No, let's just start processes as is and let the OS do its job.
