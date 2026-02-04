# Topics

This list captures the current planning topics derived from ideas.md, the subsystem specs, and historical Q/A. Topics are intentionally broad and cross-cutting. Each topic includes consolidated decisions and remaining open questions.

1. Message Bus Format & Threading
   - Decisions:
     - Format: append-only YAML front-matter entries separated by `---` (no legacy single-line compatibility).
     - Required headers: msg_id, ts (ISO-8601 UTC; distinct from compact filename timestamps), type, project; optional task/run_id/parents/attachment_path.
     - Types: FACT, QUESTION, ANSWER, USER, START, STOP, ERROR, INFO, WARNING, OBSERVATION, ISSUE.
     - Threading: parents[] links; append-only corrections/updates.
     - Atomic writes via run-agent bus (temp + swap); direct writes disallowed.
     - Routing: separate PROJECT-MESSAGE-BUS.md and TASK-MESSAGE-BUS.md; task messages stay in task scope, project messages are project-wide; UI aggregates at read time.
     - Size: soft 64KB body limit; larger payloads stored as attachments in the task folder with timestamp + short description naming and attachment_path metadata.
     - Reader behavior: sequential scan with optional filter/regex; no cursor file.
   - Open questions (see subsystem-message-bus-tools-QUESTIONS.md):
     - Message dependency semantics beyond parents[] (e.g., issue relationships / "beads").
   - Related: ideas.md, subsystem-message-bus-tools.md, subsystem-agent-protocol.md, subsystem-monitoring-ui.md

2. Run Lifecycle & Restart (Ralph)
   - Decisions:
     - Completion = DONE file exists AND all child runs have exited.
     - Ralph loop restarts root agent when DONE is absent; waits/monitors (and may restart root to catch up) when DONE exists but children are still running.
     - Run chain tracked via previous_run_id on each restart.
     - No lockfile coordination; uniqueness is ensured by timestamp+PID and retry on collision.
     - Idle/stuck detection uses stdout/stderr + message bus activity (idle default 5m, stuck 15m; idle means all children idle; waiting if last entry is QUESTION).
     - Termination: SIGTERM -> 30s wait -> SIGKILL; STOP/CRASH logged in message bus.
     - Message-bus handling is orchestrated by the root agent in MVP (no dedicated poller/heartbeat process).
   - Open questions (see subsystem-runner-orchestration-QUESTIONS.md):
     - None at this time.
   - Related: ideas.md, subsystem-runner-orchestration.md, subsystem-monitoring-ui.md

3. Storage Layout & Metadata
   - Decisions:
     - Layout rooted at ~/run-agent/<project>/task-<timestamp>-<slug>/runs/<run_id>/ with config.hcl at ~/run-agent/.
     - run-info.yaml is canonical run metadata (lower-case keys).
     - TASK_STATE.md is free text; DONE is completion marker; TASK_STATE updates use temp+rename (root only).
     - Timestamp precision is milliseconds (MMMM) in `YYYYMMDD-HHMMSSMMMM-PID`.
     - output.md is the final agent response; stdout/stderr are raw streams; UI defaults to output.md with raw log toggle.
     - FACT files require YAML front matter; home-folders.md is YAML with explanations.
     - Message bus stored as single append-only file per scope; no rotation/cleanup yet.
     - No symlinks/hardlinks; no host_id segmentation; no size limits enforced.
   - Open questions (see subsystem-storage-layout-QUESTIONS.md):
     - None at this time.
   - Related: ideas.md, subsystem-storage-layout.md, subsystem-runner-orchestration.md

4. Agent Governance & Delegation
   - Decisions:
     - Max delegation depth = 16 (configurable).
     - RUN_FOLDER is injected into sub-agent prompts (explicit preamble); no OWNERSHIP.md file; ownership is implicit.
     - Parents may read child output/TASK_STATE; policy does not restrict it.
     - Message bus tooling only; agents do not emit START/STOP (runner posts these).
     - CWD guidance: root in task folder; code-change agents in project root; research/review default to task folder.
     - No enforced sandbox, sensitive-path guardrails, or resource limits; scripts are allowed; cross-project access is not blocked.
     - No protocol version negotiation; assume backward compatibility.
     - Git safety is guidance only (touch only selected files).
   - Open questions (see subsystem-agent-protocol-QUESTIONS.md):
     - None at this time.
   - Related: ideas.md, subsystem-agent-protocol.md, subsystem-runner-orchestration.md

5. Configuration & Backend Selection
   - Decisions:
     - Config format: HCL at ~/run-agent/config.hcl (global only for now).
     - Tokens read from config and injected as env vars into agent processes.
     - Agent selection: round-robin by default; "I'm lucky" random with weights; selection may consult message bus history.
     - Backend failures: transient errors use exponential backoff (1s, 2s, 4s; max 3 tries); auth/quota fail fast; no proactive credential validation.
     - Supported backends/providers list is defined in config.
   - Open questions (see subsystem-runner-orchestration-QUESTIONS.md):
     - How Perplexity and xAI backends should be integrated (CLI wrapper vs API client) and exposed via config.
   - Related: ideas.md, subsystem-runner-orchestration.md, subsystem-message-bus-tools.md

6. Monitoring UI & Streaming UX
   - Decisions:
     - run-agent serve hosts UI + API; React SPA with JetBrains Mono.
     - Layout: tree ~1/3, message bus ~1/5, output pane bottom; projects are roots; order by last activity.
     - Threaded message bus view; plain text rendering in MVP.
     - Output: merged stdout/stderr with stream tags and filter toggle; output.md is the default view.
     - Streaming via SSE (WS optional) with 2s polling fallback; default tail size 1MB or 5k lines.
     - Task creation UI: existing task id prompts attach/restart vs new with timestamp suffix; prompt editor autosaves to local storage; uses run-agent task.
     - Status badges based on idle/stuck thresholds (2s refresh from run metadata + bus events).
     - Read-only for MVP; localhost only; no global search in MVP.
   - Open questions (see subsystem-monitoring-ui-QUESTIONS.md):
     - When should multi-host backend selection be introduced (MVP or later)?
   - Related: ideas.md, subsystem-monitoring-ui.md, subsystem-message-bus-tools.md

7. Environment Variable & Invocation Contract
   - Decisions:
     - Runner injects JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID (internal) and RUN_FOLDER for sub-agents; RUN_FOLDER is read-only.
     - Error messages must not instruct agents to set env vars; agents should not manipulate JRUN_*.
   - Open questions (see subsystem-env-contract-QUESTIONS.md):
     - Exact full list of JRUN_* variables (including parent_run_id name and any agent/backend metadata).
     - Read-only vs writable classification for all injected variables.
   - Related: ideas.md, subsystem-agent-protocol.md, subsystem-runner-orchestration.md
