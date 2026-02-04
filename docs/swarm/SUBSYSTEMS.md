# Subsystems

This list is derived from ideas.md and the subsystem specs (and their QUESTIONS) in this folder.

1. Runner & Orchestration
   - Scope: run-agent binary (Go) + run-agent task/job/serve/bus/stop commands, Ralph restart loop, run linking, idle/stuck handling, agent selection/rotation (round-robin/weighted), config.hcl schema and validation, stop/kill with SIGTERM/SIGKILL, run metadata, poller/handler agents.
   - Spec: subsystem-runner-orchestration.md, subsystem-runner-orchestration-config-schema.md

2. Storage & Data Layout
   - Scope: ~/run-agent layout, run-info.yaml schema (with versioning), TASK_STATE/DONE, FACT files, timestamp/run_id format, home-folders.md, UTF-8 encoding, retention.
   - Spec: subsystem-storage-layout.md, subsystem-storage-layout-run-info-schema.md

3. Message Bus Tooling & Object Model
   - Scope: run-agent bus CLI/REST, YAML front-matter message format, message types/threading, atomic appends, streaming/polling, relationship schema, cross-scope references.
   - Spec: subsystem-message-bus-tools.md, subsystem-message-bus-object-model.md

4. Monitoring & Control UI
   - Scope: React UI (TypeScript + Ring UI + JetBrains Mono) served by run-agent serve, project/task/run tree, threaded message bus view, live output streaming via SSE, task creation UI, webpack dev workflow, REST/JSON API with integration tests.
   - Spec: subsystem-monitoring-ui.md

5. Agent Protocol & Governance
   - Scope: agent behavior rules, run folder usage, delegation depth, message bus-only comms, git safety guidance, no sandbox.
   - Spec: subsystem-agent-protocol.md

6. Environment & Invocation Contract
   - Scope: JRUN_* internal vars, prompt preamble path injection (OS-native normalization), PATH prepending, full env inheritance, SIGTERM/SIGKILL signal handling (30s grace period), error-message rules.
   - Spec: subsystem-env-contract.md

7. Agent Backend Integrations
   - Scope: per-agent adapter specs (codex, claude, gemini, perplexity, xAI), CLI/REST invocation and I/O contracts, token management, env var mapping, output conventions, keep-alive mechanisms.
   - Spec: subsystem-agent-backend-codex.md, subsystem-agent-backend-claude.md, subsystem-agent-backend-gemini.md, subsystem-agent-backend-perplexity.md, subsystem-agent-backend-xai.md

8. Frontend-Backend API Contract
   - Scope: REST/JSON + SSE API endpoints for monitoring UI, request/response schemas, TypeScript type generation, error handling, security (path validation), log/message streaming.
   - Spec: subsystem-frontend-backend-api.md

## Additional Planning Documents

- RESEARCH-FINDINGS.md: Technical research on HCL, Ring UI, SSE vs WebSocket, message bus patterns, Go process management
- TOPICS.md (formerly TIPICS.md): Cross-cutting topics and design decisions
