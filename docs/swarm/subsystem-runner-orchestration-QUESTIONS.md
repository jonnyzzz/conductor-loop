# Runner & Orchestration - Questions

- Q: On Ralph restarts, how are root runs linked for UI and recovery (parent/previous chain)?
  Proposed default: Each restart creates a new run; run metadata includes previous_run_id to form a chain.
  A: TBD.

- Q: What locking mechanism prevents concurrent run-task instances for the same task, and how are stale locks recovered?
  Proposed default: task.lock with PID+start time; stale if PID missing or lock age > TTL; allow --force to clear.
  A: TBD.

- Q: How does “lucky” agent selection persist state across runs (round-robin index, health/degraded backoff)?
  Proposed default: Store last-used index and degraded backends (from backend health TTL cache) in a per-project runner state file.
  A: TBD.

- Q: How are message-bus pollers identified, deduplicated, and restarted on crash?
  Proposed default: One poller per project/task with pid+heartbeat file; restart if heartbeat stale; log to MESSAGE-BUS.
  A: TBD.

- Q: How should orphaned runs be handled when run-agent exits but the agent process continues?
  Proposed default: On supervisor start, scan PIDs; mark orphan if pid alive but no parent; keep monitoring; if pid dead and no exit_code, mark crashed.
  A: TBD.

- Q: What is the explicit stop/kill mechanism for detached agent process trees, and how is it recorded?
  Proposed default: Record process group ID; stop uses SIGTERM to pgid then SIGKILL after grace; record STOP in run-info + MESSAGE-BUS.
  A: TBD.

- Q: How should idle vs stuck vs waiting-for-user be detected, and what thresholds apply?
  Proposed default: Use last stdout/stderr time plus MESSAGE-BUS activity; idle at 5m, stuck at 15m, waiting if last bus entry is QUESTION.
  A: TBD.

- Q: What is the configuration precedence/merge order across global config, project config, task config, env vars, and CLI flags?
  Proposed default: CLI > env vars > task config > project config > global config; unknown keys warn; defaults fill missing.
  A: TBD.

- Q: How are backend credentials validated and refreshed without blocking every run?
  Proposed default: Async health checks with TTL cache; fail run only if selected backend is degraded at spawn.
  A: TBD.

- Q: What is the error propagation policy when a sub-agent crashes (notify parent, retry, fail task)?
  Proposed default: Runner posts CRASH to MESSAGE-BUS; parent receives exit_code; retry behavior is task-config driven.
  A: TBD.

- Q: What is the retry policy for transient backend failures (rate limits, network errors) vs permanent failures?
  Proposed default: Exponential backoff for transient (1s, 2s, 4s; max 3 retries); fail fast for auth/quota errors.
  A: TBD.

- Q: How are concurrent run-task invocations across different tasks coordinated (backend pool limits, shared resource contention)?
  Proposed default: Per-backend semaphore with max_concurrent limit from config; queued tasks get QUEUED status in MESSAGE-BUS.
  A: TBD.

- Q: What is the behavior when the Ralph supervisor itself crashes or restarts (PID tracking, poller recovery)?
  Proposed default: On restart, discover running agents via pid files, reattach pollers using heartbeat files, post SUPERVISOR_RESTART to MESSAGE-BUS.
  A: TBD.
