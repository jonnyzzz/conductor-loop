# Research Task: CLI Version Compatibility (ISSUE-004)

## Context
You are a research agent working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

## Required Reading
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md — see ISSUE-004
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — code conventions
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md — tool paths

## Task
Research the current state of CLI version detection in the conductor-loop codebase and propose what's still needed:

1. Read internal/runner/job.go — find the ValidateAgent function and understand what it currently does
2. Read internal/agent/ backends — check how each agent handles version detection
3. Check what `claude --version`, `codex --version`, `gemini --version` output formats look like (if available)
4. Propose a concrete implementation plan for version constraint enforcement:
   - Where should minimum version constraints be stored (config? hardcoded?)
   - How should version parsing work (semver? regex?)
   - What should happen when a version is incompatible (fail fast? warn?)
   - Should this be a separate validate-config command?

## Output
Write your findings to /Users/jonnyzzz/Work/conductor-loop/prompts/session-4/research-cli-version-compat-output.md

Create the DONE file when complete:
```bash
touch /Users/jonnyzzz/Work/conductor-loop/conductor-loop/task-research-cli-version/DONE
```
