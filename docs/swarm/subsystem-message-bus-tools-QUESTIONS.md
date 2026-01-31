# Message Bus Tooling - Questions

- Q: Should message bus entries include a unique message id?
  Proposed default: Yes, add msg_id=<ulid> to each entry.
  A: Yes, and projectId, taskId, runId too. Read (and assert these) from environment variables.

- Q: Should poll-message.sh support JSON output for easier parsing?
  Proposed default: Provide --json flag that wraps each entry as JSON.
  A: No. Just text answers and clear separators between items.

- Q: How are user responses distinguished from agent messages?
  Proposed default: Add author=<user|agent> field in the header line.
  A: There is type field, use FACT, QUESTION, ANSWER, USER. Explain the meaning in the prompts.

- Q: Where do post-message.sh and poll-message.sh resolve the message bus root path?
  Proposed default: Use ~/run-agent by default, override with JRUN_ROOT.
  A: Nice, yes, use JRUN_ROOT to resolve sub folders

- Q: What is the locking strategy for concurrent writes to the same MESSAGE-BUS file?
  Proposed default: Use flock around appends with retry/backoff.
  A: Agree. Since we use the same go binary, it allows more flexibility and encapsulation. I assume the idea is to use file-system-baed locking

- Q: How does poll-message.sh --wait detect new messages?
  Proposed default: Simple polling on file size/mtime every 1-2 seconds.
  A: We move to go binary, so use any technique to detect that, poling can do, hopefuly there are more effective ways. Mind we support all OS-es.

- Q: What is the canonical header format for a message entry?
  Proposed default: [ISO8601] id=<ulid> type=<type> project=<id> task=<id> run_id=<id>
  A: ACCEPTED: [ISO8601] id=<ulid> type=<FACT|QUESTION|ANSWER|USER> project=<id> task=<id> run_id=<id>. All IDs read from JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID environment variables.

- Q: How do CLI args (--project/--task) interact with JRUN_* environment variables?
  Proposed default: CLI args override env vars; if both missing, fail with error.
  A: project/task/runid are internals of the run-agent command. Environment variables must be set, error messages must never contain hints for agent to set variables. run-task must set all variables, since variables are inherited it will work.
