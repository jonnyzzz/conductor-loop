# TODO

## UX + Product Review Backlog (2026-02-21)

Source: manual design review + run-agent UX review tasks (`ux-layout`, `ux-flow`, `ux-bus`).

- [ ] Merge task row with agent marker (`[codex]`) into a single line in the tree.
- [ ] Fix Message Bus panel; currently appears empty / non-usable.
- [ ] Add `+ New Task` entry per project directly in the tree.
- [ ] Hide completed runs behind an expandable `... N completed` link.
- [ ] New task flow: do not allow direct task-id editing; allow only a modifier/suffix.
- [ ] Review outputs from other running review tasks and fold findings into implementation.
- [ ] Fix Live Logs visibility/reliability issues.
- [ ] Reduce Message Bus panel footprint; it currently takes too much space from task details.
- [ ] Task details: move key attributes to the top.
- [ ] Clarify restart behavior in task details/tree (whether task will restart after end/failure).
- [ ] Fix task timeout semantics: timeout must be idle-output timeout (no timeout while output is flowing).
- [ ] Update `.md` docs for Message Bus usage via `run-agent`.

## Execution Plan

- [ ] Frontend UX batch (tree, message bus, logs, layout, task details, new task flow).
- [ ] Backend timeout semantics batch (idle-timeout behavior + tests).
- [ ] Documentation batch (message bus usage + operator guidance).
- [ ] Cross-check against existing run-agent review outputs and close gaps.

## Reviewed Outputs

Reviewed UX/run-agent reports:
- `task-20260221-181000-ux-flow`
- `task-20260221-181000-ux-bus`
- `task-20260221-181000-ux-layout`

Their findings are reflected in the backlog above and the implemented changes in this iteration.
