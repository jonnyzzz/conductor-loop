# Architecture Review Notes - Detailed Docs (C)

Date: 2026-02-24
Scope reviewed:
- docs/architecture/agent-integration.md
- docs/architecture/deployment.md
- docs/architecture/frontend-architecture.md
- docs/architecture/observability.md
- docs/architecture/security.md
- docs/architecture/concurrency.md

Method:
- Compared each doc against `docs/facts/FACTS-*.md` (with `FACTS-reconciled.md` precedence guidance), then verified uncertain points against current code in `internal/*`, `cmd/*`, `frontend/*`, and `web/*`.

## Findings (ordered by severity)

### 1) High - `frontend-architecture.md` has stale API route prefixes and stale embedding model

Evidence:
- `frontend-architecture.md:52,55,59,60` documents project APIs under `/api/v1/projects...`.
- Current routes and frontend client use `/api/projects...` for project/task/run/project-stats APIs:
  - `internal/api/routes.go:34-37`
  - `frontend/src/api/client.ts` (`/api/projects/...` throughout)
- `frontend-architecture.md:25` says frontend assets are embedded via `go:embed` as part of primary build flow.
- Current implementation embeds fallback `web/src` only; primary React UI is served from filesystem `frontend/dist` when present:
  - `web/embed.go:11-20`
  - `internal/api/routes.go:38-40,60-70`

Impact:
- API contract section is misleading for maintainers/integrators.
- Deployment/packaging expectations are incorrect (primary React bundle is not embedded today).

Recommended correction:
- Replace `/api/v1/projects...` references with `/api/projects...` where applicable.
- Clarify mixed API surface:
  - `/api/v1/*` for core v1 endpoints (health/version/status/runs/messages stream-all)
  - `/api/projects/*` for project-centric UI APIs.
- Update embedding section to state: fallback UI is embedded (`web/src`), primary React UI is preferred from `frontend/dist` when available.

### 2) High - `observability.md` metric names do not match emitted Prometheus series

Evidence:
- `observability.md:14-18` lists:
  - `conductor_active_runs`
  - `conductor_message_bus_append_total`
  - `conductor_queued_runs`
- Actual metric names are:
  - `conductor_active_runs_total`
  - `conductor_messagebus_appends_total`
  - `conductor_queued_runs_total`
  - plus `conductor_completed_runs_total`, `conductor_agent_runs_total`, `conductor_agent_fallbacks_total`
  - source: `internal/metrics/metrics.go:150-173,210-243`
  - corroborated by `docs/facts/FACTS-user-docs.md:163`

Impact:
- Dashboards/alerts built from this doc will query non-existent metrics.

Recommended correction:
- Update listed metric names to exact exported series.
- Include `conductor_completed_runs_total` and optional per-agent counters.

### 3) Medium - `security.md` overstates UI destructive-action guard behavior

Evidence:
- `security.md:171` says browser-origin detection includes `User-Agent` patterns.
- Actual detector checks `X-Conductor-Client`, `Sec-Fetch-*`, `Origin`, `Referer`; no `User-Agent` check:
  - `internal/api/ui_safety.go:17-35`
- `security.md:180` says `403` is returned regardless of auth state because block is applied before auth middleware.
- Actual middleware order applies auth wrapper before handlers:
  - `internal/api/routes.go:53-56`
  - `internal/api/auth.go:18-49`
  - `rejectUIDestructiveAction(...)` is called inside handlers (after auth pass): `internal/api/handlers_projects.go:1297,1328,2392`

Impact:
- Security behavior description is inaccurate for unauthenticated requests (they can fail with `401` before reaching UI-destructive guard).

Recommended correction:
- Remove `User-Agent` from detector description.
- Reword auth interaction: guard blocks browser-origin destructive actions in handler path; auth middleware may reject first when enabled and credentials are missing/invalid.

### 4) Medium - `agent-integration.md` CLI invocation rows include obsolete `-C <cwd>` flags

Evidence:
- `agent-integration.md:21-22` documents `claude ... -C <cwd>` and `codex ... -C <cwd>`.
- Current runner sets working dir via process spawn options (`SpawnOptions.Dir`) and command args do not include `-C`:
  - `internal/runner/job.go:515-520`
  - `internal/runner/job.go:850-868`
- This also aligns with older fact note that working dir is runner-controlled rather than requiring `-C` in command flags (`docs/facts/FACTS-runner-storage.md:386`).

Impact:
- Operator troubleshooting and commandline parity checks can fail due to inaccurate expected args.

Recommended correction:
- Update invocation table to match current `commandForAgent` arg lists.
- Add a note that cwd is set by runner process options, not CLI flags.

### 5) Low - `deployment.md` directory tree implies `TASK-CONFIG.yaml` is always present

Evidence:
- `deployment.md:24` includes `TASK-CONFIG.yaml` in canonical tree without marking it optional.
- Task dependency config file is created only when needed; removed when `depends_on` is empty:
  - `internal/taskdeps/taskdeps.go:84-85`

Impact:
- Minor documentation ambiguity about baseline task layout.

Recommended correction:
- Mark `TASK-CONFIG.yaml` as optional in the tree.

## Docs reviewed with no material drift found

- `concurrency.md`: aligned with facts/code on Ralph loop behavior, run semaphore model, planner-based root-task cap, and message-bus lock semantics.
- `deployment.md`: otherwise aligned (YAML-primary config priority, self-update state machine, GC flags, filesystem-first model).

## Explicit stale-check note requested in task prompt

- No stale "HCL is primary" claim found in reviewed architecture docs.
- Reviewed docs correctly describe YAML as primary with HCL compatibility (`deployment.md:46`), consistent with `docs/facts/FACTS-reconciled.md:34`.
