# Agent Protocol & Governance - Questions

- Q: When a parent agent delegates to a sub-agent, must the parent block until the child completes, or can it exit and rely on Ralph to manage completion? 
  Proposed default: Parent blocks to preserve context and return path. 
  A: That is absolutely dependent on the agent behaviour, an paretn agent can decide to start more processes in parallel, or wait for the child to complete. We should not block that at all. Parent may even exit before a child finishes, in that case we need to wait for children to finish, given it's easy to implement.

- Q: When should the system move from root-agent polling to a dedicated message-bus polling service? 
  Proposed default: Post-MVP, only if message volume or latency requires it. 
  A: We only poll the root agent, message bus polling is fully up to the root agent and the prompt.
