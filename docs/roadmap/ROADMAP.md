# Product Roadmap: Project Evolution

Generated: 2026-02-24
Status: Draft

## Q1 2026 (Reliability & Fixes)
**Theme**: Stability, correctness, and release readiness.

| Workstream | Task | Complexity | Prompt File |
| :--- | :--- | :--- | :--- |
| **Reliability** | Fix Conductor Binary Port (8080 vs 14355) | S | `prompts/tasks/fix-conductor-binary-port.md` |
| **Reliability** | Fix Monitor Process Cap | M | `prompts/tasks/fix-monitor-process-cap.md` |
| **Reliability** | Fix SSE Stream CPU Hotspot | M | `prompts/tasks/fix-sse-cpu-hotspot.md` |
| **Correctness** | Implement `output synthesize` | M | `prompts/tasks/implement-output-synthesize.md` |
| **Correctness** | Implement `review quorum` | S | `prompts/tasks/implement-review-quorum.md` |
| **Correctness** | Implement `iterate` | M | `prompts/tasks/implement-iterate.md` |
| **Security** | Repository Token Leak Audit | L | `prompts/tasks/token-leak-audit.md` |
| **Release** | First Release Readiness Gate | M | `prompts/tasks/release-readiness-gate.md` |
| **Release** | Unified Bootstrap Script | M | `prompts/tasks/unified-bootstrap.md` |
| **UX** | Fix Web UI Latency | M | `prompts/tasks/ui-latency-fix.md` |
| **UX** | Add Regression Test Suite for UI Task Tree | M | `prompts/tasks/ui-task-tree-guardrails.md` |
| **Compat** | Implement Gemini Stream JSON Fallback | S | `prompts/tasks/gemini-stream-json-fallback.md` |

## Q2 2026 (Evolution & Architecture)
**Theme**: Architectural cleanup, Windows support, and scaling.

| Workstream | Task | Complexity | Prompt File |
| :--- | :--- | :--- | :--- |
| **Architecture** | Merge `conductor` and `run-agent` Binaries | M | `prompts/tasks/merge-conductor-run-agent.md` |
| **Architecture** | Deprecate HCL Config Support | S | `prompts/tasks/hcl-config-deprecation.md` |
| **Architecture** | Implement Environment Sanitization | M | `prompts/tasks/env-sanitization.md` |
| **Windows** | Implement Windows File Locking | L | `prompts/tasks/windows-file-locking.md` |
| **Windows** | Implement Windows Process Groups | L | `prompts/tasks/windows-process-groups.md` |

## Q3 2026 (Innovation)
**Theme**: Advanced orchestration and knowledge management.

| Workstream | Task | Complexity | Prompt File |
| :--- | :--- | :--- | :--- |
| **Architecture** | Global Fact Storage & Promotion | L | `prompts/tasks/global-fact-storage.md` |
