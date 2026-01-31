# Message Bus Tooling Subsystem

## Overview
Provides the command-line tooling and conventions for writing to and reading from project/task message buses. Agents must communicate only through message bus files.

## Goals
- Provide a consistent way to append messages.
- Enable blocking or polling reads.
- Support both CLI and REST access via message-bus MCP.

## Non-Goals
- Implementing the message-bus service itself.
- Replacing existing message bus MCP features.

## Responsibilities
- Define message format.
- Define post-message.sh and poll-message.sh behavior.
- Define how agents route messages to project vs task bus.

## Interfaces / CLI
### post-message.sh
- Purpose: append a message entry.
- Args:
  - --project <id>
  - --task <id> (optional)
  - --type <info|question|decision|error>
  - --message <text>
- Routing:
  - If --task is set, append to TASK-MESSAGE-BUS.md.
  - Else, append to PROJECT-MESSAGE-BUS.md.

### poll-message.sh
- Purpose: read and optionally wait for new messages.
- Args:
  - --project <id>
  - --task <id> (optional)
  - --wait (block until new message)
  - --since <timestamp> (optional)
- Output: new messages only (append-only stream).

## Message Format
Append-only entries, e.g.:

[2026-01-31T20:00:00Z] type=question project=alpha task=task-20260131-foo
Message: What is the desired restart policy?

## Workflows
### Agent posting a question
1. Agent calls post-message.sh --type question.
2. User responds by editing the same message bus file.
3. Agent poller detects the response and spawns a sub agent if needed.

### Polling
- poll-message.sh is run by a background agent.
- On new message, it posts a notification or starts a processing agent.

## Error Handling
- If message bus file is missing, create it.
- If file is locked, retry with exponential backoff.

## Observability
- post-message.sh logs to stdout the appended line.
- poll-message.sh logs wait timeouts and last seen timestamp.

## Security / Permissions
- Only the local user can write to message bus files.
- Avoid secrets in message bus; use SECRETS.md or env vars instead.
