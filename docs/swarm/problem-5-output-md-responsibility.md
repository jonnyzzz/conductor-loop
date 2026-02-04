# Problem: output.md Creation Responsibility Ambiguity

## Context
Conflicting specifications:
- Agent protocol: "Agents SHOULD write output.md" (best-effort)
- Backend specs: "runner may create output.md" (Claude/Codex)
- CLI agents cannot write files natively

## Problem
Who is responsible for ensuring output.md exists?

## Proposed Solution (from Gemini #1)
**Unified Rule**: If output.md doesn't exist after agent exits, runner MUST copy agent-stdout.txt → output.md

## Your Task
Decide:
1. **Approach A: Runner Fallback** - Runner always creates output.md from stdout if missing
2. **Approach B: Agent Required** - Make it agent's strict responsibility, fail if missing
3. **Approach C: Backend-Specific** - Each backend handles differently

Specify:
- Who creates output.md and when?
- What if stdout is empty?
- What about stderr?
- Changes to subsystem-agent-protocol.md and backend specs
