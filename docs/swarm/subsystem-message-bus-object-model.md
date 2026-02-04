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
- `kind`: relationship type (see Open Questions).

#### Optional Fields
- `meta`: free-form map for future extensions (e.g., timestamps, reasons).

## Backward Compatibility
- Tooling MUST accept string entries and treat them as `{msg_id: <msg_id>, kind: reply}`.
- Writers SHOULD prefer object form for any non-reply relationship.

## Related Files
- subsystem-message-bus-tools.md
- subsystem-message-bus-tools-QUESTIONS.md
