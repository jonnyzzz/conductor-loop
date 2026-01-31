# Message Bus Tooling - Questions

- Q: Should message bus entries include a unique message id?
  Proposed default: Yes, add msg_id=<ulid> to each entry.

- Q: Should poll-message.sh support JSON output for easier parsing?
  Proposed default: Provide --json flag that wraps each entry as JSON.

- Q: How are user responses distinguished from agent messages?
  Proposed default: Add author=<user|agent> field in the header line.
