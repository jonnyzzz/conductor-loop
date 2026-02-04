# Environment & Invocation Contract - Questions

- Q: How are paths normalized in the prompt preamble (OS-native vs POSIX style)? 
  A: OS-native using Go filepath.Clean.

- Q: Does the agent process inherit the full parent environment or a restricted set? 
  Proposed default: Full inheritance in MVP (no sandbox). 
  A: Yes, inherit full environment, we set JRUN_ID and probably some other variables on top

- Q: What is the signal handling contract between run-agent and agents (SIGTERM propagation, grace period)? 
  A: run-agent forwards SIGTERM to the agent process group and waits 30s before SIGKILL.

- Q: Should the current date/time be injected into the prompt preamble for agents? 
  Proposed default: Yes, ISO-8601 UTC. 
  A: No need for it. Agent can access date time itself, it should not be needed, because our tooling will manage it itself.
