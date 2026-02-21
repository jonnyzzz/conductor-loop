# TODO

## UX + Product Review Backlog (2026-02-21)

Source: manual design review + run-agent UX review tasks (`ux-layout`, `ux-flow`, `ux-bus`).

- [x] Merge task row with agent marker (`[codex]`) into a single line in the tree.
- [x] Fix Message Bus panel; currently appears empty / non-usable.
- [x] Add `+ New Task` entry per project directly in the tree.
- [x] Hide completed runs behind an expandable `... N completed` link.
- [x] New task flow: do not allow direct task-id editing; allow only a modifier/suffix.
- [x] Review outputs from other running review tasks and fold findings into implementation.
- [x] Fix Live Logs visibility/reliability issues.
- [x] Reduce Message Bus panel footprint; it currently takes too much space from task details.
- [x] Task details: move key attributes to the top.
- [x] Clarify restart behavior in task details/tree (whether task will restart after end/failure).
- [x] Fix task timeout semantics: timeout must be idle-output timeout (no timeout while output is flowing).
- [x] Update `.md` docs for Message Bus usage via `run-agent`.

## Execution Plan

- [x] Frontend UX batch (tree, message bus, logs, layout, task details, new task flow).
- [x] Backend timeout semantics batch (idle-timeout behavior + tests).
- [x] Documentation batch (message bus usage + operator guidance).
- [x] Cross-check against existing run-agent review outputs and close gaps.

## Reviewed Outputs

Reviewed UX/run-agent reports:
- `task-20260221-181000-ux-flow`
- `task-20260221-181000-ux-bus`
- `task-20260221-181000-ux-layout`
- `task-20260221-184300-ux-review-messagebus`
- `task-20260221-184300-ux-review-runs-logs`
- `task-20260221-184300-ux-review-layout` (failed run; partial evidence only)

Their findings are reflected in the backlog above and the implemented changes in this iteration.

## User Design Review (verbatim)

- merge task with [codex] like line, it's all one line
- fix message bus, it's empty
- + New Task -- add in the tree for each project
- hide completed runs under ... N completed link
- new task -- do not allow changing task id, only allow to write a modifier
- review outout from other running review tasks
- live logs not visible, and do not work well so far
- message bus section takes space from the task details
- task details -- attributes should do up
- task details, tree view -- not clear if a task will restart after end/failure
- task timeout computed incorrectly -- it should measure idle time if there is no new output for that time, but a task should not timeout if the output is flowing
- the message bus requires update in the .md files to explain how to use run-agent for that
