# Task Lifecycle Data Flow

This page describes the task lifecycle implemented today, from task submission to project-scope completion propagation.

Implementation grounding:
- `internal/runner/task.go`
- `internal/runner/ralph.go`
- `internal/runner/job.go`
- `internal/runner/task_completion_propagation.go`
- `internal/storage/runinfo.go`
- `internal/storage/atomic.go`

## End-to-End Sequence (ASCII)

```text
submitter (CLI/API)     RunTask/task.go      taskdeps        RalphLoop        runJob/job.go        storage/*         agent backend      TASK-BUS        PROJECT-BUS
       |                      |                 |                 |                 |                   |                  |               |                 |
1.     | submit task -------->|                 |                 |                 |                   |                  |               |                 |
2.     |                      | ensure TASK.md  |                 |                 |                   |                  |               |                 |
3.     |                      | resolve deps -->| validate/ready? |                 |                   |                  |               |                 |
4.     |                      |<----------------|                 |                 |                   |                  |               |                 |
5.     |                      | wait until deps ready (or DONE already present)    |                   |                  |               |                 |
6.     |                      |-------------------------------> NewRalphLoop()      |                   |                  |               |                 |
7.     |                      |                                 check DONE?          |                   |                  |               |                 |
8.     |                      |                                 start attempt #N --->| create run dir     |                  |               |                 |
9.     |                      |                                                      | write run-info(running, exit=-1) ---->|              |                 |
10.    |                      |                                                      | post RUN_START ------------------------------------->|                 |
11.    |                      |                                                      | execute agent -------------------------------------->|                 |
12.    |                      |                                                      |<-------------------------------------- stdout/stderr |
13.    |                      |                                                      | update run-info(completed/failed) ---->|             |                 |
14.    |                      |                                                      | ensure output.md (CreateOutputMD fallback)          |                 |
15.    |                      |                                                      | post RUN_STOP/RUN_CRASH ---------------------------->|                 |
16.    |                      |                                 check DONE again; restart only if DONE absent         |               |                 |
17.    |                      | if DONE: stop restarts; wait active children if any |                   |                  |               |                 |
18.    |                      | propagateTaskCompletionToProject() ---------------------------------------------------->| append FACT ------>|
19.    |                      | task bus gets FACT/INFO/ERROR about propagation      |                   |                  |<--------------|                 |
```

## 1. Submit Task

`RunTask(projectID, taskID, TaskOptions)` is the root entry point for task lifecycle orchestration.

Key effects:
- Resolves root/task directories and ensures task directory exists.
- Resolves prompt from `TASK.md`, `--prompt`, or `--prompt-file`.
- Persists `TASK.md` when missing and prompt is provided.
- Opens task message bus (`TASK-MESSAGE-BUS.md`) for progress/fact/error messages.

## 2. Dependency Gating

Before entering Ralph loop, `RunTask` resolves dependencies and gates execution:
- Reads/normalizes `depends_on` (`TASK-CONFIG.yaml`), optionally overriding from `TaskOptions.DependsOn`.
- Validates no dependency cycle.
- Polls unresolved dependencies via `taskdeps.BlockedBy(...)`.

Dependency readiness (current implementation):
- Ready if dependency task has `DONE`, or
- Ready if dependency’s latest run is `completed` and no dependency run is currently `running`.

While blocked, task bus receives `PROGRESS` updates (`task blocked by dependencies: ...`).
When unblocked, task bus receives a `FACT` (`dependencies satisfied; starting task`).

## 3. Ralph Loop and Restart Behavior

`RalphLoop.Run()` controls retries/restarts.

Core loop behavior:
- Checks `DONE` before starting each attempt.
- If `DONE` exists, it does not start a new root run.
- Runs one root attempt via `runRoot(ctx, attempt)` (backed by `runJob`).
- Re-checks `DONE` after attempt completion.
- Stops with error if `maxRestarts` is exceeded.

Restart/resume prompt prefix behavior:
- `RestartPrefix` is: `Continue working on the following:\n\n`.
- Prefix is added when:
  - `attempt > 0` (retry), or
  - `TaskOptions.ResumeMode == true` (even on first attempt in that invocation).
- `previousRunID` is threaded across attempts and persisted in each run’s `run-info.yaml` as `previous_run_id`.

## 4. Run Creation and Agent Execution

