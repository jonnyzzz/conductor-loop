# Storage & Data Layout - Questions

## Open Questions
1. Should run_id timestamp precision be standardized to 4-digit fractional seconds (Go layout `20060102-1504050000`) across internal/storage and runner? Current code uses 3 digits in internal/storage and 4 digits in runner.
Answer: (Pending - user)

2. Should run-info.yaml always include version/end_time/exit_code fields (even when zero), or is omission acceptable? Current code omits version in internal/storage and omits end_time/exit_code when zero.
Answer: (Pending - user)

3. Do we want per-run metadata files (parent-run-id, agent-type, cwd, commandline) in addition to run-info.yaml, as suggested in ideas.md, or is run-info.yaml sufficient?
Answer: (Pending - user)

4. Should task IDs be enforced to follow `task-<timestamp>-<slug>` by the CLI, or remain caller-defined?
Answer: (Pending - user)

5. Should task fact filenames include a `<name>` suffix (e.g., `TASK-FACTS-<timestamp>-<name>.md`) to match project FACT naming?
Answer: (Pending - user)

6. Should run start/stop events be stored in a dedicated log file in addition to message bus entries?
Answer: (Pending - user)

## Resolved Questions
- UTF-8 encoding: Strict UTF-8 without BOM (integrated into spec)
- Schema versioning: run-info.yaml v1 defined in subsystem-storage-layout-run-info-schema.md
