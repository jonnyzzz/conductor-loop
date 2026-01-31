# Subsystems

This list is derived from ideas.md and groups related capabilities into implementable subsystems.

1. Runner & Orchestration
   - Scope: run-agent.sh and run-task/start-task orchestration (agent spawning, restarts, env vars, logging).
   - Spec: subsystem-runner-orchestration.md

2. Storage & Data Layout
   - Scope: on-disk layout under ~/run-agent (projects, tasks, runs, state, facts, message bus files).
   - Spec: subsystem-storage-layout.md

3. Message Bus Tooling
   - Scope: message bus integration and helper scripts (post-message.sh, poll-message.sh, CLI/REST expectations).
   - Spec: subsystem-message-bus-tools.md

4. Monitoring & Control UI
   - Scope: React web UI for task tree, message bus, agent outputs, and task creation flow.
   - Spec: subsystem-monitoring-ui.md

5. Agent Protocol & Governance
   - Scope: agent communication rules, delegation behavior, state/fact updates, git-safety requirements.
   - Spec: subsystem-agent-protocol.md
