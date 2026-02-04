# Environment & Invocation Contract - Questions

- Q: How are paths normalized in the prompt preamble (OS-native vs POSIX style)? | Proposed default: OS-native using Go filepath.Clean. | A: TBD.
- Q: Does the agent process inherit the full parent environment or a restricted set? | Proposed default: Full inheritance in MVP (no sandbox). | A: TBD.
- Q: What is the signal handling contract between run-agent and agents (SIGTERM propagation, grace period)? | Proposed default: run-agent forwards SIGTERM to the agent process group and waits 30s before SIGKILL. | A: TBD.
- Q: Should the current date/time be injected into the prompt preamble for agents? | Proposed default: Yes, ISO-8601 UTC. | A: TBD.
