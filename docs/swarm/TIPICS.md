# Topics

This list captures the current planning topics derived from ideas.md, the subsystem specs, and historical Q/A. Topics are intentionally broad and cross-cutting. Each topic includes consolidated decisions and remaining open questions.

1. Message Bus Format & Threading
   - Decisions:
     - Format: append-only YAML front-matter entries separated by `---`.
     - Required headers: msg_id, ts (ISO-8601 UTC; distinct from compact filename timestamps), type, project; optional task/run_id/parents/attachment_path.
     - Types: FACT, QUESTION, ANSWER, USER, START, STOP, ERROR, INFO, WARNING, OBSERVATION, ISSUE.
     - Threading: parents[] links; append-only corrections/updates.
     - Atomic writes via run-agent bus (temp + swap); direct writes disallowed.
     - Routing: separate PROJECT-MESSAGE-BUS.md and TASK-MESSAGE-BUS.md; task messages stay in task scope, project messages are project-wide; UI aggregates at read time.
     - Size: soft 64KB body limit; larger payloads stored as attachments with attachment_path metadata.
   - Open questions (see subsystem-message-bus-tools-QUESTIONS.md):
     - Attachment storage location and cleanup policy.
   - Related: ideas.md, subsystem-message-bus-tools.md, subsystem-agent-protocol.md, subsystem-monitoring-ui.md

2. Run Lifecycle & Restart (Ralph)
   - Decisions:
     - Completion = DONE file exists AND all child runs have exited.
     - Ralph loop restarts root agent when DONE is absent; waits/monitors (and may restart root to catch up) when DONE exists but children are still running.
     - Run chain tracked via previous_run_id on each restart.
     - Idle/stuck detection uses stdout/stderr + message bus activity (default idle 5m, stuck 15m).
     - Termination: SIGTERM -> 30s wait -> SIGKILL; STOP/CRASH logged in message bus.
   - Open questions (see subsystem-runner-orchestration-QUESTIONS.md):
     - Locking/coordination for concurrent run-task invocations.
   - Related: ideas.md, subsystem-runner-orchestration.md, subsystem-monitoring-ui.md

3. Storage Layout & Metadata
   - Decisions:
     - Layout rooted at ~/run-agent/<project>/task-<timestamp>-<slug>/runs/<run_id>/.
     - run-info.yaml is canonical run metadata (lower-case keys).
     - TASK_STATE.md is free text; DONE is completion marker.
     - FACT files require YAML front matter; home-folders.md is YAML with explanations.
     - Message bus stored as single append-only file per scope; no rotation/cleanup yet.
     - No symlinks/hardlinks; no host_id segmentation.
   - Open questions (see subsystem-storage-layout-QUESTIONS.md):
     - Exact sub-second precision for canonical timestamp format (format is fixed; MMMM precision TBD).
   - Related: ideas.md, subsystem-storage-layout.md, subsystem-runner-orchestration.md

4. Agent Governance & Delegation
   - Decisions:
     - Max delegation depth = 16 (configurable).
     - RUN_FOLDER is injected into sub-agent prompts; all temp artifacts stay there.
     - Message bus tooling only; agents do not emit START/STOP (runner posts these).
     - CWD guidance: root in task folder; code-change agents in project root; research/review default to task folder.
     - No enforced sandbox or resource limits yet; git safety is guidance only.
   - Open questions (see subsystem-agent-protocol-QUESTIONS.md):
     - Exact RUN_FOLDER prompt preamble format (wording/variable name).
   - Related: ideas.md, subsystem-agent-protocol.md, subsystem-runner-orchestration.md

5. Configuration & Backend Selection
   - Decisions:
     - Config format: HCL at ~/run-agent/config.hcl (global only for now).
     - Tokens read from config and injected as env vars into agent processes.
     - Agent selection: round-robin by default; "I'm lucky" random with weights.
     - Backend failures: transient errors use exponential backoff (1s, 2s, 4s; max 3 tries); auth/quota fail fast (no refresh yet).
   - Open questions:
     - None beyond future per-project/task config support.
   - Related: ideas.md, subsystem-runner-orchestration.md, subsystem-message-bus-tools.md

6. Monitoring UI & Streaming UX
   - Decisions:
     - run-agent serve hosts UI + API; React SPA with JetBrains Mono.
     - Layout: tree ~1/3, message bus ~1/5, output pane bottom; projects are roots.
     - Threaded message bus view; output streaming via SSE/WS with polling fallback.
     - Status badges based on idle/stuck thresholds (idle default 5m; warn after idle N/2; stuck default 15m with no output).
   - Open questions (see subsystem-monitoring-ui-QUESTIONS.md):
     - None (read-only for MVP is decided).
   - Related: ideas.md, subsystem-monitoring-ui.md, subsystem-message-bus-tools.md
