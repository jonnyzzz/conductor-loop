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
  A: 2 seconds is fine, but consider filesystem monitoring as the next step. Since we provide run-agent binary, we can use it to notify more effectively.

- Q: Should the UI auto-refresh continuously or pause when the user is reading a long output?
  Proposed default: Auto-refresh with a per-pane "Pause updates" toggle.
  A: Auto refresh continuously. No need to pause. I want to see the progress of my agents and tasks and messages and issues.

- Q: How does the UI detect a new task created externally (outside this UI instance)?
  Proposed default: Periodic scan or file watcher on project/task directories.
  A: Yes, the directory tree is watched and UI refreshes on changes.

- Q: Should the message bus view show all entries or allow filtering by type?
  Proposed default: Show all, with optional filter dropdown.
  A: Show the last entries. Header-only. When clicking show the entry details. There should be UI to comment or answer the message, once done you create an answer message with relevant ID, so we pick the answers/discussions thread for each message together. Agents should be able to read the whole thread if the answer is added to each. Basically, all messages in the bus can have threads. 

- Q: ideas.md says "left 1/5 screen for message bus view," while spec says tree 1/3 width. Which layout wins?
  Proposed default: Tree 1/3 left, message bus 1/5 left (stacked or side-by-side), agent output bottom.
  A: Tree 1/3 left, message bus 1/5 left (stacked or side-by-side), agent output bottom. Conduct UX review to find the better layout.

- Q: Should task-level and project-level message buses be merged or shown separately?
  Proposed default: Separate tabs/sections (Project Bus / Task Bus).
  A: Message busses are managed by the `run-agent` binary. A task agent have read-only access to the project bus, all updates to the project bus are submitted in task-bus and processed by the root agents.

- Q: How are FACT-<date-time>-<name>.md files displayed in the UI?
  Proposed default: Expandable list under project/task node, click to view markdown.
  A: Expandable list under project/task node, click to view markdown.

- Q: Should the UI support editing FACT files or are they read-only?
  Proposed default: Read-only for MVP; editing via external editor.
  A: Readonly for now. Editing is only possible by posting the message box message to a selected task queue, which refers to the fact.

- Q: When a task has multiple runs, which TASK_STATE.md is shown?
  Proposed default: Show the single task-level TASK_STATE.md (latest state).
  A: We have only one TASK_STATE.md per task; show and allow edits in task view. That is normal to start a new run for a selected task, we will definitely do that, and that auto-restart should be a vital part of the run-agent featuers.

- Q: "Start again" action: does it create a new run or resume the last run?
  Proposed default: Create a new run with the same TASK.md prompt; inherit existing state.
  A: We start the task once again with the same task prompt (previous agent could change it, it's ok), and with the THE_PROMPT*, at the beginning of the prompt we ask the root agent to resume working on the task.

- Q: Is there a visual indicator when the root agent is running vs stopped?
  Proposed default: Status dot/badge on task node (green=running, red=stopped).
  A: Yes, each agent in the tree should have semaphore-like icon to indicate its status. We have PID for each agent, so we should check with it. Also, there must be a detection of stuck agents, such agents that do not write anything for several minutes (the configurable constant in the settings)

- Q: ideas.md mentions a max depth of 16 agents. Should the UI warn when depth exceeds 16?
  Proposed default: Show depth in run metadata; highlight >16 in red.
  A: No, no need to warn, just make a settings in the settiongs of the tool. The run-agent command should fail starting new agent if the depth exceeds the limit.

- Q: When creating a new project, should the UI ask for the project source code folder path?
  Proposed default: Prompt for "Project source code folder" if new project name is entered.
  A: Yes, ask, and provide the list of presets, which are stored in the settings file. Resolve ~ paths and expand env vars.

- Q: Should the prompt editor in "Start new Task" support markdown preview?
  Proposed default: Plain text for MVP; markdown preview as enhancement.
  A: No, just let me type Markdown with JetBrains Mono font. Use Markdown-basic editor in UI to make it nice.

- Q: Is the local storage autosave for prompt text per-project or global?
  Proposed default: Per-project key to avoid mixing prompts.
  A: Yes, per each editor. Allows the actuion to look up the previous texts for all editors. There must be a common editor component with that feature to help reuse prompts

- Q: ideas.md says "agent output colored per agent." How are colors assigned?
  Proposed default: Hash agent/run ID to generate stable colors.
  A: No need to align, we select the tree node, so the log shows everything from all agents under that node. Each line of the output is colored depending on the run-id. So you just need to assign the colors per each run-id. It will make it easier to read and follow the output. 

- Q: Should stderr be visually distinguished from stdout in the agent output pane?
  Proposed default: Stderr in red, stdout default.
  A: You can actually maintain two separate streams per agent, that is acceptable. This way each error stream should have it's unique red-ish color and bold. You need to carefully read process output to ensure it reads both streams at the same time, includes timestamps.

- Q: What API endpoints does the Go backend expose (list projects, read logs, post message, start task, kill agent)?
  Proposed default: REST endpoints for /projects, /tasks, /runs, /message-bus, /start-task, /kill-agent.
  A: Up to you. The UI/backend should be decoupled, and it should be possible to restart it while the other part is running. The REST API should have decent specifiaction and documentation.

- Q: Should the backend watch filesystem changes and push updates via WebSocket, or should the UI poll REST endpoints?
  Proposed default: WebSocket for realtime updates; REST for tree metadata.
  A: Use Websocket for realtime updates. REST and slow-running streams are absolutely fine to fetch logs. Use SSE like approach, and send empty messages to keep the connection alive.

- Q: Should the Go backend bind to localhost only or allow LAN access?
  Proposed default: Localhost only for MVP; config option for LAN.
  A: Localhost. Add environment variable to configure it. Our docker integration will need tweaks.

- Q: Should the UI require authentication or assume single local user?
  Proposed default: No auth for MVP (single local user).
  A: Agree; local standalone tool.

- Q: If run-task fails to start (missing env vars, permissions), how does the UI notify the user?
  Proposed default: Toast notification + error log in task view.
  A: No, it does not. It just fails and get handled by the caller (run-agent).

- Q: If a run crashes or is killed externally, how is that reflected in the UI?
  Proposed default: Mark run as FAILED/KILLED with exit code in metadata.
  A: Yes, log a message related to the processes and statuses and mark the fact it was killed externally or crashed. Agents will catch up and recover. root can be restarted too.

- Q: If MESSAGE-BUS format switches to folder-based messages, how should the UI read and display it?
  Proposed default: Support both formats; prefer folder-based if present.
  A: Message bus is fully supported by our go binary, there should be code-reuse of that logic.
