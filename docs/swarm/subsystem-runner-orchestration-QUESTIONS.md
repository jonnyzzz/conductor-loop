# Runner & Orchestration - Questions

## Open Questions (From Codex Review Round 2)

### Q1: Config Credential Schema - Which Approach?
**Issue**: Backend specs reference `openai_api_key`, `anthropic_api_key`, etc. as "config keys", but config schema uses per-agent `token` + `env_var` fields.

**Config Schema** (subsystem-runner-orchestration-config-schema.md:92-108):
```hcl
agent "codex" {
  token = "@~/.config/openai/token"
  env_var = "OPENAI_API_KEY"
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

**Answer**: [PENDING]

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

**Answer**: [PENDING]

---

All previously resolved questions:
- Config validation: Addressed in subsystem-runner-orchestration-config-schema.md
- Binary updates: Manual install/rebuild for MVP All previous questions have been resolved and integrated into subsystem-runner-orchestration.md and subsystem-runner-orchestration-config-schema.md (to be created).
