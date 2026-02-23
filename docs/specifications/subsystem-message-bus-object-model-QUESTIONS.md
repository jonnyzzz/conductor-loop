# Message Bus Object Model - Questions

## Status
All previously open questions are resolved against current implementation (`internal/messagebus/messagebus.go`).

## Resolved

1. Does the parser support object-form `parents` with `kind`/`meta`?
Answer: yes.
Evidence: `parseParents` accepts both string arrays and object arrays; string entries are converted into `Parent{MsgID: ...}`.

2. Should `issue_id` be a dedicated field or alias to `msg_id`?
Answer: alias semantics are implemented.
Evidence: `AppendMessage` auto-fills `issue_id = msg_id` when `type == "ISSUE"` and `issue_id` is empty.

3. Are parent/dependency kinds enforced?
Answer: no.
Evidence: parent `kind` is accepted as free-form text; bus append validation only checks non-empty `type` and `project_id`.

## Notes
- Current canonical schema includes `links` with `url/label/kind` and does not include `attachments` fields.
