# Storage & Data Layout - Questions

## Open Questions

None. All previously open questions have been resolved.

---

## Resolved Questions

### Q1: run_id timestamp precision
**Question**: Should run_id timestamp precision be standardized to 4-digit fractional seconds (Go layout `20060102-1504050000`) across internal/storage and runner?

**Answer**: 4-digit fractional seconds. Format: `YYYYMMDD-HHMMSSffff-PID-SEQ` where SEQ is a process-local atomic counter added to prevent same-process same-millisecond collisions. Example: `20260223-2048580000-66788-1`.

---

### Q2: end_time and exit_code field presence
**Question**: Should run-info.yaml always include version/end_time/exit_code fields (even when zero), or is omission acceptable?

**Answer**: Yes, always include. `end_time` and `exit_code` are always present. While running: `exit_code = -1`, `end_time = zero value`. On success: `exit_code = 0`. On failure: `exit_code` = non-zero value.

---

### Q3: Per-run metadata files
**Question**: Do we want per-run metadata files (parent-run-id, agent-type, cwd, commandline) in addition to run-info.yaml?

**Answer**: No. Keep all data in run-info.yaml. No separate per-run metadata files.

---

### Q4: Task ID enforcement
**Question**: Should task IDs be enforced to follow `task-<timestamp>-<slug>` by the CLI, or remain caller-defined?

**Answer**: Yes, enforced by `run-agent`. Fully controlled by the binary. Invalid formats fail immediately with exit code 1.

---

### Q5: Task fact filenames
**Question**: Should task fact filenames include a `<name>` suffix (e.g., `TASK-FACTS-<timestamp>-<name>.md`) to match project FACT naming?

**Answer**: Not in this release. Deferred to backlog. Use the same timestamp format as task_id/run_id for generating IDs.

---

### Q6: Run start/stop event storage
**Question**: Should run start/stop events be stored in a dedicated log file in addition to message bus entries?

**Answer**: No dedicated log file. Events live in TASK-MESSAGE-BUS.md only. Keep 1 file.

---

## Previously Resolved Questions
- UTF-8 encoding: Strict UTF-8 without BOM (integrated into spec)
- Schema versioning: run-info.yaml v1 defined in subsystem-storage-layout-run-info-schema.md
