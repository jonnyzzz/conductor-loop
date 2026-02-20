# Message Bus Object Model

## Overview
Defines the object schema for structured message relationships in the message bus. This extends the basic parents[] threading mechanism to support richer dependency semantics ("beads"-style relationships) while remaining backward-compatible with simple parent msg_id lists.

## Goals
- Provide a stable schema for structured parents[] entries.
- Allow richer relationships beyond simple replies (dependencies, supersedes, duplicates).
- Support issue linkage without requiring a full issue tracker.
- Keep compatibility with string-only parents[] entries.

## Non-Goals
- Enforcing relationship semantics at runtime.
- Providing stateful issue workflow enforcement.

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
- `meta`: free-form map for future extensions (timestamps, reasons, UI hints).

## Validation
- Tooling MUST accept any `kind` string; no validation or warnings in MVP.
- Writers SHOULD use snake_case for `kind` values to keep conventions consistent.

## Issue & Dependency Modeling
- `ISSUE` messages represent issue records. If `issue_id` is omitted, use `msg_id` as the issue identifier.
- Use `parents[]` with relationship kinds to express dependencies and links between issues or between issues and other messages.
- For beads-style chains, use `child_of` or `depends_on` to link sequential work items and `supersedes` for replacements.

### Suggested Relationship Kinds (Non-Normative)
Common values seen in practice:
- reply
- answers
- relates_to
- depends_on
- blocks
- blocked_by
- supersedes
- duplicates
- child_of

These are recommendations only; tooling remains permissive in MVP.

## Cross-Scope References
- Task messages MAY reference project-level messages via parents[].
- UI aggregates both PROJECT-MESSAGE-BUS.md and TASK-MESSAGE-BUS.md and resolves cross-scope references with clear scope labels (for example, "[project]", "[task]").

## Related Files
- subsystem-message-bus-tools.md
- subsystem-message-bus-tools-QUESTIONS.md
