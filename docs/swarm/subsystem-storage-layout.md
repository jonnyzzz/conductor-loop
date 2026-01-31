# Storage & Data Layout Subsystem

## Overview
Defines the on-disk layout under ~/run-agent for projects, tasks, and individual agent runs. This layout is the single source of truth for monitoring and recovery.

## Goals
- Provide a predictable, human-readable structure.
- Persist task state, facts, and message bus history.
- Allow easy reconstruction of run trees from disk.

## Non-Goals
- Storing large binary artifacts.
- Managing long-term archival or retention policies beyond simple cleanup.

## Responsibilities
- Define directory layout and naming conventions.
- Define file formats for run metadata and state files.
- Specify how agents read/write state and facts.

## Directory Layout
Base root:

~/run-agent/
  <project>/
    PROJECT-MESSAGE-BUS.md
    FACT-<timestamp>-<name>.md
    task-<timestamp>-<name>/
      TASK.md
      TASK-MESSAGE-BUS.md
      TASK_STATE.md
      TASK-FACTS-<timestamp>.md
      runs/
        <runId>-<timestamp>/
          parent-run-id
          prompt.md
          output.md
          agent-type
          cwd
          agent-stdout.txt
          agent-stderr.txt

## Naming Conventions
- Project folder: short, human-readable identifier.
- Task folder: task-<date-time>-<slug>.
- Run folder: <runId>-<date-time> (runId is globally unique).
- FACT files: FACT-<date-time>-<name>.md (one fact bundle per write).
- TASK_STATE.md: short, current state only (not a log).

## File Formats
### TASK_STATE.md
- First line should include status, e.g. "status: in_progress" or "status: done".
- Short bullet list of current plan/next steps.

### parent-run-id
- Single line containing the parent run id.

### agent-type
- Single line: codex | claude | gemini.

### cwd
- Key/value lines:
  - RUN_ID=...
  - CWD=...
  - AGENT=...
  - CMD=...
  - PROMPT=...
  - STDOUT=...
  - STDERR=...
  - PID=...
  - EXIT_CODE=...

## Lifecycle
- Project created on first task creation.
- Task directory created by run-task.
- Run directory created by run-agent per execution.
- Facts are appended by creating new FACT-*.md files.
- TASK_STATE.md overwritten on each root-agent cycle.

## Retention / Cleanup
- Keep all run directories by default.
- Optional cleanup rule: delete runs older than N days when disk is low.

## Error Handling
- If any required file is missing, agents should recreate it rather than fail.
- If TASK_STATE.md is corrupted, rewrite with "status: unknown" and re-evaluate.

## Notes
- Monitoring UI reads only from this layout and message bus files.
- No external database is required for the MVP.
