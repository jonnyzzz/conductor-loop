# Concurrency Architecture

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/concurrency.md`.

## Content Requirements
1. **Ralph Loop**:
    - Restart semantics, `DONE` file checks.
    - `wait_timeout` and `poll_interval`.
2. **Run Concurrency**:
    - `max_concurrent_runs` semaphore (all runs).
    - `max_concurrent_root_tasks` planner (root tasks).
3. **Message Bus**:
    - Exclusive write lock (`flock`) with retries.
    - Lockless reads.
4. **Task Dependencies**:
    - DAG execution via `depends_on`.
    - Cycle detection and blocking waits.
5. **Process Groups**:
    - Signal propagation (`SIGTERM` to `-PGID`).

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`

## Instructions
- Detail how the system handles concurrency and synchronization.
- Name the file `concurrency.md`.
