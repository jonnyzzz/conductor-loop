# Monitoring UI - Questions

## Status
Previously open questions are resolved by current implementation.

## Resolved

1. Is `run-agent serve` implemented with host/port options?
Answer: yes.
Evidence: `run-agent serve` exists; defaults include host `0.0.0.0` and port `14355`.

2. Are project-scoped API endpoints available (beyond `/api/v1/tasks|runs|messages`)?
Answer: yes.
Evidence: `/api/projects/...` route family is implemented and used by frontend client.

3. Does task creation support `project_root`, `attach_mode`, and return `run_id`?
Answer: yes.
Evidence: `POST /api/v1/tasks` request includes `project_root` and `attach_mode`; response includes `run_id`.

4. Are message posting and metadata-rich message SSE implemented?
Answer: yes.
Evidence:
- Posting endpoints implemented at project/task and `/api/v1/messages` levels.
- SSE message events include full payload and set SSE `id` to `msg_id`.

5. Are file read endpoints available for task/run artifacts?
Answer: yes.
Evidence:
- Task file endpoint supports `TASK.md`.
- Run file endpoint supports `stdout|stderr|prompt|output.md` plus SSE file-tail stream.

6. Is the monitoring UI implemented in this repository?
Answer: yes.
Evidence: active frontend app under `frontend/src` with React components, API hooks, and Vite build.

## Notes
- Destructive operations (project/task/run delete and project GC) are intentionally blocked for UI-origin requests by backend safety checks.
