# Message Bus Tooling - Questions

- Q: Do we need message dependency semantics beyond parents[] (e.g., issue relationships / "beads")?
  Proposed default: Keep parents[] threading only for MVP; add explicit relationship types later if needed.
  A: Yes. Let's allow each parent be of a form of an object, with fields `kind`, `message_id (aka the dependency message id, written the same name as it is for each message), any optional fields. Provide the dedicated subsystem specification to define object model of the messages in the message-bus. Run one more round to inspect beats features and layout.
