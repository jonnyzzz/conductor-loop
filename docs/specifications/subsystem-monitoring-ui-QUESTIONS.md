# Monitoring & Control UI - Questions

## Open Questions (Docs vs Code, 2026-02-05)

### Q1: Is `run-agent serve` planned as a CLI command?
**Issue**: The UI and API specs assume `run-agent serve` exists, but `cmd/run-agent/main.go` only defines `task` and `job` commands.

**Question**: Should we add a `serve` command to `run-agent` (and wire `internal/api` there), or should the specs describe a different binary/entry point?

**Answer**: yes, there should be additional parameters to pick the host (defaults to 127.0.0.1) and port (defaults to 14355)

---

### Q2: Project-scoped API endpoints vs current task/run endpoints
**Issue**: Specs define project-first endpoints like `/api/v1/projects/:project_id/...`, but `internal/api/routes.go` exposes `/api/v1/tasks`, `/api/v1/runs`, and `/api/v1/messages` with query params.

**Question**: Should the backend be extended to add project-scoped endpoints, or should the specs be revised to match the current `/api/v1/tasks|runs|messages` shape?

**Answer**: yes

---

### Q3: Task creation payload and response shape
**Issue**: Specs include `project_root`, `attach_mode`, and `run_id` in the task creation flow, but `internal/api/handlers.go` expects `{project_id, task_id, agent_type, prompt, config}` and returns `{project_id, task_id, status}` only.

**Question**: Should the API add `project_root` and `attach_mode` handling and return `run_id`, or should the specs drop those fields and stick to the current payload/response?

**Answer**: yes

---

### Q4: Message bus POST endpoints and SSE payload
**Issue**: Specs require POST endpoints for USER/ANSWER messages and expect SSE payloads to include full message metadata, but the code only supports GET + SSE and the SSE payload contains `{msg_id, content, timestamp}`.

**Question**: Should the backend implement message posting and expand SSE payloads to include `type`, `parents`, `project_id`, `task_id`, and `attachment_path`, or should the UI rely on read-only message bus access for MVP?

**Answer**: Message posting should be independent from the messages listening, just implement independent logic to post messages to message bus (there is the same for CLI) and yet another logic to monitor that from the web 

---

### Q5: File read endpoints for TASK_STATE and run artifacts
**Issue**: Specs define file read endpoints for task and run artifacts, but `internal/api` does not currently expose file read routes.

**Question**: Should the backend add file read endpoints (with a safe allowlist), or should the UI avoid direct file reads and rely only on SSE/log streams for MVP?

**Answer**: Yes, use the streams for that instead, UI should not know about the files on the disk

---

### Q6: Web UI implementation status
**Issue**: `web/src` is empty, yet the monitoring UI spec describes a concrete Ring UI layout and behavior.

**Question**: Should we proceed with implementing the UI under `web/` next, or keep the spec as a target while API work lands first?

**Answer**: yes
