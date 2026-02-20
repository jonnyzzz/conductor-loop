# Runner & Orchestration - Questions

## Open Questions (From Codex Review Round 2)

### Q1: Config Credential Schema - Which Approach?
**Issue**: Backend specs reference `openai_api_key`, `anthropic_api_key`, etc. as "config keys", but config schema uses per-agent `token` + `env_var` fields.

**Config Schema** (subsystem-runner-orchestration-config-schema.md:92-108):
```hcl
agent "codex" {
  token = "@~/.config/openai/token"  <-- PROPSED: use dedicated field for token_file instead of @ inside.
  env_var = "OPENAI_API_KEY"  <--- ANSWER remove that, it's implementation detail we can easily hardcode.
}
```

**Backend Specs** (e.g., subsystem-agent-backend-codex.md:30):
```
- Config key: `openai_api_key` (in `config.hcl`)
```

**Question**: Should backend specs be updated to reference the config schema's `token` + `env_var` approach? If yes, how should we phrase it in backend specs?

**Proposed Fix**: Update all backend specs from:
```
- Config key: `openai_api_key` (in `config.hcl`)
```
To:
```
- Config: Token set in agent "codex" block with `token` field and `env_var = "OPENAI_API_KEY"`
- See subsystem-runner-orchestration-config-schema.md for full schema
```

**Answer**: The env var for each agent is always a constant, we do not need it in the config schema.
The only parameters we include for each of the agents are -- token -- a token value. We need to allow to provide `token_file` instead of the token.
In the case of Token File, it must read the file, trim the contents and use that as the token value.

See comments in the code.

---

### Q2: Codex cli_flags Example Incomplete
**Issue**: Config schema example for Codex (line 96) shows:
```hcl
cli_flags = ["exec", "--dangerously-bypass-approvals-and-sandbox", "-C"]
```
But actual invocation in run-agent.sh:42 is:
```bash
codex exec --dangerously-bypass-approvals-and-sandbox -C "$CWD" -
```
Missing: the CWD argument value and the `-` stdin marker.

**Question**: Should cli_flags include placeholders like `"<cwd>"` and `"-"`, or should these be handled specially by the runner?

**Proposed Fix**: Either:
1. Document that `-C` requires CWD injection by runner, `-` for stdin is automatic
2. Or show complete example: `cli_flags = ["exec", "--dangerously-bypass-approvals-and-sandbox", "-C", "<cwd>", "-"]`

**Answer**: We just start agents in the given working directory, with all permisions (aka with no restictions). 
Our run-agent tool should make sure it sets the working directory correctly, so we do not need to use -C or something like that. 
At this point, out goal or all agents to start them in unrestricted mode, so we hardcode all set settings in our tool, 
and there is no need to provide any additional CLI flags in the tool configuration. It makes sense to wrap the agent process
invocation in a shell script, which is started with the correct working directory.

---

## New Questions (2026-02-05)

### Q3: Config Format and Default Location
**Issue**: The runner spec now targets HCL at ~/run-agent/config.hcl, but the current implementation loads YAML via internal/config and only when --config is provided.

**Question**: Should HCL be the single source of truth, with YAML deprecated? If yes, should run-agent default to ~/run-agent/config.hcl when --config is omitted?

**Answer**: (Pending - user)

---

### Q4: run-agent CLI Surface Area
**Issue**: The spec includes run-agent serve, bus, and stop. cmd/run-agent currently exposes only task and job.

**Question**: Should serve, bus, and stop be implemented now, or should the docs mark them as planned only?

**Answer**: (Pending - user)

---

### Q5: Agent Selection Strategy
**Issue**: The spec calls for round-robin or weighted selection and an I'm lucky random mode with cooldown on failures. The current code picks the configured default or the first agent alphabetically.

**Question**: Which selection algorithm should be considered authoritative, and how should cooldown or degradation be modeled?

**Answer**: (Pending - user)

---

### Q6: Delegation Depth Limit
**Issue**: The spec sets a default delegation depth limit of 16, but there is no enforcement or tracking in the runner.

**Question**: Where should depth be tracked (env var, run-info, or prompt), and what should run-agent do when the limit is exceeded?

**Answer**: (Pending - user)

---

### Q7: Restart Prompt Prefix
**Issue**: The spec requires prefixing restarts with "Continue working on the following:", but the current runner does not modify prompts on restart.

**Question**: Should the runner add this prefix, and should it also include previous output or TASK_STATE.md context?

**Answer**: (Pending - user)

---

### Q8: Environment Safety for Runner-Owned Variables
**Issue**: The spec reserves JRUN_* variables and disallows overrides, but the current runner merges caller-provided environment values after setting JRUN_*.

**Question**: Should the runner drop or overwrite reserved keys from caller-provided environment maps and inherited env?

**Answer**: (Pending - user)

---

### Q9: Start/Stop Event Detail
**Issue**: Ideas call for a dedicated start/stop log and richer metadata, while the current implementation posts RUN_START/RUN_STOP messages with minimal body content.

**Question**: Should start/stop events include pid, prompt path, and output path in the message bus, and should a dedicated event log file be added?

**Answer**: (Pending - user)

---

### Q10: Task Folder Naming and Creation
**Issue**: The storage spec expects task-<timestamp>-<slug> folders and TASK.md to exist, but run-agent task uses raw task IDs and does not create TASK.md.

**Question**: Should run-agent generate task IDs and write TASK.md when starting a task, or keep the current behavior and require pre-created task directories?

**Answer**: (Pending - user)

---

All previously resolved questions:
- Config validation: Addressed in subsystem-runner-orchestration-config-schema.md
- Binary updates: Manual install/rebuild for MVP
