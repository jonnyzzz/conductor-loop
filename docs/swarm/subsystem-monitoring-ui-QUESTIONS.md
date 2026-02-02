# Monitoring & Control UI - Questions

- Q: When a user enters a task id that already exists, should the UI attach/restart or create a new task with a suffix?
  A: Prompt user; default to restart/attach; offer "Create new with timestamp suffix".

- Q: In aggregated output views, should stdout/stderr be merged or separated?
  A: Merge chronologically with stream tags and color coding; allow quick filter toggle.

- Q: What is the default log tail size and paging behavior for large outputs?
  A: Show last 1MB (or 5k lines) per run; "Load more" fetches older chunks.

- Q: Should message bodies and prompts render as plain text or Markdown in the UI?
  Proposed default: Plain text for MVP; optional Markdown toggle later.
  A: Plain text with JetBrains mono font.

- Q: How should projects/tasks be ordered in the tree (alphabetical vs last activity)?
  A: Order by last activity (most recent first), with alphabetical tiebreakers.

- Q: Should the UI expose run controls (stop/kill/restart) for run nodes or stay read-only?
  Proposed default: Read-only for MVP; controls added later via backend.
  A: Let's keep readony for MVP.

- Q: How should the UI reflect real-time agent states (running/idle/stuck/crashed)?
  A: Status badges from backend polling run metadata and message bus events (2s refresh). Note: thresholds defined in runner config.

- Q: Should the UI enforce localhost-only access or support auth when exposed to LAN?
  Proposed default: Localhost-only for MVP; if LAN enabled, require token-based auth via config.
  A: No token auth, jsut use localhost. The WebUI should assume the REST on the same host. We serve web UI from the run-agent binary and embeedd as resources

- Q: Should the UI support search/filter across all messages and outputs, or only browse by hierarchy?
  A: Browse-only for MVP; add full-text search later.
