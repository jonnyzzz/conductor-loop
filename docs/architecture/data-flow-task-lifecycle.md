# Data Flow: Task Lifecycle

This document describes the control and data flow for a task from submission through completion propagation.

Grounded in:
- `docs/facts/FACTS-runner-storage.md`
- `docs/facts/FACTS-architecture.md`

## Lifecycle Phases

### 1. Submission

A task enters the system through either:
- CLI: `run-agent job ...`
- API: `POST /tasks`

At submission time, the runner resolves project/task directories under storage and reads task input from `TASK.md` (or creates it during task setup flows).

### 2. Orchestration

Task execution is coordinated by the Ralph loop:
- The loop checks `<task>/DONE` before starting a root attempt.
- If `DONE` is present, it does not start or restart the root agent.
- If `DONE` is absent, it starts an attempt (subject to restart/time budget policy).

This is the primary control gate: `DONE` controls loop continuation.

### 3. Execution

For each attempt, the runner performs run setup and agent execution:
- Creates `runs/<run_id>/`.
- Creates `run-info.yaml` with initial running state (`status=running`, `exit_code=-1`).
- Spawns the agent process/backend.
- Injects run context environment:
  - `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID`
  - `MESSAGE_BUS`, `TASK_FOLDER`, `RUN_FOLDER`, `RUNS_DIR`

During and after execution:
- `run-info.yaml` is updated with final status/outcome (`completed` or `failed`).
- Runner guarantees `output.md` exists by end of run.
- Task message bus receives run lifecycle events (`RUN_START`, `RUN_STOP`/`RUN_CRASH`).

### 4. Completion

Completion signal is file-based:
- The root agent writes `<task>/DONE` when task work is complete.
- Ralph loop observes `DONE`, stops restarting root runs, and transitions toward shutdown.
- If child runs are still active, the loop waits up to configured timeout before final exit.

Operationally, a task is complete when `DONE` exists and child execution is no longer active.

### 5. Propagation

After Ralph loop exits, completion facts are propagated upward:
- Runner synthesizes task-level facts and run outcomes.
- It writes a project-scoped `FACT` message to `PROJECT-MESSAGE-BUS.md`.
- Idempotency files/locks prevent duplicate propagation.
- Propagation failures are logged on the task bus as `ERROR`, but do not re-open task execution.

## Key Data Artifacts

### `TASK.md` (input)
- Canonical task prompt/instructions.
- Read by runner/agent as the task contract.

### `run-info.yaml` (run state; atomic updates)
- Canonical per-run metadata (`run_id`, status, timing, exit, paths, lineage).
- Persisted with atomic write semantics (temp file + `fsync` + rename).

### `output.md` (result)
- Canonical run/task result artifact.
- Ensured by runner at the end of execution.

### `DONE` (signal)
- Empty marker file in the task directory.
- Drives Ralph loop termination behavior and prevents further root restarts.

## ASCII Sequence Diagram

```text
Submitter            Runner (RunTask/Ralph)       Storage                Agent               TASK Bus               PROJECT Bus
    |                         |                      |                      |                    |                        |
1.  | run-agent job /         |                      |                      |                    |                        |
    | POST /tasks ----------> |                      |                      |                    |                        |
2.  |                         | read TASK.md ------> |                      |                    |                        |
3.  |                         | check DONE --------> | stat <task>/DONE     |                    |                        |
4.  |                         | if DONE missing: start attempt               |                    |                        |
5.  |                         | create run dir ----> | runs/<run_id>/        |                    |                        |
6.  |                         | write run-info ----> | run-info.yaml(running)|                    |                        |
7.  |                         | spawn + env --------------------------------> |                    |                        |
8.  |                         |                     (JRUN_*, TASK_FOLDER, RUN_FOLDER, MESSAGE_BUS)                    |
9.  |                         | post RUN_START -------------------------------------------------> |                        |
10. |                         | <------------------- execution/output --------|                    |                        |
11. |                         | update run-info ---> | run-info.yaml(final)  |                    |                        |
12. |                         | ensure output.md --->| output.md              |                    |                        |
13. |                         | post RUN_STOP/CRASH -------------------------------------------> |                        |
14. |                         | check DONE again --->| stat <task>/DONE       |                    |                        |
15. |                         | if DONE present: stop restart loop            |                    |                        |
16. |                         | propagate completion FACT ---------------------------------------> | append FACT            |
17. |                         | (on propagation error: log ERROR) -----------> |                    |                        |
```

## Control and Data Summary

- Control flow is loop-driven and `DONE`-gated.
- Data flow is file-first (`TASK.md`, `run-info.yaml`, `output.md`, `DONE`) with append-only buses for lifecycle events and propagated facts.
- Project-level visibility is achieved by post-completion fact propagation from task scope to project scope.
