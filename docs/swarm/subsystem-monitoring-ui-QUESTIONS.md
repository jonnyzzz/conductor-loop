# Monitoring & Control UI - Questions

- Q: Should the UI/backend read the projects root from ~/run-agent/config.json or assume fixed ~/run-agent?
  Proposed default: Read config when present, fall back to ~/run-agent.
  A: We start backend, it reads the config and follows the folders witten there. same applies to all runs of the run-agent tool.

- Q: For "Start new Task", should the backend invoke run-agent task (unified CLI) or legacy run-task/run-task.sh?
  Proposed default: Prefer run-agent task when available; fall back to run-task for compatibility.
  A: There are no *sh files, only the run-agent binary.

- Q: When task start fails (env vars/permissions), how should the UI surface the error?
  Proposed default: Error banner/toast with a link to logs; keep task in failed state.
  A: No, tasks should function without backend process running, so there no need to log the error, just output it and move on.

- Q: Should task-level FACT files be visible under the task node/view?
  Proposed default: Yes, show task facts under the task node in the tree.
  A: Yes. Double check if fact files are in the message bus or not.

- Q: When selecting a project/task node, should the output pane aggregate logs from all descendant runs or require a single-run selection?
  Proposed default: Aggregate logs from all descendant runs for project/task selection; show single-run logs only for run nodes.
  A: Aggregate all information for the selected sub-tree.

- Q: Should TASK_STATE.md be editable directly in the UI or read-only?
  Proposed default: Read-only in UI; updates via MESSAGE-BUS or external editor.
  A: Nope, UI is not letting you change the files. Updates go via message bus and can ask an agent to update the task file or task-state.

- Q: Should the message bus view be flat chronological or threaded (parent/reply)?
  Proposed default: Flat chronological with thread indicators based on reply_to.
  A: Threaded. Looks like yet another tree-like view for the UI. 

- Q: How should the UI detect and present "stuck" agents (no output for N minutes)?
  Proposed default: Visual indicator + optional notification; no auto-kill.
  A: Trafic-lightstyle indicator. All agents that are silent for N/2 minutes should reported and killed after N minutes. It is configured in the settings, and enforced by each local run-agent instance. 

- Q: When combining logs for multiple runs, should output be interleaved by timestamp or grouped by agent?
  A: Interleave by timestamp with per-run color coding.

- Q: Should the UI support multiple backend hosts (ideas.md mentions multi-backend/hosts)?
  Proposed default: Config defines backends[] with a selector in the UI.
  A: Potentially, if it's cheap to implement. Otherwise, backlog it.

- Q: Should the UI expose backend/token configuration status (configured/missing/expired)?
  Proposed default: 
  A: Read-only status panel; edits via config/CLI. Never show tokens in UI, check if present in config. Check/update available agent infos.

- Q: Should the backend expose an OpenAPI/Swagger spec for its REST endpoints?
  Proposed default: Yes; provide OpenAPI 3.0 and a /docs endpoint.
  A: That is non-goal for now. Depends on the technical feasibility and selected tech stack.
