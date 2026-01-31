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
