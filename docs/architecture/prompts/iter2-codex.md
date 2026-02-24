# Data Flow: Task Lifecycle

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/data-flow-task-lifecycle.md`.

## Content Requirements
1. **Lifecycle Phases**:
    - **Submission**: CLI (`run-agent job`) or API (`POST /tasks`).
    - **Orchestration**: Ralph loop entry, checks `DONE`.
    - **Execution**: Agent process spawn, env injection, `run-info.yaml` creation.
    - **Completion**: Agent writes `DONE`, runner detects it, loop exits.
    - **Propagation**: Fact propagation to Project Message Bus.
2. **Key Data Artifacts**:
    - `run-info.yaml` (atomic updates)
    - `TASK.md` (input)
    - `output.md` (result)
    - `DONE` (signal)
3. **Diagram**: Include a text-based ASCII sequence diagram of the flow.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

## Instructions
- Focus on the *flow* of data and control.
- Name the file `data-flow-task-lifecycle.md`.
