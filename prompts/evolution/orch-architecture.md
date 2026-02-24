# Orchestrator: Create Architecture Pages (Element 2)

## FIRST: Read your operating manuals

```bash
cat /Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5.md
cat /Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5_orchestrator.md
cat /Users/jonnyzzz/Work/jonnyzzz.com-src/RLM.md
```

## Your Mission

Run **5 iterations** of a 3-agent workgroup to create comprehensive architecture documentation under `docs/architecture/`.

Target output: a complete set of architecture pages that explain the system from multiple angles:
- System overview
- Component boundaries and responsibilities
- Data flows and sequences
- Deployment and operations topology
- Decision rationale (why we built it this way)

## Setup

```bash
mkdir -p /Users/jonnyzzz/Work/conductor-loop/docs/architecture
```

## Facts & Source to Read

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md
cat /Users/jonnyzzz/Work/conductor-loop/README.md | head -60
```

## Iteration Plan

### Iteration 1 — Overview + System Context

**Agent A (claude):** Create `docs/architecture/overview.md`
- What is conductor-loop, what problem it solves
- System context: who uses it, what agents it integrates
- Key design principles (offline-first, filesystem as truth, no auth required for local)
- Technology stack (Go 1.24, React+Ring UI, SSE, YAML config)
- Commit: `git commit -m "docs(arch): add architecture overview"`

**Agent B (gemini):** Create `docs/architecture/components.md`
- All major components: run-agent CLI, conductor CLI, API server, Ralph loop, message bus, storage, frontend
- For each: responsibilities, interfaces, key files under `internal/`
- Component dependency graph (text/ASCII)
- Commit: `git commit -m "docs(arch): add component reference"`

**Agent C (codex):** Create `docs/architecture/decisions.md`
- Why filesystem over database
- Why O_APPEND+flock over a message queue
- Why CLI-wrapped agents over direct API integration
- Why YAML over HCL
- Why 14355 as default port
- Source: `docs/decisions/` folder + facts files
- Commit: `git commit -m "docs(arch): add architecture decisions"`

### Iteration 2 — Data Flows

**Agent A (codex):** Create `docs/architecture/data-flow-task-lifecycle.md`
- Complete lifecycle: submit task → Ralph loop → agent run → output → completion
- Include: DONE file semantics, output.md creation, run-info.yaml, fact propagation to project bus
- Use ASCII sequence diagram

**Agent B (gemini):** Create `docs/architecture/data-flow-message-bus.md`
- Message bus write path: O_APPEND + lock, message ID generation
- Read paths: lockless read, SSE streaming, tail/follow
- Scope hierarchy: run → task → project
- CLI discovery chain

**Agent C (claude):** Create `docs/architecture/data-flow-api.md`
- REST API request lifecycle
- SSE streaming architecture
- Auth model (optional API key)
- Path confinement / traversal protection
- Prometheus metrics scrape path

### Iteration 3 — Agent Integration + Deployment

**Agent A (claude):** Create `docs/architecture/agent-integration.md`
- How each agent type (Claude/Codex/Gemini/Perplexity/xAI) is invoked
- CLI vs REST distinction
- Env vars injected (JRUN_*, MESSAGE_BUS, RUNS_DIR)
- Diversification policy and round-robin
- Version detection and min-version enforcement

**Agent B (codex):** Create `docs/architecture/deployment.md`
- Single-binary deployment model
- Config file discovery chain
- Run directory structure (root/project/task/runs/)
- GC policy and artifact lifecycle
- Self-update mechanism

**Agent C (gemini):** Create `docs/architecture/frontend-architecture.md`
- React + JetBrains Ring UI + Vite stack
- Build output: `frontend/dist` embedded in binary
- Fallback to `web/src` baseline assets
- SSE data subscription model in UI
- Key UI features: task tree, message bus view, live logs, stop/resume

### Iteration 4 — Operations + Observability

**Agent A (gemini):** Create `docs/architecture/observability.md`
- Prometheus metrics: what's exposed at `/metrics`
- Structured logging via `internal/obslog` (key=value format)
- Form submission audit log (`_audit/form-submissions.jsonl`)
- Request correlation via `X-Request-ID`
- Health check endpoint

**Agent B (claude):** Create `docs/architecture/security.md`
- Security model: optional API key, no auth for local use
- Path confinement: how traversal protection works
- Webhook HMAC signing
- Token handling: env vars, token_file, no inline tokens
- GHA / CI security (SHA-pinned actions, scoped permissions)

**Agent C (codex):** Create `docs/architecture/concurrency.md`
- Ralph loop: restart semantics, DONE file, max restarts
- Run concurrency: `max_concurrent_runs` semaphore
- Root-task planner: FIFO queue, `max_concurrent_root_tasks`
- Message bus: lock contention model, retry with backoff
- Task dependencies: DAG, cycle detection, dependency gating

### Iteration 5 — Review, Index, and Validate

All 3 agents (codex/gemini/claude) review the full `docs/architecture/` set:
- Check consistency across pages (no contradictions)
- Verify all facts from FACTS-*.md are represented
- Create `docs/architecture/README.md` — index of all pages with one-line descriptions
- Validate all source code references are correct
- Commit final: `git commit -m "docs(arch): finalize architecture documentation"`

## Between each iteration: commit

```bash
cd /Users/jonnyzzz/Work/conductor-loop
git add docs/architecture/
git commit -m "docs(arch): iteration $ITER complete" --allow-empty
git push origin main
```

## Completion signal

```bash
echo "DONE" > /Users/jonnyzzz/run-agent/conductor-loop/TASK_ARCHITECTURE_DONE
```

Write final summary to `output.md`.
