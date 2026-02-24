# Orchestrator: Implement All 30 Task Prompts

## FIRST: Read your operating manuals

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PROMPT_v5.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PROMPT_v5_orchestrator.md
cat /Users/jonnyzzz/Work/jonnyzzz.com-src/RLM.md
```

## Context

All 30 task prompt files are in `prompts/tasks/`. The goal is to:
1. Update FACTS files to capture requirements from each task
2. Implement every task (code, tests, docs)
3. Verify tests pass after each batch
4. Commit all work

Working directory: `/Users/jonnyzzz/Work/conductor-loop`

## Phase 1 — FACTS Update (1 agent, sequential)

**Agent (gemini)**: Read ALL 30 task prompt files and update FACTS:
```bash
ls prompts/tasks/
cat prompts/tasks/*.md
cat docs/facts/FACTS-*.md
```
- Add any new facts from task prompts that are not yet in FACTS files
- Focus on: architectural decisions, acceptance criteria, key implementation facts
- Update `docs/facts/FACTS-issues-decisions.md`, `docs/facts/FACTS-architecture.md`, `docs/facts/FACTS-suggested-tasks.md`
- Mark already-completed tasks: `fix-version-injection`, `fix-update-smoke-test`, `fix-conductor-binary-port` (RESOLVED)
- Commit: `git commit -m "docs(facts): capture all task requirements into facts files"`

Wait for Phase 1 to complete before starting Phase 2.

## Phase 2 — P0 Reliability (3 parallel agents)

**Agent A (codex) — SSE + RunInfo hardening**:
- `prompts/tasks/fix-sse-cpu-hotspot.md`
- `prompts/tasks/runinfo-missing-noise-hardening.md`
- After each: `go test ./internal/api/... ./internal/storage/... -count=1`
- Commit after each task

**Agent B (claude) — Monitor reliability**:
- `prompts/tasks/fix-monitor-process-cap.md`
- `prompts/tasks/monitor-stop-respawn-race.md`
- After each: `go test ./cmd/run-agent/... ./internal/runner/... -count=1`
- Commit after each task

**Agent C (gemini) — Run status + webserver**:
- `prompts/tasks/run-status-finish-criteria.md`
- `prompts/tasks/webserver-uptime-autorecover.md`
- After each: `go test ./internal/storage/... ./internal/api/... -count=1`
- Commit after each task

Wait for Phase 2 to complete. Run `go test $(go list ./... | grep -v '/test/docker')` to verify.

## Phase 3 — Independent P0/P1 (3 parallel agents)

**Agent A (codex) — Blocked dependency + env sanitization**:
- `prompts/tasks/blocked-dependency-deadlock-recovery.md`
- `prompts/tasks/env-sanitization.md`
- Tests after each task
- Commit after each task

**Agent B (gemini) — Config cleanup + security**:
- `prompts/tasks/hcl-config-deprecation.md`
- `prompts/tasks/token-leak-audit.md`
- `prompts/tasks/run-artifacts-git-hygiene.md`
- `prompts/tasks/gemini-stream-json-fallback.md`
- Tests after each task
- Commit after each task

**Agent C (claude) — Bootstrap + Release gate**:
- `prompts/tasks/unified-bootstrap.md`
- `prompts/tasks/release-readiness-gate.md`
- `prompts/tasks/messagebus-empty-regression.md`
- Tests after each task
- Commit after each task

Wait for Phase 3 to complete. Run full test suite.

## Phase 4 — CLI Features (3 parallel agents, independent)

**Agent A (codex) — iterate + output synthesize**:
- `prompts/tasks/implement-iterate.md`
- `prompts/tasks/implement-output-synthesize.md`
- Tests after each
- Commit after each

**Agent B (claude) — review quorum + global facts**:
- `prompts/tasks/implement-review-quorum.md`
- `prompts/tasks/global-fact-storage.md`
- Tests after each
- Commit after each

**Agent C (gemini) — Binary merge + Windows stubs**:
- `prompts/tasks/merge-conductor-run-agent.md`
- `prompts/tasks/windows-file-locking.md`
- `prompts/tasks/windows-process-groups.md`
- Tests after each
- Commit after each

Wait for Phase 4 to complete.

## Phase 5 — UI + Final (3 parallel agents)

**Agent A (codex) — UI performance**:
- `prompts/tasks/ui-latency-fix.md`
- `prompts/tasks/ui-refresh-churn-cpu-budget.md`
- Vitest tests: `cd frontend && npm test`
- Commit after each

**Agent B (claude) — UI reliability**:
- `prompts/tasks/ui-task-tree-guardrails.md`
- `prompts/tasks/ui-new-task-submit-durability.md`
- `prompts/tasks/live-logs-regression-guardrails.md`
- Vitest tests after each
- Commit after each

**Agent C (gemini) — Final verification**:
- `prompts/tasks/run-status-finish-criteria.md` (verify UI integration complete)
- Run: `go test $(go list ./... | grep -v '/test/docker')`
- Run: `cd frontend && npm test`
- Run: `bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh`
- Run: `bash scripts/release-gate.sh` (if it exists)
- Fix any remaining failures
- Update `docs/facts/FACTS-suggested-tasks.md` to mark all tasks as COMPLETED
- Final commit

## Execution model

Use `run-agent job` for each agent. Example:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
TS=$(date +%Y%m%d-%H%M%S)

# Phase 2 example (3 parallel agents)
./bin/run-agent job --config config.local.yaml --agent codex --project conductor-loop \
  --task "task-${TS}-p2-sse-runinfo" \
  --root /Users/jonnyzzz/run-agent --cwd /Users/jonnyzzz/Work/conductor-loop \
  --prompt "Read and implement prompts/tasks/fix-sse-cpu-hotspot.md then prompts/tasks/runinfo-missing-noise-hardening.md. Run tests after each. Commit each separately." \
  --timeout 30m &

./bin/run-agent job --config config.local.yaml --agent claude --project conductor-loop \
  --task "task-${TS}-p2-monitor" \
  --root /Users/jonnyzzz/run-agent --cwd /Users/jonnyzzz/Work/conductor-loop \
  --prompt-file prompts/tasks/fix-monitor-process-cap.md \
  --timeout 30m &

wait
```

## Between phases: always verify

```bash
cd /Users/jonnyzzz/Work/conductor-loop
go build ./...
go test $(go list ./... | grep -v '/test/docker') -count=1
git log --oneline -5
```

## Completion

After all 5 phases:
```bash
echo "DONE" > /Users/jonnyzzz/run-agent/conductor-loop/TASK_IMPLEMENT_ALL_DONE
```

Write final summary to `output.md` covering:
- Tasks completed per phase
- Tests that passed
- Any tasks that were deferred (with reason)
- Total commits made
