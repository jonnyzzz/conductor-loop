# Subsystems

This list is derived from ideas.md and the subsystem specs (and their QUESTIONS) in this folder.

1. Runner & Orchestration
   - Scope: run-agent binary + run-task flow, Ralph restart loop, run linking, idle/stuck handling, agent selection/rotation, stop/kill, run metadata, poller/handler agents.
   - Spec: subsystem-runner-orchestration.md

2. Storage & Data Layout
   - Scope: ~/run-agent layout, run-info.yaml, TASK_STATE/DONE, FACT files, timestamp/run_id format, home-folders.md, retention.
   - Spec: subsystem-storage-layout.md

3. Message Bus Tooling
   - Scope: run-agent bus CLI/REST, YAML front-matter message format, message types/threading, atomic appends, streaming/polling.
   - Spec: subsystem-message-bus-tools.md

4. Monitoring & Control UI
   - Scope: React UI served by run-agent backend, project/task/run tree, threaded message bus view, live output streaming, task creation UI.
   - Spec: subsystem-monitoring-ui.md

5. Agent Protocol & Governance
   - Scope: agent behavior rules, run folder usage, delegation depth, message bus-only comms, git safety guidance, no sandbox.
   - Spec: subsystem-agent-protocol.md
