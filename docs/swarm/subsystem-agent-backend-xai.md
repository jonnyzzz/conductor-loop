# Agent Backend: xAI

## Overview
Defines the placeholder for xAI backend integration. The specific coding agent/model and invocation method are pending research.

## Goals
- Capture the intended integration surface for xAI.
- Track open questions required before implementation.

## Non-Goals
- Implementing the xAI adapter before research is complete.

## Invocation (TBD)
- Research required to decide API vs CLI integration and model selection.

## I/O Contract
- Input: prompt text (from prompt.md).
- Output: plain text response captured into output.md.
- Errors: logged to agent-stderr.txt; non-zero exit code for failures.

## Environment / Config
- API key storage and injection is TBD; expected to follow the same config.hcl pattern as other backends.

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
