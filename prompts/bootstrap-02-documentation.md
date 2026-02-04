# Task: Create Documentation Structure

**Task ID**: bootstrap-02
**Phase**: Bootstrap
**Agent Type**: Research/Documentation (Claude preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Set up project documentation following THE_PROMPT_v5.md conventions.

## Required Actions

1. **Create AGENTS.md**
   Define:
   - Project conventions (Go style, commit format)
   - Agent types (Orchestrator, Implementation, Review, Test, Debug)
   - Permissions (file access, tool access)
   - Subsystem ownership

2. **Create Instructions.md**
   Document:
   - Repository structure
   - Build commands
   - Test commands
   - Tool paths (go, docker, make)
   - Environment setup

3. **Create Role Prompt Files**
   Copy from THE_PROMPT_v5.md template and adapt:
   - THE_PROMPT_v5_orchestrator.md
   - THE_PROMPT_v5_research.md
   - THE_PROMPT_v5_implementation.md
   - THE_PROMPT_v5_review.md
   - THE_PROMPT_v5_test.md
   - THE_PROMPT_v5_debug.md

4. **Create DEVELOPMENT.md**
   - Local development setup
   - Running tests
   - Debugging tips
   - Contributing guidelines

## Success Criteria
- All role prompt files exist
- AGENTS.md defines clear conventions
- Instructions.md has all tool paths

## References
- THE_PROMPT_v5.md: Role-Specific Prompts section
- docs/specifications/ for technical details

## Output
Log to MESSAGE-BUS.md:
- FACT: Documentation structure created
- FACT: Role prompts ready
