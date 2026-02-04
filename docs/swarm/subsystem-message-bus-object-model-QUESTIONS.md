# Message Bus Object Model - Questions

- Q: What is the canonical parents[] object schema (required fields and field names, e.g., msg_id vs message_id)? | Proposed default: parents objects require msg_id and kind; allow optional meta; accept string shorthand for reply. | A: TBD.
- Q: What is the initial vocabulary of relationship kinds (reply, blocks, supersedes, relates_to, answers, etc.) for parents[]? | Proposed default: Start with reply (default), blocks, supersedes, relates_to, answers; extend later. | A: TBD.
