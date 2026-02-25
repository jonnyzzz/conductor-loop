# Task: Environment Sanitization — Inject Only Agent-Specific API Keys

## Context

**Current behavior** (from `docs/facts/FACTS-swarm-ideas.md:120`, `docs/facts/FACTS-architecture.md`):
- Agents inherit the **full parent process environment** (MVP: "full inheritance in MVP, no sandbox").
- This means every agent (Claude, Gemini, Codex, Perplexity, xAI) receives all API keys that are set in the shell environment — including keys for agents it should never need.

**Security risk documented** (`docs/facts/FACTS-swarm-ideas.md:122-123`):
- "Agent should not be able to manipulate environment variables. Example from Codex changing MESSAGE-BUS variable caused mixed messages between tasks. Env var injection protection is critical."
- A compromised or misbehaving agent with access to foreign API keys can make unauthorized calls to services it has no business contacting.

**Current per-backend env var mapping** (`docs/facts/FACTS-swarm-ideas.md:165`):
| Agent | Expected Env Var |
|-------|-----------------|
| Codex | `OPENAI_API_KEY` |
| Claude | `ANTHROPIC_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Perplexity | `PERPLEXITY_API_KEY` |
| xAI | `XAI_API_KEY` (or equivalent) |

**What should happen**: When launching a Claude agent, the child process should have `ANTHROPIC_API_KEY` set but `OPENAI_API_KEY`, `GEMINI_API_KEY`, `PERPLEXITY_API_KEY`, and `XAI_API_KEY` should be **unset** (or not inherited).

**Relevant code location**: `internal/runner/job.go` — where agent processes are launched and environment is constructed before `cmd.Start()`.

## Requirements

1. **Implement an environment allow-list per agent type**:
   - Create a mapping in `internal/runner/` (or `internal/agent/`) that defines which environment variables each agent type is permitted to receive.
   - The allow-list must include: system path vars (`PATH`, `HOME`, `TMPDIR`, `USER`, `SHELL`, `LANG`, `LC_ALL`, `TERM`), JRUN-internal vars (`JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID`, `JRUN_MESSAGE_BUS`, `JRUN_RUN_FOLDER`, `JRUN_TASK_FOLDER`), and the single API key relevant to the agent type.
   - All other vars (especially other agents' API keys) must be **excluded**.

2. **Sanitize the child environment in the runner**:
   - In `internal/runner/job.go` (or wherever `exec.Cmd` is created for agent runs), build `cmd.Env` explicitly from the allow-list rather than inheriting `os.Environ()`.
   - Do NOT use `cmd.Env = nil` (which inherits everything). Construct a clean slice.

3. **Verify the implementation does not break existing functionality**:
   - Agents that require specific env vars beyond API keys (e.g., Codex may need `OPENAI_ORG_ID`) must still receive them — add these to the per-agent allow-list as needed.
   - The `run-agent` binary's own env vars needed for sub-commands (e.g., `JRUN_*`) must still propagate.

4. **Log the sanitized env vars (debug level)**:
   - On agent launch, log (at debug/trace level) which env vars were passed and which were stripped — useful for diagnosing auth failures.
   - Do NOT log the values of API keys, only their names.

5. **Add tests**:
   - Unit test: given a full environment with all known API keys, running the sanitize function for `claude` agent type returns only `ANTHROPIC_API_KEY` (and allow-listed system vars) — not `OPENAI_API_KEY` etc.
   - Integration test (or smoke test): Start an agent run and verify the agent subprocess does not have access to foreign API keys. This can be done by writing a minimal test agent that prints its environment.

6. **Config extensibility** (optional but preferred):
   - Allow the per-agent env allow-list to be extended via `config.yaml` (e.g., `agents.claude.extra_env: [MY_CUSTOM_VAR]`) so operators can add project-specific env vars without code changes.

## Acceptance Criteria

- A Claude agent run does NOT receive `OPENAI_API_KEY`, `GEMINI_API_KEY`, or `PERPLEXITY_API_KEY` in its process environment.
- A Codex agent run does NOT receive `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, or `PERPLEXITY_API_KEY`.
- All JRUN_* variables and system path vars are always present regardless of agent type.
- Existing agent integration tests still pass (agents can authenticate with their respective API services).
- Unit tests for the environment sanitization function pass.

## Verification

```bash
# Unit tests
go test ./internal/runner/... -run TestEnvSanitize -v

# Manually verify env in a running agent:
# Launch a test Claude run whose task is: "Print all environment variables that start with
# OPENAI, GEMINI, PERPLEXITY, ANTHROPIC, XAI. Write results to output.md."
# Then check output.md — Claude should see ANTHROPIC_API_KEY only.

# Inspect what env vars the runner passes
grep -n "cmd.Env\|os.Environ\|Environ()" internal/runner/job.go

# Verify no foreign keys leak
go test ./internal/runner/... -v -run TestAgentEnv
```

## Key Source Files

- `internal/runner/job.go` — primary location where `exec.Cmd` is constructed for agent runs
- `internal/runner/` — may need a new `env_sanitize.go` file
- `internal/config/config.go` — for any config-based allow-list extension
- `docs/facts/FACTS-swarm-ideas.md:165` — canonical per-backend env var mapping reference
