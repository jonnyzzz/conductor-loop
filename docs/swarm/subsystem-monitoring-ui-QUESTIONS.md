# Monitoring & Control UI - Questions

- Q: Should the UI run as a local static app or as a server-backed app?
  Proposed default: Local static app with a small local file server.

- Q: How should the UI trigger run-task (direct shell exec, or via a backend service)?
  Proposed default: Backend service wrapper for security and portability.

- Q: Should the UI support live streaming of agent stdout/stderr?
  Proposed default: Tail-follow mode with periodic refresh (2s).
