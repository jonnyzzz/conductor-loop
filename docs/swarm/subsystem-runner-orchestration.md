# Runner & Orchestration Subsystem

## Overview
This subsystem owns how agents are started, restarted, and coordinated. It defines the CLI tools and runtime contract for spawning agents (run-agent.sh) and for running a task end-to-end (run-task/start-task flow).

## Goals
- Start agents with a consistent run layout and metadata.
- Track parent-child relationships between runs.
- Keep the root orchestrator running until the task is complete ("ralph" restart loop).
- Provide a stable CLI interface for starting tasks and agents.
- Rotate or auto-select agent types to avoid stalling on one model.

## Non-Goals
- Implementing the message bus itself (handled by Message Bus subsystem).
- Implementing the monitoring UI (handled by Monitoring UI subsystem).
- Writing the actual prompt content for every task (handled by task authoring/UI).

## Responsibilities
- Validate required environment variables for run tracking.
- Create run directories and log metadata for each agent run.
- Launch agents with a provided prompt file.
- Restart the root agent on exit until the task is done.
- Spawn and manage message-bus polling agents.

## Components
### run-agent.sh
- Single responsibility: start one agent run and block until it finishes.
- Required environment:
  - JRUN_TASK_ID: task identifier.
  - JRUN_PROJECT_ID: project identifier.
  - JRUN_ID: unique run identifier for this invocation.
  - JRUN_PARENT_ID: optional parent run id (empty for root).
- Records run metadata and links parent-child relations.
- Supports explicit agent selection (codex/claude/gemini) or a "lucky" mode.

### run-task (start-task) flow
- Reads the task prompt from TASK.md or user input.
- Resolves or generates:
  - Project id
  - Task id
- Creates the task folder and initial files.
- Starts the root agent with a canonical root prompt.
- Implements the restart loop (restarts root agent if it exits before task completion).
- Starts background pollers for project and task message buses.

## Interfaces / CLI
### run-agent.sh
- Usage: run-agent.sh [agent] [cwd] [prompt_file]
- Writes run metadata files (see Storage subsystem).
- Exit code: pass-through from agent process.

### run-task
- Usage: run-task --project <id> --task <id> [--prompt <file>]
- If missing, prompts to generate project/task id.
- Creates TASK.md if not present.

## Root Orchestrator Prompt Contract
The root agent prompt must include these requirements:
- Read TASK_STATE.md and TASK-MESSAGE-BUS.md on start.
- Only communicate via message bus.
- Create facts in FACT-*.md files.
- Update TASK_STATE.md with short, current state.
- Delegate sub-tasks by starting sub agents (run-agent.sh).
- Promote stable facts to project-level FACT files.
- Restart-safe behavior (idempotent read/decide).

## Workflows
### Start New Task (happy path)
1. User creates task via UI or CLI.
2. run-task creates project/task folder and TASK.md.
3. run-task starts root agent (run-agent.sh).
4. Root agent processes task and spawns sub agents.
5. run-task restarts root agent until done condition is satisfied.

### Root Agent Restart ("ralph")
1. Root agent exits (success or failure).
2. run-task checks TASK_STATE.md for completion flag.
3. If incomplete, re-run root agent with same prompt and context.

## Data / State Touched
- TASK.md (input prompt)
- TASK_STATE.md (short state)
- TASK-MESSAGE-BUS.md and PROJECT-MESSAGE-BUS.md
- FACT-*.md files (task and project)
- runs/<runId> metadata (prompt, output, parent-run-id, agent-type, cwd)

## Error Handling
- If required env vars are missing, run-agent exits with error.
- If prompt file is missing, run-agent exits with error.
- If root agent fails repeatedly, run-task should backoff and log error.
- If message bus pollers crash, restart them in the background.

## Observability
- run-agent.sh writes:
  - agent stdout/stderr
  - prompt.md
  - cwd.txt (run metadata)
  - parent-run-id
- run-task writes a run log for start/stop events.

## Security / Permissions
- Only the run-task user can write to the project/task directory.
- Avoid passing secrets in prompts; store them in SECRETS.md or env vars.
- Agents should not touch unrelated files outside the task scope.
