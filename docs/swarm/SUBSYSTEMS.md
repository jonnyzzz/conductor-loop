# Subsystems

This list is derived from ideas.md and the subsystem specs (and their QUESTIONS) in this folder.

1. Runner & Orchestration
   - Scope: run-agent/run-task lifecycle, agent selection/rotation, restart/backoff/idle handling, process control (detach/stop), concurrency/locks, config-driven behavior.
   - Spec: subsystem-runner-orchestration.md

2. Storage & Data Layout
   - Scope: on-disk layout under ~/run-agent (projects, tasks, runs, state, facts), run metadata (run-info), message bus storage/rotation/offsets, archival/indexing.
   - Spec: subsystem-storage-layout.md

3. Message Bus Tooling
   - Scope: run-agent bus CLI/REST, message format/types, compaction/archival, streaming/polling, issue/dependency metadata.
   - Spec: subsystem-message-bus-tools.md

4. Monitoring & Control UI
   - Scope: React UI + Go backend, project/task/run tree, message bus view, output streaming, task creation, multi-backend support, start/stop controls.
   - Spec: subsystem-monitoring-ui.md

5. Agent Protocol & Governance
   - Scope: agent behavior rules, delegation depth/ownership, state/fact updates, git-safety requirements, folder boundaries.
   - Spec: subsystem-agent-protocol.md
