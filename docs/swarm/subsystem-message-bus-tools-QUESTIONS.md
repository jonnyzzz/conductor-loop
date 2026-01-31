# Message Bus Tooling - Questions

- Q: Should message bus entries include a unique message id?
  Proposed default: Yes, add msg_id=<ulid> to each entry.
=== A: yes, and projectId, taskId, runId too. Read (and assert these) from environment variables

- Q: Should poll-message.sh support JSON output for easier parsing?
  Proposed default: Provide --json flag that wraps each entry as JSON.
=== A: no. Just text answers and clear separators between items.

- Q: How are user responses distinguished from agent messages?
  Proposed default: Add author=<user|agent> field in the header line.
=== A: there is type field, use FACT, QUESTION, ANSWER, USER. Explain the meaning in the prompts