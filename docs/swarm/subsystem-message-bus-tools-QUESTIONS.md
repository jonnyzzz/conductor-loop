# Message Bus Tooling - Questions

- Q: What is the canonical message entry format for multi-line bodies (header + delimiter)?
  Proposed default: YAML front-matter delimited by `---` with required fields, followed by body until next `---` or EOF.
  A: Accepted. We use run-agent binary to manage it anyways.

- Q: What is the canonical set of message types (including START/STOP, ERROR, ISSUE)?
  Proposed default: FACT, QUESTION, ANSWER, USER, START, STOP, ERROR, ISSUE, INFO, WARNING.
  A: Add OBSERVATION. Looks good. Conduct additional research online to look for basic approaches of Agentic Memory. Check if there are any other types of messages we are missing.

- Q: Should the bus support issue/dependency relations (beads-style), and how are they encoded?
  Proposed default: For ISSUE entries, add issue_id and depends_on[] fields in the header.
  A: Yes, it should. We need message-answer like support too for other messages. Yes, parents[] should be enough. For new related message, the run-agent should be able to show the whole thread. All messages from the top to the current new related comment/update.

- Q: How should ANSWER/USER entries reference the QUESTION they respond to?
  Proposed default: Add reply_to=<msg_id> (and optional thread_id=<msg_id>); tooling enforces for type=ANSWER.
  A: Yes, the same way as above, via parents[] field. There can be multiple parents, it's acceptable.

- Q: Are user responses append-only, or may they edit prior entries?
  Proposed default: Append-only; users add new USER/ANSWER entries with reply_to.
  A: Append only.

- Q: Are post-message.sh/poll-message.sh still first-class interfaces, or should they be thin wrappers around run-agent bus?
  Proposed default: run-agent bus is canonical; .sh scripts are optional wrappers for compatibility.
  A: All goes to run-agent command and sub commands. There must be no other scripts or commands.

- Q: Should polling use a cursor/offset (msg_id or byte offset) in addition to timestamps?
  A: Add --cursor/--since-id and keep --since for human troubleshooting.

- Q: What is the compaction/archival policy (what to archive vs delete)?
  Proposed default: Archive entries older than N days (configurable), retain FACT/DECISION/ISSUE; delete only on explicit command.
  A: Only explicit. Can be postponed to backlog. With run-agent binary, we can manage it. Intoduce filters for message types to allow skipping some messages.

- Q: How should HTTP streaming map to run-agent bus (endpoint + auth)?
  Proposed default: GET /api/bus/stream?project=<id>&task=<id> with token auth; returns text/event-stream.
  A: Looks good. We do not implement any authentication right now. The REST should have the same set of parameters as the console app.

- Q: What ordering guarantees should bus readers assume when multiple agents append concurrently?
  A: Timestamp ordering only; we include timestamp and pid into msg_id, use that for ordering. No need for complexity, so it's ok at low probabilities to get an incorrect order.

- Q: Should poll support server-side filters (type, run_id, issue_id)?
  Proposed default: Yes; add --type/--run-id/--issue-id filters.
  A: Just use the same setup as for the CLI

- Q: How should run-agent bus handle partial writes or corrupted entries?
  Proposed default: Detect and skip malformed entries; log to ISSUES.md and continue.
  A: It should recover. It should use the atomic way of writing to disk (write temp file, assert size, swap). We should not let agents touch these files directly.
