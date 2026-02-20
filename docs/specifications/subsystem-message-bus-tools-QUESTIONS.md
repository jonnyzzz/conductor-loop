# Message Bus Tooling - Questions

1. The `run-agent` CLI currently exposes only `task` and `job` commands. Should we implement the `bus` subcommands described in the spec, or adjust the spec to match a different access path?
Answer: yes, it should export message-bus sub command and related instructions.

2. The REST API currently provides `GET /api/v1/messages` and `GET /api/v1/messages/stream` only. Should we implement `POST /api/v1/messages` (or `/api/v1/bus`) for message submission and add filtering parameters (`type`, `regex`, etc.)?
Answer: Yes. The user should be able to post a message with type user or issue to the message bus of project/task levels. 

3. The Go message bus model supports `project_id`, `task_id`, `run_id`, and `parents` as a string list only. Should we extend the implementation to support structured parents, `attachments[]`, `links[]`, `issue_id`, and `meta` fields defined in the spec?
Answer: yes, but issue_id is just an alias for msg_id

4. The runner currently emits `RUN_START` and `RUN_STOP` messages with a short body only. Should we standardize on `START`/`STOP`/`CRASH` (or alias), and include structured run metadata in headers or `meta`?
Answer: yes

5. The SSE message stream currently sends `message` events with `{msg_id, content, timestamp}` and does not set an SSE `id`. Should the stream emit full message payloads and include `id` for resumable clients?
Answer: yes
