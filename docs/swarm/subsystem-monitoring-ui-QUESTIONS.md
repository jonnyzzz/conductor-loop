# Monitoring & Control UI - Questions

- Q: Should the UI run as a local static app or as a server-backed app?
  Proposed default: Local static app with a small local file server.
  A: We have Go binary backend that bundled the static resources of React app, we keep all standard web development pipeline

- Q: How should the UI trigger run-task (direct shell exec, or via a backend service)?
  Proposed default: Backend service wrapper for security and portability.
  A: The backend is used to look at the local system and answer all questions.

- Q: Should the UI support live streaming of agent stdout/stderr?
  Proposed default: Tail-follow mode with periodic refresh (2s).
  A: Yes of course. You can use websocket or anything else for that, live answers are really nice. Potentially, just reading the file would make it work.

- Q: How does a user post a message to the message bus from the UI?
  Proposed default: A text input in the Message Bus panel that appends a USER entry.
  A: Agree. Second approach, we add the command to the go binary to do that. That command must never be available if the go binary is started under a run-task environment (adn the env variables are set)

- Q: Should the UI allow terminating a running agent or task?
  Proposed default: Yes, provide a Stop button that kills the PID from run metadata.
  A: Yes. With a UI popup to confirm the kill. You can kill any agent, or whole sub-trees of agents.

- Q: ideas.md mentions D3/graph visualization but spec says Tree view. Which is MVP?
  Proposed default: Standard tree view for MVP; D3 graph as later enhancement.
  A: Agree. Let's start with the most straitforward react app. 

- Q: What is the polling interval for filesystem updates (tree, message bus, output)?
  Proposed default: 2s for active run output, 5s for tree/message bus.
  A: TBD.

- Q: Should the UI auto-refresh continuously or pause when the user is reading a long output?
  Proposed default: Auto-refresh with a per-pane "Pause updates" toggle.
  A: TBD.

- Q: How does the UI detect a new task created externally (outside this UI instance)?
  Proposed default: Periodic scan or file watcher on project/task directories.
  A: Yes, the directory tree is watched and UI refreshes on changes.

- Q: Should the message bus view show all entries or allow filtering by type?
  Proposed default: Show all, with optional filter dropdown.
  A: TBD.

- Q: ideas.md says "left 1/5 screen for message bus view," while spec says tree 1/3 width. Which layout wins?
  Proposed default: Tree 1/3 left, message bus 1/5 left (stacked or side-by-side), agent output bottom.
  A: TBD.

- Q: Should task-level and project-level message buses be merged or shown separately?
  Proposed default: Separate tabs/sections (Project Bus / Task Bus).
  A: TBD.

- Q: How are FACT-<date-time>-<name>.md files displayed in the UI?
  Proposed default: Expandable list under project/task node, click to view markdown.
  A: TBD.

- Q: Should the UI support editing FACT files or are they read-only?
  Proposed default: Read-only for MVP; editing via external editor.
  A: TBD.

- Q: When a task has multiple runs, which TASK_STATE.md is shown?
  Proposed default: Show the single task-level TASK_STATE.md (latest state).
  A: We have only one TASK_STATE.md per task; show and allow edits in task view.

- Q: "Start again" action: does it create a new run or resume the last run?
  Proposed default: Create a new run with the same TASK.md prompt; inherit existing state.
  A: Agree; this is a manual restart of a task run.

- Q: Is there a visual indicator when the root agent is running vs stopped?
  Proposed default: Status dot/badge on task node (green=running, red=stopped).
  A: TBD.

- Q: ideas.md mentions a max depth of 16 agents. Should the UI warn when depth exceeds 16?
  Proposed default: Show depth in run metadata; highlight >16 in red.
  A: TBD.

- Q: When creating a new project, should the UI ask for the project source code folder path?
  Proposed default: Prompt for "Project source code folder" if new project name is entered.
  A: TBD.

- Q: Should the prompt editor in "Start new Task" support markdown preview?
  Proposed default: Plain text for MVP; markdown preview as enhancement.
  A: TBD.

- Q: Is the local storage autosave for prompt text per-project or global?
  Proposed default: Per-project key to avoid mixing prompts.
  A: TBD.

- Q: ideas.md says "agent output colored per agent." How are colors assigned?
  Proposed default: Hash agent/run ID to generate stable colors.
  A: TBD.

- Q: Should stderr be visually distinguished from stdout in the agent output pane?
  Proposed default: Stderr in red, stdout default.
  A: TBD.

- Q: What API endpoints does the Go backend expose (list projects, read logs, post message, start task, kill agent)?
  Proposed default: REST endpoints for /projects, /tasks, /runs, /message-bus, /start-task, /kill-agent.
  A: TBD.

- Q: Should the backend watch filesystem changes and push updates via WebSocket, or should the UI poll REST endpoints?
  Proposed default: WebSocket for realtime updates; REST for tree metadata.
  A: TBD.

- Q: Should the Go backend bind to localhost only or allow LAN access?
  Proposed default: Localhost only for MVP; config option for LAN.
  A: TBD.

- Q: Should the UI require authentication or assume single local user?
  Proposed default: No auth for MVP (single local user).
  A: Agree; local standalone tool.

- Q: If run-task fails to start (missing env vars, permissions), how does the UI notify the user?
  Proposed default: Toast notification + error log in task view.
  A: TBD.

- Q: If a run crashes or is killed externally, how is that reflected in the UI?
  Proposed default: Mark run as FAILED/KILLED with exit code in metadata.
  A: TBD.

- Q: If MESSAGE-BUS format switches to folder-based messages, how should the UI read and display it?
  Proposed default: Support both formats; prefer folder-based if present.
  A: TBD.
