# Topics

This list captures the current planning topics derived from ideas.md and the subsystem specs. Topics are intentionally broad and cross-cutting.

1. Message Bus Format & Lifecycle
   - Covers: message types, header/body format, compaction/archival, streaming/polling, issue/dependency metadata, start/stop events.
   - Related: ideas.md, subsystem-message-bus-tools.md, subsystem-storage-layout.md, subsystem-agent-protocol.md

2. Agent Lifecycle & Process Control
   - Covers: Ralph restart loop, idle detection, detach/stop/kill, sub-agent completion handling, locks/concurrency, backoff/circuit breaker.
   - Related: ideas.md, subsystem-runner-orchestration.md, subsystem-monitoring-ui.md, subsystem-agent-protocol.md

3. Configuration, Credentials & Multi-Backend
   - Covers: config.json schema/validation/migrations, token storage and injection, backend selection and health, UI multi-backend support.
   - Related: ideas.md, subsystem-runner-orchestration.md, subsystem-monitoring-ui.md, subsystem-storage-layout.md

4. Ownership, Safety & Boundaries
   - Covers: folder ownership, read/write scope, conflict resolution, destructive commands, sandbox/allowlist, agent safety rules.
   - Related: ideas.md, subsystem-agent-protocol.md, subsystem-runner-orchestration.md
