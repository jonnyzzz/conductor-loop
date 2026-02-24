# Concurrency Architecture

This document describes how Conductor Loop coordinates parallel work and synchronizes shared state.

Primary source documents:
- `docs/facts/FACTS-architecture.md`
- `docs/facts/FACTS-runner-storage.md`

Implementation references:
- `internal/runner/ralph.go`
- `internal/runner/semaphore.go`
- `internal/runner/task.go`
- `internal/api/root_task_planner.go`
- `internal/messagebus/messagebus.go`
- `internal/messagebus/lock.go`
- `internal/taskdeps/taskdeps.go`
- `internal/runner/stop_unix.go`

## Ralph Loop

`RalphLoop.Run()` is the restart supervisor for a root task.

Restart semantics:
- The loop checks `<task_dir>/DONE` before each root attempt and again after each attempt.
- A zero exit code does not terminate the loop by itself. If `DONE` is still absent and restart budget remains, a new attempt is started.
- The loop stops launching new root attempts once `DONE` exists.
`DONE` checks:
- `DONE` must be a file. If `DONE` is a directory, the loop treats it as an error.
- When `DONE` is present, the loop transitions to child-drain behavior rather than restart behavior.
`wait_timeout` and `poll_interval`:
- The child-drain phase uses `waitTimeout` and `pollInterval` in the loop implementation (documented as child wait timeout/poll interval in facts).
- Defaults are `waitTimeout=300s` and `pollInterval=1s`.
- If children are still active at timeout, the loop logs a warning and completes without restarting the root.

## Run Concurrency

### `max_concurrent_runs` (all runs)

All run executions pass through one package-level semaphore in `internal/runner/semaphore.go`.

- The semaphore is initialized from `defaults.max_concurrent_runs`.
- `n > 0` creates a bounded channel semaphore.
- `n <= 0` means unlimited run concurrency.
- `acquireSem(ctx)` blocks while all slots are occupied; `releaseSem()` frees a slot.
- This gate applies to all runs that execute through `runJob` (root attempts and child jobs), not just root tasks.

### `max_concurrent_root_tasks` (root tasks)

Root task admission is separately controlled in the API layer by a planner.

- Enabled by `defaults.max_concurrent_root_tasks`.
- Implemented by `rootTaskPlanner` with persistent state at `.conductor/root-task-planner.yaml`.
- Planner entries are queued/running; scheduling promotes queued entries while running count is below the configured limit.
- Scope is root task starts submitted through the API planner path, not all run attempts.

## Message Bus

The message bus is append-only with single-writer synchronization.

Exclusive write lock:
- Writers open with `O_APPEND` and acquire an exclusive lock before append.
- On Unix, locking uses `flock(LOCK_EX|LOCK_NB)` with polling until lock timeout.
Append retries:
- Lock acquisition timeout is bounded (default `10s`).
- On lock-timeout failures, append retries with exponential backoff (default `3` attempts, base `100ms`).
Lockless reads:
- Read paths (`ReadMessages`, `ReadLastN`) do not take the write lock.
- This gives lockless-read behavior on Unix while writes are serialized.

## Task Dependencies

Task dependencies form a per-project DAG through `depends_on`.

DAG execution:
- Dependencies are stored in `TASK-CONFIG.yaml` as `depends_on`.
- Tasks do not enter Ralph execution until dependencies are satisfied.
Cycle detection:
- `ValidateNoCycle` builds the dependency graph and rejects updates that introduce a cycle.
- Rejections use explicit dependency cycle errors.
Blocking waits:
- `waitForDependencies` performs a blocking wait loop with periodic polling.
- Default dependency polling interval is `2s` when not overridden.
- While blocked, it posts `PROGRESS` messages listing unresolved dependencies.
- When blockers clear, it posts a `FACT` message and proceeds to execution.
- If the current task receives `DONE` while blocked, dependency waiting exits immediately.

## Process Groups

Conductor uses process groups to propagate stop signals across an entire run tree.

Signal propagation:
- On Unix, stop logic sends `SIGTERM` to `-PGID` (`syscall.Kill(-pgid, SIGTERM)`), targeting the whole process group.
- If graceful termination fails and a force path is used, escalation sends `SIGKILL` to `-PGID`.
- Liveness and cleanup logic also use PGID-aware checks, so group state rather than only one PID drives orchestration decisions.
