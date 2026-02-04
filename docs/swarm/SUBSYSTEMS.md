# Subsystems

This list is derived from ideas.md and the subsystem specs (and their QUESTIONS) in this folder.

1. Runner & Orchestration
   - Scope: run-agent binary + run-agent task flow, Ralph restart loop, run linking, idle/stuck handling, agent selection/rotation, stop/kill, run metadata, poller/handler agents.
   - Spec: subsystem-runner-orchestration.md

2. Storage & Data Layout
   - Scope: ~/run-agent layout, run-info.yaml, TASK_STATE/DONE, FACT files, timestamp/run_id format, home-folders.md, retention.
   - Spec: subsystem-storage-layout.md

3. Message Bus Tooling
   - Scope: run-agent bus CLI/REST, YAML front-matter message format, message types/threading, atomic appends, streaming/polling, relationship schema.
   - Spec: subsystem-message-bus-tools.md; subsystem-message-bus-object-model.md

4. Monitoring & Control UI
   - Scope: React UI served by run-agent backend, project/task/run tree, threaded message bus view, live output streaming, task creation UI.
   - Spec: subsystem-monitoring-ui.md

5. Agent Protocol & Governance
   - Scope: agent behavior rules, run folder usage, delegation depth, message bus-only comms, git safety guidance, no sandbox.
   - Spec: subsystem-agent-protocol.md

6. Environment & Invocation Contract
   - Scope: JRUN_* internal vars, prompt preamble path injection, PATH prepending, error-message rules.
   - Spec: subsystem-env-contract.md

7. Agent Backend Integrations
   - Scope: per-agent adapter specs (codex, claude, gemini, perplexity, xAI), CLI/REST invocation and I/O contracts.
   - Spec: subsystem-agent-backend-codex.md; subsystem-agent-backend-claude.md; subsystem-agent-backend-gemini.md; subsystem-agent-backend-perplexity.md; subsystem-agent-backend-xai.md
