# Monitoring & Control UI - Questions

- Q: Should the UI run as a local static app or as a server-backed app?
  Proposed default: Local static app with a small local file server.
=== A: We have Go binary backend that bundled the static resources of React JS app, we keep all standard web developmnet pipeline

- Q: How should the UI trigger run-task (direct shell exec, or via a backend service)?
  Proposed default: Backend service wrapper for security and portability.
=== A: The backend is used to look at the local system and answer all questions.

- Q: Should the UI support live streaming of agent stdout/stderr?
  Proposed default: Tail-follow mode with periodic refresh (2s).
=== A: yes of course. You can use websocket or anything else for that, live answers are really nice. Potentially, just reading the file would make it work.