Each attempt executes `runJob(...)`:
- Allocates run directory under `<task>/runs/<run_id>` (`createRunDir`).
- Uses preallocated run dir for the first attempt when provided (`TaskOptions.FirstRunDir`).
- Writes prompt preamble into `prompt.md`.
- Initializes `RunInfo` with `status=running`, `exit_code=-1`, path fields (`prompt_path`, `output_path`, `stdout_path`, `stderr_path`), IDs, agent type, and timing.

Execution then diverges by backend type:
- CLI agents (`claude`, `codex`, `gemini`): spawn process, stream stdout/stderr to files.
- REST agents (`perplexity`, `xai`): execute backend adapter and finalize run info.

Run events posted to task bus:
- `RUN_START`
- `RUN_STOP` on success
- `RUN_CRASH` on failure

## 5. `output.md` Guarantee and Fallback

After agent execution, runner enforces `output.md` existence:
- Parser helpers may generate `output.md` for stream-json agents.
- `agent.CreateOutputMD(runDir, "")` is always called to guarantee file presence.
- With empty fallback argument, `CreateOutputMD` copies from `agent-stdout.txt` if `output.md` is missing.
- If `output.md` already exists, it is preserved.

Effect: normal runs end with `output.md` available; if fallback copy cannot be done, the run returns an error (`ensure output.md` path).

## 6. DONE Semantics and No-Restart-After-DONE

`DONE` semantics are file-based:
- Valid marker is a file at `<taskDir>/DONE`.
- `DONE` as a directory is treated as an error.

No-restart guarantees:
- Dependency wait loop exits early if task already has `DONE`.
- Ralph loop checks `DONE` before any new attempt; if present, root runner is not called.
- Ralph loop also checks `DONE` after each attempt, so a run that creates `DONE` immediately blocks further restarts.

Child-run handling when DONE is present:
- Loop enumerates active child runs and waits for them up to configured timeout.
- If timeout occurs, warning is posted to task bus and loop returns without restarting root.

## 7. `run-info.yaml` Fields and State Transitions (High Level)

Schema is defined in `internal/storage/runinfo.go`.

Important fields:
- Identity/lineage: `run_id`, `parent_run_id`, `previous_run_id`, `project_id`, `task_id`
- Execution identity: `agent`, `agent_version`, `process_ownership`, `pid`, `pgid`, `commandline`
- Timing/outcome: `start_time`, `end_time`, `exit_code`, `status`, `error_summary`
- Artifacts: `cwd`, `prompt_path`, `output_path`, `stdout_path`, `stderr_path`

High-level state transitions:
1. Run allocated and initialized with `status=running`, `exit_code=-1`.
2. On normal completion: `status=completed`, `exit_code=0`, `end_time` set.
3. On failure/timeout/crash: `status=failed`, non-zero `exit_code` (or failure summary), `end_time` set.

Persistence semantics:
- `WriteRunInfo` writes atomically (temp file + sync + chmod + rename).
- `UpdateRunInfo` uses lock file + exclusive lock + read-modify-write + atomic rewrite to avoid lost concurrent updates.

## 8. Task Completion FACT Propagation to `PROJECT-MESSAGE-BUS.md`

After Ralph loop returns, `RunTask` calls `propagateTaskCompletionToProject(...)`.

Propagation behavior:
- Requires `DONE` file; if missing, no project FACT is posted.
- Builds run summary from `<task>/runs/*/run-info.yaml`.
- Reads task-level `FACT` signals and latest run stop/crash event from task bus.
- Computes deterministic `propagation_key` from DONE mtime + run summary + fact count.
- Uses task-local lock/state files for idempotency:
  - `TASK-COMPLETE-FACT-PROPAGATION.lock`
  - `TASK-COMPLETE-FACT-PROPAGATION.yaml`
- Also de-duplicates by scanning existing project FACT messages with matching `meta.kind=task_completion_propagation` and `meta.propagation_key`.

Posted project message:
- Type `FACT`, written to `<project>/PROJECT-MESSAGE-BUS.md`
- Includes metadata (source task/run paths, summary counts, latest run status/output path, propagation key)
- Includes links to task bus, DONE marker, latest `run-info.yaml`, latest `output.md`
- Includes parents tracing to latest run event and recent task FACT messages

If propagation fails:
- Task bus gets an `ERROR` message (`task completion fact propagation failed: ...`)
- `RunTask` still returns success (failure is surfaced but non-fatal to task completion)
