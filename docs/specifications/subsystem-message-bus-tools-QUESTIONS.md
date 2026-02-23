# Message Bus Tooling - Questions

## Status
All previously tracked questions are resolved in current code.

## Resolved

1. Should `run-agent` expose message bus CLI commands?
Answer: yes, implemented.
Evidence: `run-agent bus` provides `post`, `read`, and `discover`.

2. Should REST support posting messages (not only reading/streaming)?
Answer: yes, implemented.
Evidence:
- `POST /api/v1/messages`
- `POST /api/projects/{project_id}/messages`
- `POST /api/projects/{project_id}/tasks/{task_id}/messages`

3. Should message model support structured parents?
Answer: yes, implemented.
Evidence: `parents` accepts string or object form (`msg_id`, `kind`, `meta`).

4. Should runner lifecycle include crash events?
Answer: yes, implemented.
Evidence: `RUN_START`, `RUN_STOP`, `RUN_CRASH` constants and runner emission logic.

5. Should SSE include resumable IDs and full metadata payload?
Answer: yes, implemented.
Evidence: message SSE sets `id: <msg_id>` and payload includes `msg_id`, `timestamp`, `type`, `project_id`, `task_id`, `run_id`, `issue_id`, `parents`, `meta`, `body`.

## Clarifications
- CLI default post type is `INFO`; API default post type is `USER` when omitted.
- There is no `run-agent bus watch`; use `run-agent bus read --follow`.
- `attachments` fields are not part of the current core message struct.
