# Product Roadmap: Project Evolution

Generated: 2026-02-24
Status: evo-r3 synthesis (2026-02-24 round 3 update)

## Prioritization Policy

- Q1 2026: P0 reliability only
- Q2 2026: P1 correctness, security, release, UX
- Q3 2026: architecture and platform evolution
- Long-term: ideas requiring additional design/prototyping

## Q1 2026 (P0 Reliability)

| Priority | Task | Complexity | prompt-file |
| :--- | :--- | :--- | :--- |
| ~~P0~~ | ~~Reconcile conductor binary default port (8080 vs 14355)~~ **RESOLVED 2026-02-24** | S | `prompts/tasks/fix-conductor-binary-port.md` |
| P0 | Reduce SSE CPU hotspot and full-bus reparse pressure | M | `prompts/tasks/fix-sse-cpu-hotspot.md` |
| P0 | Enforce monitor process cap/single ownership | M | `prompts/tasks/fix-monitor-process-cap.md` |
| P0 | Fix monitor stop-respawn race (suppression window) | S | `prompts/tasks/monitor-stop-respawn-race.md` |
| P0 | Blocked dependency deadlock recovery and diagnostics | M | `prompts/tasks/blocked-dependency-deadlock-recovery.md` |
| P0 | Add explicit run-status finish criteria (all_finished, blocked) | M | `prompts/tasks/run-status-finish-criteria.md` |
| P0 | Harden status/list/stop against missing run-info.yaml | S | `prompts/tasks/runinfo-missing-noise-hardening.md` |
| P0 | Webserver uptime auto-recovery (watchdog + health probe) | M | `prompts/tasks/webserver-uptime-autorecover.md` |

## Q2 2026 (P1 Correctness, Security, UX)

| Priority | Task | Complexity | prompt-file |
| :--- | :--- | :--- | :--- |
| P1 | Implement `run-agent output synthesize` | M | `prompts/tasks/implement-output-synthesize.md` |
| P1 | Implement `run-agent review quorum` | M | `prompts/tasks/implement-review-quorum.md` |
| P1 | Implement `run-agent iterate` | M | `prompts/tasks/implement-iterate.md` |
| P1 | Add Gemini stream-json compatibility fallback | S | `prompts/tasks/gemini-stream-json-fallback.md` |
| P1 | Fix Web UI update latency | M | `prompts/tasks/ui-latency-fix.md` |
| P1 | Add UI task-tree regression guardrails | M | `prompts/tasks/ui-task-tree-guardrails.md` |
| P1 | Run repository token leak audit and guardrails | L | `prompts/tasks/token-leak-audit.md` |
| P1 | Create release readiness gate script and policy | M | `prompts/tasks/release-readiness-gate.md` |
| P1 | Unify bootstrap/update scripts | M | `prompts/tasks/unified-bootstrap.md` |
| P1 | Fix message bus empty-state regression (SSE hydration) | M | `prompts/tasks/messagebus-empty-regression.md` |
| P1 | Lock live-log layout with regression tests | M | `prompts/tasks/live-logs-regression-guardrails.md` |
| P1 | New task submit durability (draft persistence, submit states) | M | `prompts/tasks/ui-new-task-submit-durability.md` |
| P1 | Define and enforce SSE/refresh CPU budget | M | `prompts/tasks/ui-refresh-churn-cpu-budget.md` |

## Q3 2026 (Architecture and Platform)

| Priority | Task | Complexity | prompt-file |
| :--- | :--- | :--- | :--- |
| Architecture | Merge/clarify `conductor` vs `run-agent` binary split | M | `prompts/tasks/merge-conductor-run-agent.md` |
| Architecture | Deprecate HCL config surface (YAML-only policy) | S | `prompts/tasks/hcl-config-deprecation.md` |
| Architecture | Sanitize child env to agent-specific secrets | M | `prompts/tasks/env-sanitization.md` |
| Architecture | Promote facts across task/project scopes | L | `prompts/tasks/global-fact-storage.md` |
| Platform (Windows) | Add shared-lock reader behavior | L | `prompts/tasks/windows-file-locking.md` |
| Platform (Windows) | Implement Job Object process-group control | L | `prompts/tasks/windows-process-groups.md` |
| P2 | Prevent run/* artifact clutter from polluting git status | S | `prompts/tasks/run-artifacts-git-hygiene.md` |

## Long-Term Ideas (Design Required)

- Beads-style dependency semantics for message bus parents and readiness queries.
- Multi-host monitoring UI support.
- Explicit pause/resume semantics beyond stop/restart.
- xAI model policy and coding-agent mode completion.
- Automated fact curation/librarian process beyond baseline promotion.

