# Agent Protocol & Governance - Questions

- Q: How is folder ownership assigned and recorded, and how is it passed to sub-agents?
  Proposed default: Root agent maintains OWNERSHIP.md (path -> owner run_id) and includes owned paths in sub-agent prompts.
  A: There is no OWNDERSHIP.md file. You start `run-agent` command to start sub agent, it manages all the run folders by itself, it adds "RUN_FOLDER=<path>, use this folder for prompts, task outputs, and other temp files. Put the results to $RUN_FOLDER/output.md file" to the beginning of the promots. Together wish base prompts to each runs. So 

- Q: What is the ownership handoff protocol when a parent reassigns paths or a sub-agent finishes?
  Proposed default: Parent posts HANDOFF message to TASK-MESSAGE-BUS with paths + new owner, then updates OWNERSHIP.md and TASK_STATE.md.
  A: Agents are never working with theis paths directly, no need to reassign. There is no OWNDERSHIP.md file.

- Q: Are parent agents allowed to read/monitor child TASK_STATE.md and output during execution?
  Proposed default: Parent may read child TASK_STATE.md/output for monitoring but must not write; all commands via MESSAGE-BUS.
  A: We do not help or block it. It's up the the agent to figure it out.

- Q: Should agents have resource limits (disk writes, network calls, subprocess count), and who enforces them?
  Proposed default: Optional limits defined in config; run-agent enforces pre-flight and logs violations to ISSUES.md.
  A: No. We just run the agent processes.

- Q: What guardrails apply to sensitive paths (~/.ssh, ~/.gnupg, /etc) when no sandbox is enforced?
  Proposed default: Treat as restricted; require explicit user approval via MESSAGE-BUS before read/write; log access.
  A: No isolation or sandboxing so far.

- Q: Should agents be allowed to execute arbitrary scripts in repositories, or only sanctioned tools?
  Proposed default: Allow within project boundaries after safety checks (no external network, no outside writes); prompt warns about untrusted code.
  A: Yes, there are no limits now.

- Q: Should agents emit START/STOP/STATUS entries themselves, or is that runner-only?
  Proposed default: Runner posts START/STOP; agents may post STATUS updates only.
  A: Nope, these type of messages are only allowed to emit by the run-agent command.

- Q: What is the cross-project interaction policy (can agents read/write outside their project boundary)?
  Proposed default: Agents are project-scoped; cross-project access requires explicit user approval via MESSAGE-BUS.
  A: No boundarise yet. We just ignore that problem. Not help, not support, not block.

- Q: What is the agent cancellation protocol (graceful shutdown signal, cleanup obligations)?
  A: Runner sends SIGTERM to agent pgid, waits 30s, then SIGKILL; agent must flush MESSAGE-BUS and TASK_STATE on SIGTERM.

- Q: How are agent protocol versions negotiated when runner/agent versions drift?
  Proposed default: Runner includes protocol_version in spawn; agent checks compatibility; fail fast with VERSION_MISMATCH if incompatible.
  A: No Version check yet. We assume ever backward compatible implementations based on the protocol specifcations and reviews.
