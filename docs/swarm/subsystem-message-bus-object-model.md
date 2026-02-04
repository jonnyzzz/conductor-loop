# Message Bus Object Model

## Overview
Defines the object schema for structured message relationships in the message bus. This extends the basic parents[] threading mechanism to support richer dependency semantics ("beads"-style relationships) while remaining backward-compatible with simple parent msg_id lists.

## Goals
- Provide a stable schema for structured parents[] entries.
- Allow richer relationships beyond simple replies (e.g., blocks, supersedes).
- Keep compatibility with string-only parents[] entries.

## Non-Goals
- Defining a full issue-tracking workflow.
- Enforcing relationship semantics at runtime (tooling may remain permissive in MVP).

## parents[] Schema
The `parents` header field accepts either:
- A list of strings (msg_id values), or
- A list of objects with relationship metadata.

### String Form (Shorthand)
```
parents: [<msg_id>, <msg_id>, ...]
```
Equivalent to object form with `kind: reply` for each entry.

### Object Form
```
parents:
  - msg_id: <msg_id>
    kind: <relationship>
    meta: <optional object>
```

#### Required Fields
- `msg_id`: ID of the referenced message.
- `kind`: relationship type (free-form; no fixed vocabulary).

#### Optional Fields
- `meta`: free-form map for future extensions (e.g., timestamps, reasons).

## Backward Compatibility
- Tooling MUST accept string entries and treat them as `{msg_id: <msg_id>, kind: reply}`.
- Writers SHOULD prefer object form for any non-reply relationship.

## Validation
- Tooling MUST accept any `kind` string; no validation or warnings in MVP.
- Writers SHOULD use snake_case for `kind` values to keep conventions consistent.

## Cross-Scope References
- Task messages MAY reference project-level messages via parents[].
- UI aggregates both PROJECT-MESSAGE-BUS.md and TASK-MESSAGE-BUS.md and resolves cross-scope references with clear scope labels (e.g., "[project]", "[task]").

## Suggested Relationship Kinds (Non-Normative)
Common values seen in practice:
- reply
- blocks
- supersedes
- relates_to
- answers

These are recommendations only; tooling remains permissive in MVP.

## Related Files
- subsystem-message-bus-tools.md
- subsystem-message-bus-tools-QUESTIONS.md
