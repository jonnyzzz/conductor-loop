# Agent Protocol & Governance - Questions

- Q: Do we need an explicit git allowlist per task (e.g., TASK_GIT_ALLOWLIST.md)?
  Proposed default: Optional allowlist; if present, agents must restrict writes to those paths.
  A: No allow list. We need to allow agent to use git. There must be git repository for the tasks and metedata files (configured globally for run-agent), there are multiple project repositories there angets are working. We ask the agent to start agent processes in the projec folders (or subfolders) to localize the context and the work.

- Q: Should the protocol enforce folder ownership (each folder owned by its agent; others must delegate)?
  Proposed default: Yes; agents should delegate rather than edit folders they do not own.
  A: Yes, that is great idea, we need to enforce that on prompt basis. Agent should prefer to delegate rather than edit folders it does not own, expecially if it's not running in the project folder and the task is to change the project folder. Same applied for bigger subsystems.

- Q: Are agents allowed to read files outside the project/task folders, and what are the read boundaries?
  Proposed default: Reads limited to project root + task folder + explicit allowlist in config.
  A: No boundaries at all. Just recommendations to delegate. There is a chance that some folders can be on the other host with no direct access at all. Running agent process will still work correctly.

- Q: Should agents have read-only vs read-write permissions for project vs task folders?
  Proposed default: Read-write for task artifacts and project code; read-only for project FACT files.
  A: Give agents all possible permissions, at this stage we do not deal with any restrictions or permissions. Place that to backlog.

- Q: Who owns conflict resolution when sub-agents touch the same files?
  Proposed default: Root agent resolves conflicts; sub-agents report conflicts via MESSAGE-BUS and stop.
  A: Agreed.

- Q: Should agents be prohibited from destructive commands (rm -rf, git clean -fdx) unless explicitly approved?
  Proposed default: Disallow by default; require explicit user approval via MESSAGE-BUS.
  A: Right now we just give agents all permissions, we can add more restrictions later.

- Q: Should agents check for STOP/CANCEL requests in MESSAGE-BUS, and how should they acknowledge and exit?
  Proposed default: Check at start and before long operations; post cancellation status, update TASK_STATE.md, then exit.
  A: Yes, we prompt agent to check for STOP/CANCEL requests, it will unlikely work, we can try.

- Q: Should agents emit periodic heartbeat/progress entries when no stdout/stderr is produced?
  Proposed default: Emit STATUS heartbeat after N minutes of inactivity; N configurable.
  A: It would be great to make an agent verbosely report its progress, say to stderr, and write the final results to output.md.

- Q: Should agents be allowed to read sibling task MESSAGE-BUS files?
  Proposed default: No; only own TASK-MESSAGE-BUS and PROJECT-MESSAGE-BUS.
  A: Agent can read parent and sibling tasks via the binary, but the agent has to write access to these files. All is managed via the run-agent binary. Moreover, we track tree-like relationships between message bus messages of all types.

- Q: Should agents be prohibited from referencing JRUN_* environment variables in prompts/logs?
  Proposed default: Yes; env vars are implementation details and must not be surfaced to agents.
  A: No special setup is needed, on the onthe hand we must make sure these variables are not used by agents directly.
