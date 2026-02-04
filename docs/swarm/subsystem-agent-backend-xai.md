# Agent Backend: xAI

## Overview
Defines the placeholder for xAI backend integration. This work is deferred post-MVP.

## Goals
- Capture the intended integration surface for xAI.
- Document the intended future integration path (OpenCode + xAI).

## Non-Goals
- Implementing the xAI adapter before post-MVP work begins.

## Invocation (Planned)
- Use OpenCode agent configured to target xAI models (only if xAI token is provided).
- Default to the best available xAI model when enabled.

## I/O Contract
- Input: prompt text (from prompt.md).
- Output: plain text response captured into output.md.
- Errors: logged to agent-stderr.txt; non-zero exit code for failures.

## Environment / Config
- API key storage follows the same config.hcl pattern as other backends (token required to enable xAI).

## Related Files
- subsystem-runner-orchestration.md
- subsystem-env-contract.md
