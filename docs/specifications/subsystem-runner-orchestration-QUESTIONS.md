# Runner & Orchestration - Questions

## Open Questions

None. All previously open questions have been resolved.

---

## Resolved Questions

### Q1: Config Credential Schema - Which Approach?
**Issue**: Backend specs reference `openai_api_key`, `anthropic_api_key`, etc. as "config keys", but config schema uses per-agent `token` + `env_var` fields.

**Answer**: The env var for each agent is always a constant, hardcoded in runner. Only parameters needed per agent in config are: `token` (inline) or `token_file` (file path). Token file contents are read, trimmed, and used as the token value. The `env_var` config field is removed from schema. See subsystem-runner-orchestration-config-schema.md for full schema.

---

### Q2: Codex cli_flags Example Incomplete
**Issue**: Config schema example showed incomplete cli_flags for Codex invocation (missing CWD argument and stdin marker).

**Answer**: Runner starts agents in the correct working directory. No `cli_flags` configuration is needed. All CLI flags and working directory setup are hardcoded in the runner tool. Agents run in unrestricted mode.

---

### Q3: Config Format and Default Location
**Issue**: The runner spec targeted HCL at `~/run-agent/config.hcl`, but the implementation loads YAML and only when `--config` is provided.

**Answer**: Both YAML and HCL are supported. YAML is primary (checked first). Default search order: `./config.yaml` → `./config.yml` → `./config.hcl` → `$HOME/.config/conductor/config.yaml` → `$HOME/.config/conductor/config.yml` → `$HOME/.config/conductor/config.hcl`.

---

### Q4: run-agent CLI Surface Area
**Issue**: The spec included `run-agent serve`, `bus`, and `stop`, but `cmd/run-agent` only exposed `task` and `job`.

**Answer**: `run-agent serve`, `bus`, and `stop` are all implemented.

---

### Q5: Agent Selection Strategy
**Issue**: Spec called for round-robin or weighted selection and "I'm lucky" random mode with cooldown on failures. The code picked the configured default or first agent alphabetically.

**Answer**: The parent agent (not runner) decides which sub-agent to select. The diversification policy (round-robin/weighted with fallback-on-failure) is implemented in the runner's `DiversificationConfig`. The balancing/round-robin idea was logged for future use only originally; now implemented.

---

### Q6: Delegation Depth Limit
**Issue**: The spec set a default delegation depth limit of 16, but there was no enforcement or tracking.

**Answer**: Delegation depth limit enforcement is not implemented in this release. Logged for future use.

---

### Q7: Restart Prompt Prefix
**Issue**: The spec required prefixing restarts with "Continue working on the following:", but the runner did not modify prompts on restart.

**Answer**: Yes, prepend "Continue working on the following:" on restart attempts > 0. The preamble is always included, even on restarts.

---

### Q8: Environment Safety for Runner-Owned Variables
**Issue**: The spec reserves JRUN_* variables and disallows overrides, but the runner merged caller-provided environment values after setting JRUN_*.

**Answer**: The runner sets JRUN_* variables correctly in the started agent process. Agent process will start `run-agent` again for sub-agents. Variables must be maintained carefully. Assert and validate consistency.

---

### Q9: Start/Stop Event Detail
**Issue**: Ideas called for a dedicated start/stop log and richer metadata, while the implementation posted RUN_START/RUN_STOP messages with minimal content.

**Answer**: Each run has its `run_id` folder. Start/stop events include exit code, folder path, and known output files (if any). No dedicated event log file — events live in TASK-MESSAGE-BUS.md.

---

### Q10: Task Folder Naming and Creation
**Issue**: The storage spec expected `task-<timestamp>-<slug>` folders and TASK.md to exist, but `run-agent task` used raw task IDs and did not create TASK.md.

**Answer**: Yes, `run-agent` takes care of consistency of the folders. It assigns TASK_ID and creates all necessary files and folders according to the specs. The task_id format is enforced by the CLI; invalid formats fail immediately with exit code 1.

---

All previously resolved questions:
- Config validation: Addressed in subsystem-runner-orchestration-config-schema.md
- Binary updates: Manual install/rebuild for MVP
